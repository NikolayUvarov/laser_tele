package proxy

// Gateway for Bot API clients. Bots work over HTTPS, not MTProto, so they can't use the MTProto proxy;
// the gateway is an HTTP CONNECT and SOCKS5 proxy on one port that opens tunnels only to the allowed
// hosts (api.telegram.org:443). Trusted addresses need no password

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	gatewayHandshakeTimeout = 15 * time.Second
	gatewayDialTimeout      = 10 * time.Second
	authCacheTTL            = 10 * time.Minute
	authFailDelay           = time.Second
)

type hostPattern struct {
	host     string
	wildcard bool
	port     string
}

func (p hostPattern) match(host, port string) bool {
	h := strings.TrimSuffix(strings.ToLower(host), ".")
	if p.port != "*" && p.port != port {
		return false
	}
	if p.wildcard {
		return h == p.host || strings.HasSuffix(h, "."+p.host)
	}
	return h == p.host
}

// Gateway is the HTTP CONNECT / SOCKS5 proxy for Bot API
type Gateway struct {
	host, port    string
	acl           *access.ACL
	username      string
	passwordHash  string
	allowed       []hostPattern
	viaWorker     bool
	workerDomains []string
	noSecure      bool
	fallbackIPs   map[string][]string // host -> other addresses of the same service

	routeMu  sync.Mutex
	lastGood map[string]string // host:port -> address that worked last time

	ctx    context.Context
	cancel context.CancelFunc
	ln     net.Listener
	wg     sync.WaitGroup

	connsMu sync.Mutex
	conns   map[net.Conn]struct{}

	authMu    sync.Mutex
	authCache map[[32]byte]time.Time
}

// NewGateway creates the gateway from the configuration
func NewGateway(cfg *config.Config, acl *access.ACL) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	g := &Gateway{
		host: cfg.Gateway.Host, port: strconv.Itoa(cfg.Gateway.Port), acl: acl,
		username: cfg.Access.Username, passwordHash: cfg.Access.PasswordHash,
		viaWorker: cfg.Gateway.ViaCFWorker, noSecure: cfg.NoSecure,
		ctx: ctx, cancel: cancel, conns: map[net.Conn]struct{}{}, authCache: map[[32]byte]time.Time{},
		fallbackIPs: map[string][]string{}, lastGood: map[string]string{},
	}
	for host, ips := range cfg.Gateway.FallbackIPs {
		g.fallbackIPs[strings.ToLower(host)] = append([]string(nil), ips...)
	}
	if cfg.CFProxyWorkerEnabled {
		g.workerDomains = normalizeDomainPool(cfg.CFProxyWorkerDomain)
	}
	for _, pattern := range cfg.Gateway.AllowedHosts {
		host, port, err := config.ParseHostPattern(pattern)
		if err != nil {
			continue
		}
		wildcard := strings.HasPrefix(host, "*.")
		g.allowed = append(g.allowed, hostPattern{host: strings.TrimPrefix(host, "*."), wildcard: wildcard, port: port})
	}
	return g
}

// Start opens the listening socket
func (g *Gateway) Start() error {
	ln, err := net.Listen("tcp", net.JoinHostPort(g.host, g.port))
	if err != nil {
		g.cancel()
		return err
	}
	g.ln = ln
	logx.Infof("Bot API gateway (HTTP CONNECT + SOCKS5) listening on %s:%s, allowed: %s", g.host, g.port, g.allowedList())
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if g.ctx.Err() != nil {
					return
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			g.connsMu.Lock()
			g.conns[conn] = struct{}{}
			g.connsMu.Unlock()
			g.wg.Add(1)
			go func() {
				defer g.wg.Done()
				defer func() {
					g.connsMu.Lock()
					delete(g.conns, conn)
					g.connsMu.Unlock()
					conn.Close()
				}()
				g.handle(conn)
			}()
		}
	}()
	return nil
}

func (g *Gateway) allowedList() string {
	var parts []string
	for _, p := range g.allowed {
		h := p.host
		if p.wildcard {
			h = "*." + h
		}
		parts = append(parts, net.JoinHostPort(h, p.port))
	}
	return strings.Join(parts, ", ")
}

