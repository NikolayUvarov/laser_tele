// Package proxy is the MTProto proxy ported from tg-ws-proxy: Telegram clients connect with
// obfuscated2 or Fake TLS, traffic goes to Telegram DCs through WebSocket (wss://kwsN.web.telegram.org),
// Cloudflare fallbacks or plain TCP
package proxy

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	initTimeout            = 10 * time.Second
	badHandshakeDrainLimit = 64 * 1024
	badHandshakeDrainTime  = 10 * time.Second
	clientCloseTimeout     = 2 * time.Second
	listenerRetryDelay     = 5 * time.Second
)

// Runtime is the parsed configuration of the MTProto part
type Runtime struct {
	host          string
	port          int
	publicHost    string
	dcRedirects   map[int]string
	bufferSize    int
	poolSize      int
	cfProxy       bool
	h2            bool
	userDomains   []string
	workerDomains []string
	domainsURL    string
	noSecure      bool
	forceTestDC   bool
	fakeTLSDomain string
	proxyProtocol bool
}

// NewRuntime parses cfg, it must be valid
func NewRuntime(cfg *config.Config) (*Runtime, error) {
	redirects, err := config.ParseDCIP(cfg.DCIP)
	if err != nil {
		return nil, err
	}
	rc := &Runtime{
		host: cfg.Host, port: cfg.Port, publicHost: cfg.PublicHost,
		dcRedirects: redirects, bufferSize: cfg.BufKB * 1024, poolSize: cfg.PoolSize,
		cfProxy: cfg.CFProxy, h2: cfg.H2, domainsURL: cfg.CFProxyDomainsURL, noSecure: cfg.NoSecure,
		forceTestDC: cfg.ForceTestDC, fakeTLSDomain: strings.ToLower(cfg.FakeTLSDomain), proxyProtocol: cfg.ProxyProtocol,
	}
	if cfg.CFProxyUserDomainEnabled {
		rc.userDomains = normalizeDomainPool(cfg.CFProxyUserDomain)
	}
	if cfg.CFProxyWorkerEnabled {
		rc.workerDomains = normalizeDomainPool(cfg.CFProxyWorkerDomain)
	}
	return rc, nil
}

func (rc *Runtime) h2Enabled() bool { return rc.h2 && rc.cfProxy && !rc.noSecure && !rc.forceTestDC }

// Server is one run of the MTProto proxy
type Server struct {
	rc    *Runtime
	acl   *access.ACL
	users *Users

	ctx    context.Context
	cancel context.CancelFunc

	lnMu sync.Mutex
	ln   net.Listener

	pool     *wsPool
	workers  *workerPool
	h2       *h2Pool
	balancer *balancer
	cf       *cfState

	tcpMu         sync.Mutex
	tcpFailures   map[string]int
	tcpRetryAfter map[string]time.Time
	tcpConnecting map[string]bool

	connsMu sync.Mutex
	conns   map[net.Conn]struct{}
	wg      sync.WaitGroup

	started time.Time
}

// NewServer creates the proxy, Start runs it
func NewServer(rc *Runtime, acl *access.ACL, users *Users) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{rc: rc, acl: acl, users: users, ctx: ctx, cancel: cancel, balancer: newBalancer(), cf: newCFState(),
		tcpFailures: map[string]int{}, tcpRetryAfter: map[string]time.Time{}, tcpConnecting: map[string]bool{},
		conns: map[net.Conn]struct{}{}}
	s.pool = newWSPool(s)
	s.workers = newWorkerPool(s)
	if rc.h2Enabled() {
		s.h2 = newH2Pool(s)
	}
	return s
}

func (s *Server) listen() (net.Listener, error) {
	return net.Listen("tcp", net.JoinHostPort(s.rc.host, strconv.Itoa(s.rc.port)))
}