// Stop closes the gateway and its tunnels
func (g *Gateway) Stop() {
	g.cancel()
	if g.ln != nil {
		g.ln.Close()
	}
	g.connsMu.Lock()
	for c := range g.conns {
		c.Close()
	}
	g.connsMu.Unlock()
	done := make(chan struct{})
	go func() {
		g.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}

func (g *Gateway) isAllowed(host, port string) bool {
	for _, p := range g.allowed {
		if p.match(host, port) {
			return true
		}
	}
	return false
}

// checkAuth verifies credentials of an untrusted client, successful ones are cached
func (g *Gateway) checkAuth(user, pass string) bool {
	if g.passwordHash == "" {
		return false
	}
	key := sha256.Sum256([]byte(user + "\x00" + pass))
	now := time.Now()
	g.authMu.Lock()
	if until, ok := g.authCache[key]; ok && now.Before(until) {
		g.authMu.Unlock()
		return true
	}
	g.authMu.Unlock()
	ok := subtle.ConstantTimeCompare([]byte(user), []byte(g.username)) == 1 && config.CheckPassword(g.passwordHash, pass)
	if ok {
		g.authMu.Lock()
		for k, until := range g.authCache {
			if now.After(until) {
				delete(g.authCache, k)
			}
		}
		g.authCache[key] = now.Add(authCacheTTL)
		g.authMu.Unlock()
	}
	return ok
}

func (g *Gateway) handle(conn net.Conn) {
	stats.GatewayTotal.Add(1)
	stats.GatewayActive.Add(1)
	defer stats.GatewayActive.Add(-1)
	label := conn.RemoteAddr().String()
	trusted := g.acl.ContainsAddr(label)
	conn.SetDeadline(time.Now().Add(gatewayHandshakeTimeout))
	br := bufio.NewReader(conn)
	first, err := br.Peek(1)
	if err != nil {
		return
	}
	if first[0] == 0x05 {
		g.handleSOCKS5(conn, br, label, trusted)
	} else {
		g.handleHTTP(conn, br, label, trusted)
	}
}

func (g *Gateway) deny(label, reason string) {
	stats.GatewayDenied.Add(1)
	logx.Warnf("[%s] gateway: denied: %s", label, reason)
}

func (g *Gateway) handleHTTP(conn net.Conn, br *bufio.Reader, label string, trusted bool) {
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	reply := func(status int, extra string) {
		fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\n%sContent-Length: 0\r\nConnection: close\r\n\r\n", status, http.StatusText(status), extra)
	}
	if req.Method != http.MethodConnect {
		reply(http.StatusMethodNotAllowed, "")
		return
	}
	authed := trusted
	if !authed {
		if g.passwordHash == "" {
			g.deny(label, "untrusted address and no password configured")
			reply(http.StatusForbidden, "")
			return
		}
		user, pass, ok := parseProxyAuth(req.Header.Get("Proxy-Authorization"))
		if !ok || !g.checkAuth(user, pass) {
			g.deny(label, "wrong or missing proxy credentials")
			time.Sleep(authFailDelay)
			reply(http.StatusProxyAuthRequired, "Proxy-Authenticate: Basic realm=\"laser-mtproxy\"\r\n")
			return
		}
	}
	host, port, err := net.SplitHostPort(req.Host)
	if err != nil {
		reply(http.StatusBadRequest, "")
		return
	}
	if !g.isAllowed(host, port) {
		g.deny(label, "destination "+req.Host+" is not allowed")
		reply(http.StatusForbidden, "")
		return
	}
	upstream, via, err := g.dial(host, port)
	if err != nil {
		stats.GatewayFailures.Add(1)
		logx.Warnf("[%s] gateway: can't connect to %s: %v", label, req.Host, err)
		reply(http.StatusBadGateway, "")
		return
	}
	defer upstream.Close()
	if _, err := io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		return
	}
	g.tunnel(conn, br, upstream, label, req.Host, via, trusted)
}

func parseProxyAuth(header string) (user, pass string, ok bool) {
	scheme, value, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Basic") {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return "", "", false
	}
	user, pass, ok = strings.Cut(string(decoded), ":")
	return user, pass, ok
}