// Start opens the listening socket and starts serving
func (s *Server) Start() error {
	if len(s.rc.userDomains) > 0 {
		s.balancer.updateDomains(s.rc.userDomains)
	} else {
		s.balancer.updateDomains(DefaultCFDomains())
		if s.rc.domainsURL != "" && s.rc.cfProxy {
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.balancer.refreshLoop(s.ctx, s.rc.domainsURL)
			}()
		}
	}

	// probe before listening, so a masking domain that points to this proxy fails fast
	setProfiledAppDataLen(0)
	if s.rc.fakeTLSDomain != "" {
		if n := probeAppDataLen(s.ctx, s.rc.fakeTLSDomain); n > 0 {
			setProfiledAppDataLen(n)
			logx.Infof("Fake TLS profile: first encrypted record %d bytes", n)
		} else {
			logx.Warnf("Fake TLS profile unavailable, ServerHello uses random record sizes")
		}
	}

	ln, err := s.listen()
	if err != nil {
		s.cancel()
		s.wg.Wait()
		return err
	}
	s.lnMu.Lock()
	s.ln = ln
	s.lnMu.Unlock()
	s.started = time.Now()
	s.logBanner()

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.acceptLoop()
	}()
	go func() {
		defer s.wg.Done()
		s.statsLoop()
	}()
	s.pool.warmup()
	s.workers.warmup()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		s.lnMu.Lock()
		ln := s.ln
		s.lnMu.Unlock()
		conn, err := ln.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() || isTemporaryAccept(err) {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			logx.Warnf("Listening socket died (%v), restarting server", err)
			ln.Close()
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(listenerRetryDelay):
				}
				newLn, err := s.listen()
				if err != nil {
					logx.Errorf("Failed to restart server: %v", err)
					continue
				}
				s.lnMu.Lock()
				s.ln = newLn
				s.lnMu.Unlock()
				logx.Warnf("Server restored, listening on %s:%d", s.rc.host, s.rc.port)
				break
			}
			continue
		}
		s.connsMu.Lock()
		if s.ctx.Err() != nil {
			s.connsMu.Unlock()
			conn.Close()
			return
		}
		s.conns[conn] = struct{}{}
		s.connsMu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				s.connsMu.Lock()
				delete(s.conns, conn)
				s.connsMu.Unlock()
				conn.Close()
			}()
			s.handleClient(conn)
		}()
	}
}

func isTemporaryAccept(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "too many open files") || strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection aborted")
}

func (s *Server) statsLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			logx.Infof("stats: %s", stats.Summary())
		}
	}
}

// Stop closes the listener and all connections and waits for them
func (s *Server) Stop() {
	s.cancel()
	s.lnMu.Lock()
	if s.ln != nil {
		s.ln.Close()
	}
	s.lnMu.Unlock()
	s.connsMu.Lock()
	for c := range s.conns {
		c.Close()
	}
	s.connsMu.Unlock()
	s.pool.close()
	s.workers.close()
	if s.h2 != nil {
		s.h2.close()
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		logx.Warnf("Proxy did not stop within 5 seconds")
	}
}

// StartedAt returns when the server started listening
func (s *Server) StartedAt() time.Time { return s.started }

// CFDomains returns the current pool of Cloudflare domains
func (s *Server) CFDomains() []string { return s.balancer.list() }

// Link is a connection link for Telegram applications
type Link struct {
	Kind   string `json:"kind"` // "dd" or "ee"
	Server string `json:"server"`
	Port   int    `json:"port"`
	Secret string `json:"secret"` // with the prefix
	TG     string `json:"tg"`
	TMe    string `json:"tme"`
}

// LinkHost returns the address for links: public_host for users allowed from outside,
// then host (the address the web interface was opened with), then the outbound address
func (rc *Runtime) LinkHost(host string, external bool) string {
	if rc.publicHost != "" && (external || host == "") {
		return rc.publicHost
	}
	if host != "" {
		return host
	}
	if rc.host != "0.0.0.0" && rc.host != "::" {
		return rc.host
	}
	return OutboundIP()
}