const (
	socksNoAuth       = 0x00
	socksUserPass     = 0x02
	socksNoAcceptable = 0xFF

	socksRepSuccess      = 0x00
	socksRepFailure      = 0x01
	socksRepNotAllowed   = 0x02
	socksRepHostUnreach  = 0x04
	socksRepCmdNotSupp   = 0x07
	socksRepAddrNotSuppd = 0x08
)

func (g *Gateway) handleSOCKS5(conn net.Conn, br *bufio.Reader, label string, trusted bool) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return
	}
	methods := make([]byte, hdr[1])
	if _, err := io.ReadFull(br, methods); err != nil {
		return
	}
	has := func(m byte) bool {
		for _, x := range methods {
			if x == m {
				return true
			}
		}
		return false
	}
	switch {
	case trusted && has(socksNoAuth):
		conn.Write([]byte{0x05, socksNoAuth})
	case has(socksUserPass) && (trusted || g.passwordHash != ""):
		conn.Write([]byte{0x05, socksUserPass})
		// RFC 1929
		var v [2]byte
		if _, err := io.ReadFull(br, v[:]); err != nil {
			return
		}
		user := make([]byte, v[1])
		if _, err := io.ReadFull(br, user); err != nil {
			return
		}
		var plen [1]byte
		if _, err := io.ReadFull(br, plen[:]); err != nil {
			return
		}
		pass := make([]byte, plen[0])
		if _, err := io.ReadFull(br, pass); err != nil {
			return
		}
		if !trusted && !g.checkAuth(string(user), string(pass)) {
			g.deny(label, "wrong SOCKS5 credentials")
			time.Sleep(authFailDelay)
			conn.Write([]byte{0x01, 0x01})
			return
		}
		conn.Write([]byte{0x01, 0x00})
	default:
		reason := "SOCKS5 client offers no password authentication"
		if g.passwordHash == "" {
			reason = "untrusted address and no password configured"
		}
		g.deny(label, reason)
		conn.Write([]byte{0x05, socksNoAcceptable})
		return
	}

	var req [4]byte
	if _, err := io.ReadFull(br, req[:]); err != nil {
		return
	}
	socksReply := func(code byte) {
		conn.Write([]byte{0x05, code, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	}
	if req[0] != 0x05 {
		return
	}
	var host string
	switch req[3] {
	case 0x01:
		b := make([]byte, 4)
		if _, err := io.ReadFull(br, b); err != nil {
			return
		}
		host = net.IP(b).String()
	case 0x03:
		var l [1]byte
		if _, err := io.ReadFull(br, l[:]); err != nil {
			return
		}
		b := make([]byte, l[0])
		if _, err := io.ReadFull(br, b); err != nil {
			return
		}
		host = string(b)
	case 0x04:
		b := make([]byte, 16)
		if _, err := io.ReadFull(br, b); err != nil {
			return
		}
		host = net.IP(b).String()
	default:
		socksReply(socksRepAddrNotSuppd)
		return
	}
	var p [2]byte
	if _, err := io.ReadFull(br, p[:]); err != nil {
		return
	}
	port := strconv.Itoa(int(binary.BigEndian.Uint16(p[:])))
	if req[1] != 0x01 {
		socksReply(socksRepCmdNotSupp)
		return
	}
	target := net.JoinHostPort(host, port)
	if !g.isAllowed(host, port) {
		g.deny(label, "destination "+target+" is not allowed")
		socksReply(socksRepNotAllowed)
		return
	}
	upstream, via, err := g.dial(host, port)
	if err != nil {
		stats.GatewayFailures.Add(1)
		logx.Warnf("[%s] gateway: can't connect to %s: %v", label, target, err)
		socksReply(socksRepHostUnreach)
		return
	}
	defer upstream.Close()
	socksReply(socksRepSuccess)
	g.tunnel(conn, br, upstream, label, target, via, trusted)
}

func (g *Gateway) tunnel(conn net.Conn, br *bufio.Reader, upstream io.ReadWriteCloser, label, target, via string, trusted bool) {
	conn.SetDeadline(time.Time{})
	who := "password"
	if trusted {
		who = "trusted"
	}
	logx.Infof("[%s] gateway: tunnel to %s via %s (%s)", label, target, via, who)
	started := time.Now()
	var up, down atomic.Int64
	pipeBoth(conn, br, upstream, nil, &up, &down)
	stats.GatewayBytesUp.Add(up.Load())
	stats.GatewayBytesDn.Add(down.Load())
	logx.Debugf("[%s] gateway: tunnel to %s closed: ^%s v%s in %.1fs", label, target,
		HumanBytes(up.Load()), HumanBytes(down.Load()), time.Since(started).Seconds())
}

// dialRace connects to the first reachable address. The next address starts when the previous
// one failed or after stagger, so a blocked address doesn't delay the others for the whole timeout
func dialRace(ctx context.Context, addrs []string, stagger time.Duration) (net.Conn, string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		conn net.Conn
		addr string
		err  error
	}
	results := make(chan result, len(addrs))
	next, pending := 0, 0
	launch := func() {
		if next < len(addrs) {
			addr := addrs[next]
			next++
			pending++
			go func() {
				conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
				results <- result{conn, addr, err}
			}()
		}
	}
	launch()
	timer := time.NewTimer(stagger)
	defer timer.Stop()
	var firstErr error
	for pending > 0 || next < len(addrs) {
		select {
		case r := <-results:
			pending--
			if r.err == nil {
				cancel()
				go func(left int) { // close connections that won later
					for ; left > 0; left-- {
						if late := <-results; late.conn != nil {
							late.conn.Close()
						}
					}
				}(pending)
				return r.conn, r.addr, nil
			}
			if firstErr == nil {
				firstErr = r.err
			}
			if pending == 0 {
				launch()
			}
		case <-timer.C:
			launch()
			timer.Reset(stagger)
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}
	return nil, "", firstErr
}

// dial connects by DNS and the fallback addresses of the host, then through a CF worker
// when allowed and the destination port is 443
func (g *Gateway) dial(host, port string) (io.ReadWriteCloser, string, error) {
	ctx, cancel := context.WithTimeout(g.ctx, gatewayDialTimeout)
	defer cancel()
	key := net.JoinHostPort(strings.ToLower(host), port)
	addrs := []string{net.JoinHostPort(host, port)}
	for _, ip := range g.fallbackIPs[strings.ToLower(strings.TrimSuffix(host, "."))] {
		addrs = append(addrs, net.JoinHostPort(ip, port))
	}
	g.routeMu.Lock()
	if good, ok := g.lastGood[key]; ok {
		for i, a := range addrs {
			if a == good {
				addrs[0], addrs[i] = addrs[i], addrs[0]
			}
		}
	}
	g.routeMu.Unlock()
	conn, addr, err := dialRace(ctx, addrs, 1500*time.Millisecond)
	if err == nil {
		g.routeMu.Lock()
		g.lastGood[key] = addr
		g.routeMu.Unlock()
		via := "direct"
		if addr != net.JoinHostPort(host, port) {
			via = "fallback IP " + addr
		}
		return conn, via, nil
	}
	if !g.viaWorker || len(g.workerDomains) == 0 || port != "443" || g.ctx.Err() != nil {
		return nil, "", err
	}
	domains := append([]string(nil), g.workerDomains...)
	mrand.Shuffle(len(domains), func(i, j int) { domains[i], domains[j] = domains[j], domains[i] })
	lastErr := err
	for _, domain := range domains {
		ws, werr := wsConnect(g.ctx, wsOptions{host: domain, domain: domain, path: workerPath(host, 0),
			secure: !g.noSecure, timeout: gatewayDialTimeout})
		if werr != nil {
			lastErr = fmt.Errorf("direct: %v; CF worker %s: %w", err, domain, werr)
			continue
		}
		stats.GatewayViaCF.Add(1)
		return &wsStream{ws: ws}, "CF worker " + domain, nil
	}
	return nil, "", lastErr
}

// wsStream turns a WebSocket of a CF worker into a byte stream
type wsStream struct {
	ws  *wsConn
	buf []byte
}

func (s *wsStream) Read(p []byte) (int, error) {
	for len(s.buf) == 0 {
		data, err := s.ws.Recv()
		if err != nil {
			return 0, err
		}
		s.buf = data
	}
	n := copy(p, s.buf)
	s.buf = s.buf[n:]
	return n, nil
}

func (s *wsStream) Write(p []byte) (int, error) {
	if err := s.ws.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *wsStream) Close() error {
	s.ws.Close()
	return nil
}