// OutboundIP returns the local address of the default route
func OutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// UserLink returns the connection link of a user with secret (32 hex)
func (rc *Runtime) UserLink(secret string, external bool, host string) Link {
	server := rc.LinkHost(host, external)
	kind, full := "dd", "dd"+strings.ToLower(secret)
	if rc.fakeTLSDomain != "" {
		kind, full = "ee", "ee"+strings.ToLower(secret)+hex.EncodeToString([]byte(rc.fakeTLSDomain))
	}
	q := "server=" + url.QueryEscape(server) + "&port=" + strconv.Itoa(rc.port) + "&secret=" + full
	return Link{Kind: kind, Server: server, Port: rc.port, Secret: full, TG: "tg://proxy?" + q, TMe: "https://t.me/proxy?" + q}
}

func (s *Server) logBanner() {
	rc := s.rc
	logx.Infof("%s", strings.Repeat("=", 60))
	logx.Infof("  laser-mtproxy: Telegram MTProto WS Bridge Proxy")
	logx.Infof("  Listening on   %s:%d", rc.host, rc.port)
	if rc.fakeTLSDomain != "" {
		logx.Infof("  Fake TLS:      %s", rc.fakeTLSDomain)
	}
	logx.Infof("  Trusted networks: %s", strings.Join(s.acl.Networks(), ", "))
	logx.Infof("  Users:         %s", s.users.summary())
	logx.Infof("  Target DC IPs:")
	for _, dc := range config.SortedDCs(rc.dcRedirects) {
		logx.Infof("    DC%d: %s", dc, rc.dcRedirects[dc])
	}
	if rc.cfProxy {
		mode := "auto"
		if len(rc.userDomains) > 0 {
			mode = strings.Join(rc.userDomains, ", ")
		}
		logx.Infof("  CF proxy:      enabled (%s)", mode)
	}
	if len(rc.workerDomains) > 0 {
		logx.Infof("  CF worker:     enabled (%s)", strings.Join(rc.workerDomains, ", "))
	}
	if s.h2 != nil {
		logx.Infof("  CF H2 media:   enabled")
	}
	if rc.noSecure {
		logx.Infof("  No secure:     enabled (port 80 for CF proxy/worker)")
	}
	logx.Infof("%s", strings.Repeat("=", 60))
	if first := s.users.first(); first != nil {
		logx.Infof("  Connect as %s (other users: web interface, tab Users):", first.name)
		logx.Infof("    %s", rc.UserLink(hex.EncodeToString(first.secret), first.external, "").TG)
	} else {
		logx.Infof("  No enabled users: add them in the web interface")
	}
	logx.Infof("%s", strings.Repeat("=", 60))
}

// readerConn reads through a bufio.Reader that may hold already received bytes
type readerConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *readerConn) Read(p []byte) (int, error) { return c.r.Read(p) }

type clientInit struct {
	handshake []byte
	reader    io.Reader
	writer    io.Writer
	user      *userState // verified by Fake TLS, nil for plain obfuscated2
	label     string
	ip        net.IP
}

// readClientInit reads the PROXY header, Fake TLS hello or the first 64 bytes of obfuscated2
func (s *Server) readClientInit(conn net.Conn, br *bufio.Reader, label string, peerIP net.IP) (*clientInit, bool) {
	ip := peerIP
	if s.rc.proxyProtocol {
		line, err := br.ReadString('\n')
		if err != nil {
			logx.Debugf("[%s] disconnected during PROXY header", label)
			return nil, false
		}
		text := strings.TrimSpace(line)
		if parts := strings.Fields(text); len(parts) >= 6 && parts[0] == "PROXY" {
			// a PROXY header is honoured only from a trusted peer (nginx), otherwise anyone could fake a LAN address
			if s.acl.Contains(peerIP) {
				if headerIP := net.ParseIP(parts[2]); headerIP != nil {
					ip = headerIP
					label = net.JoinHostPort(parts[2], parts[4])
				}
			} else {
				logx.Warnf("[%s] PROXY header from an untrusted peer ignored", label)
			}
			logx.Debugf("[%s] PROXY protocol: %s", label, text)
		} else {
			logx.Debugf("[%s] expected PROXY header, got: %q", label, truncate(text, 60))
		}
	}

	trusted := s.acl.Contains(ip)
	candidates := s.users.candidates(trusted)
	masking := s.rc.fakeTLSDomain

	first, err := br.Peek(1)
	if err != nil {
		logx.Debugf("[%s] client disconnected before handshake", label)
		return nil, false
	}
	init := &clientInit{label: label, ip: ip, reader: br, writer: conn}

	if first[0] == tlsRecordHandshake && masking != "" {
		hdr, err := br.Peek(5)
		if err != nil {
			logx.Debugf("[%s] incomplete TLS record header", label)
			return nil, false
		}
		recordLen := int(hdr[3])<<8 | int(hdr[4])
		hello := make([]byte, 5+recordLen)
		if _, err := io.ReadFull(br, hello); err != nil {
			logx.Debugf("[%s] incomplete TLS record body", label)
			return nil, false
		}
		var clientRandom, sessionID []byte
		var user *userState
		for _, c := range candidates {
			if r, sid, ok := verifyClientHello(hello, c.secret); ok {
				clientRandom, sessionID, user = r, sid, c.user
				break
			}
		}
		replayed := user != nil && !replayCache.firstSeen(clientRandom)
		if user == nil || replayed {
			if user == nil && !trusted {
				for _, c := range s.users.localOnly() {
					if _, _, ok := verifyClientHello(hello, c.secret); ok {
						stats.ConnectionsUntrusted.Add(1)
						s.users.refusedUntrusted(c.user)
						logx.Warnf("[%s] user %s is allowed only from trusted networks -> masking", label, c.user.name)
						break
					}
				}
			}
			reason := "verify failed"
			if replayed {
				reason = "hello replayed"
			}
			logx.Debugf("[%s] Fake TLS %s (size=%d rec=%d) -> masking", label, reason, len(hello), recordLen)
			proxyToMaskingDomain(conn, br, hello, masking, label)
			return nil, false
		}
		logx.Debugf("[%s] Fake TLS handshake ok", label)
		serverHello := buildServerHello(user.secret, clientRandom, sessionID, parseClientHello(hello))
		delay := serverHelloDelayMin + time.Duration(mrand.Int64N(int64(serverHelloDelayMax-serverHelloDelayMin)))
		time.Sleep(delay)
		if _, err := conn.Write(serverHello); err != nil {
			return nil, false
		}
		tlsConn := &fakeTLSConn{Conn: conn, r: br}
		handshake := make([]byte, handshakeLen)
		if _, err := io.ReadFull(tlsConn, handshake); err != nil {
			logx.Debugf("[%s] incomplete obfs2 init inside TLS", label)
			return nil, false
		}
		init.handshake, init.reader, init.writer, init.user = handshake, tlsConn, tlsConn, user
		return init, true
	}
	if masking != "" {
		logx.Debugf("[%s] non-TLS byte 0x%02X -> HTTP redirect", label, first[0])
		fmt.Fprintf(conn, "HTTP/1.1 301 Moved Permanently\r\nLocation: https://%s/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n", masking)
		return nil, false
	}
	handshake := make([]byte, handshakeLen)
	if _, err := io.ReadFull(br, handshake); err != nil {
		logx.Debugf("[%s] client disconnected before handshake", label)
		return nil, false
	}
	init.handshake = handshake
	return init, true
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (s *Server) handleClient(conn net.Conn) {
	stats.ConnectionsTotal.Add(1)
	stats.ConnectionsActive.Add(1)
	defer stats.ConnectionsActive.Add(-1)

	label := conn.RemoteAddr().String()
	peerIP := access.HostIP(label)
	setSockOpts(conn, s.rc.bufferSize)
	conn.SetDeadline(time.Now().Add(initTimeout))
	br := bufio.NewReaderSize(conn, 16*1024)

	init, ok := s.readClientInit(conn, br, label, peerIP)
	if !ok {
		return
	}
	label = init.label
	trusted := s.acl.Contains(init.ip)

	var info handshakeInfo
	var user *userState
	candidates := s.users.candidates(trusted)
	if init.user != nil {
		candidates = []userCandidate{{init.user, init.user.secret}}
	}
	for _, c := range candidates {
		var found bool
		if info, found = tryHandshake(init.handshake, c.secret); found {
			user = c.user
			break
		}
	}
	if user == nil {
		stats.ConnectionsBad.Add(1)
		refused := false
		if !trusted {
			for _, c := range s.users.localOnly() {
				if _, ok := tryHandshake(init.handshake, c.secret); ok {
					refused = true
					stats.ConnectionsUntrusted.Add(1)
					s.users.refusedUntrusted(c.user)
					logx.Warnf("[%s] refused: user %s is allowed only from trusted networks", label, c.user.name)
					break
				}
			}
		}
		if !refused {
			logx.Warnf("[%s] bad handshake (wrong secret or proto)", label)
		}
		conn.SetDeadline(time.Now().Add(badHandshakeDrainTime))
		io.CopyN(io.Discard, init.reader, badHandshakeDrainLimit)
		return
	}
	ipStr := ""
	if init.ip != nil {
		ipStr = init.ip.String()
	}
	uc, err := s.users.acquire(user, conn, ipStr, trusted)
	if err != nil {
		stats.ConnectionsLimited.Add(1)
		logx.Warnf("[%s] user %s refused: %v", label, user.name, err)
		return
	}
	defer s.users.release(uc)
	label += " " + user.name
	conn.SetDeadline(time.Time{})

	dc := info.dc
	isTest := s.rc.forceTestDC || dc >= 10000
	if dc >= 10000 {
		logx.Infof("[%s] test DC%d -> DC%d", label, dc, dc-10000)
		dc -= 10000
	}
	proto := protoOfTag(info.tag)
	dcIdx := int16(dc)
	if info.media {
		dcIdx = -dcIdx
	}
	logx.Debugf("[%s] handshake ok: DC%d%s proto=0x%08X", label, dc, mediaTag(info.media), proto)

	relayInit := generateRelayInit(info.tag, dcIdx)
	cc := buildCryptoCtx(info.prekeyIV, info.secret, relayInit)

	setupCtx, cancelSetup := context.WithCancel(s.ctx)
	defer cancelSetup()
	cl := &clientConn{raw: conn, w: init.writer, label: label, setupCtx: setupCtx, user: user}
	cl.r = newPumpReader(init.reader, cancelSetup)
	defer cl.interrupt()

	ws := s.pool.get(dc, info.media, isTest)
	if ws == nil {
		logx.Infof("[%s] DC%d%s WS pool unavailable -> fallback", label, dc, mediaTag(info.media))
		ok := s.doFallback(cl, fallbackArgs{relayInit: relayInit, dc: dc, test: isTest, media: info.media, cc: cc,
			splitter: newMsgSplitter(proto), tag: info.tag})
		if ok {
			logx.Infof("[%s] DC%d%s fallback closed", label, dc, mediaTag(info.media))
		} else if setupCtx.Err() == nil {
			logx.Warnf("[%s] DC%d%s no fallback available", label, dc, mediaTag(info.media))
		}
		return
	}
	logx.Infof("[%s] DC%d%s -> WS pool hit", label, dc, mediaTag(info.media))
	stats.ConnectionsWS.Add(1)
	if err := ws.SendInit(relayInit); err != nil {
		logx.Warnf("[%s] DC%d%s WS init failed: %v", label, dc, mediaTag(info.media), err)
		return
	}
	s.bridgeWS(cl, ws, cc, dc, info.media, newMsgSplitter(proto))
}
