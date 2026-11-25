package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	tcpBackoffInitial = 30 * time.Second
	tcpBackoffMax     = time.Hour
)

var errInterrupted = errors.New("interrupted")

// tcpFallbackPort is the port of Telegram DCs for the TCP fallback (changed by tests)
var tcpFallbackPort = "443"

// pumpReader reads the client in background. It lets the proxy notice that the client left
// while an upstream connection is being set up, like client_setup() of tg-ws-proxy
type pumpReader struct {
	ch       chan []byte
	err      error
	stop     chan struct{}
	stopOnce sync.Once
	buf      []byte
}

func newPumpReader(src io.Reader, onEOF func()) *pumpReader {
	p := &pumpReader{ch: make(chan []byte, 1), stop: make(chan struct{})}
	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case p.ch <- chunk:
				case <-p.stop:
					return
				}
			}
			if err != nil {
				p.err = err
				close(p.ch)
				onEOF()
				return
			}
		}
	}()
	return p
}

func (p *pumpReader) Read(b []byte) (int, error) {
	if len(p.buf) > 0 {
		n := copy(b, p.buf)
		p.buf = p.buf[n:]
		return n, nil
	}
	select {
	case chunk, ok := <-p.ch:
		if !ok {
			if p.err == nil {
				return 0, io.EOF
			}
			return 0, p.err
		}
		n := copy(b, chunk)
		p.buf = chunk[n:]
		return n, nil
	case <-p.stop:
		return 0, errInterrupted
	}
}

func (p *pumpReader) interrupt() { p.stopOnce.Do(func() { close(p.stop) }) }

// clientConn is an accepted client after the handshake
type clientConn struct {
	raw      net.Conn
	r        *pumpReader
	w        io.Writer // raw connection or Fake TLS records
	label    string
	setupCtx context.Context // done when the client disconnected
	user     *userState
}

func (c *clientConn) interrupt() { c.r.interrupt() }

// countUp and countDown add traffic to the process and to the user
func (c *clientConn) countUp(n int) {
	stats.BytesUp.Add(int64(n))
	if c.user != nil {
		c.user.up.Add(int64(n))
	}
}

func (c *clientConn) countDown(n int) {
	stats.BytesDown.Add(int64(n))
	if c.user != nil {
		c.user.down.Add(int64(n))
	}
}

func closeReason(prefix string, err error) string {
	switch {
	case err == nil || errors.Is(err, errInterrupted):
		return ""
	case errors.Is(err, io.EOF):
		if prefix == "upstream" {
			return "upstream: ws_close"
		}
		return ""
	default:
		msg := err.Error()
		if i := strings.LastIndex(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		return prefix + ": " + msg
	}
}

// bridgeWS is client TCP <-> WebSocket to Telegram with re-encryption
func (s *Server) bridgeWS(cl *clientConn, ws *wsConn, cc *cryptoCtx, dc int, media bool, splitter *msgSplitter) {
	dcTag := "DC?"
	if dc > 0 {
		dcTag = "DC" + itoa(dc)
		if media {
			dcTag += "m"
		}
	}
	var upBytes, downBytes, upPackets, downPackets int64
	started := time.Now()
	reasons := make(chan string, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // client -> Telegram
		defer wg.Done()
		buf := make([]byte, 64*1024)
		for {
			n, err := cl.r.Read(buf)
			if n > 0 {
				cl.countUp(n)
				upBytes += int64(n)
				upPackets++
				plain := make([]byte, n)
				cc.cltDec.XORKeyStream(plain, buf[:n])
				enc := make([]byte, n)
				cc.tgEnc.XORKeyStream(enc, plain)
				var serr error
				if splitter != nil {
					parts := splitter.split(enc, plain)
					switch {
					case len(parts) == 1:
						serr = ws.Send(parts[0])
					case len(parts) > 1:
						serr = ws.SendBatch(parts)
					}
				} else {
					serr = ws.Send(enc)
				}
				if serr != nil {
					reasons <- closeReason("upstream", serr)
					return
				}
			}
			if err != nil {
				if splitter != nil && errors.Is(err, io.EOF) {
					if tail := splitter.flush(); tail != nil {
						ws.Send(tail)
					}
				}
				reasons <- closeReason("client", err)
				return
			}
		}
	}()

	go func() { // Telegram -> client
		defer wg.Done()
		for {
			data, err := ws.Recv()
			if err != nil {
				reasons <- closeReason("upstream", err)
				return
			}
			n := len(data)
			cl.countDown(n)
			downBytes += int64(n)
			downPackets++
			cc.tgDec.XORKeyStream(data, data)
			cc.cltEnc.XORKeyStream(data, data)
			if _, err := cl.w.Write(data); err != nil {
				reasons <- closeReason("client", err)
				return
			}
		}
	}()

	reason := <-reasons
	ws.Close()
	cl.interrupt()
	cl.raw.Close()
	wg.Wait()
	if reason == "" {
		reason = "normal"
	}
	logx.Infof("[%s] %s WS session closed (%s): ^%s (%d pkts) v%s (%d pkts) in %.1fs",
		cl.label, dcTag, reason, HumanBytes(upBytes), upPackets, HumanBytes(downBytes), downPackets,
		time.Since(started).Seconds())
}

// bridgeTCP is client TCP <-> TCP to Telegram with re-encryption
func (s *Server) bridgeTCP(cl *clientConn, remote net.Conn, cc *cryptoCtx) {
	var wg sync.WaitGroup
	wg.Add(2)
	done := make(chan struct{}, 2)
	go func() {
		defer wg.Done()
		buf := make([]byte, 64*1024)
		for {
			n, err := cl.r.Read(buf)
			if n > 0 {
				cl.countUp(n)
				data := buf[:n]
				cc.cltDec.XORKeyStream(data, data)
				cc.tgEnc.XORKeyStream(data, data)
				if _, werr := remote.Write(data); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()
	go func() {
		defer wg.Done()
		buf := make([]byte, 64*1024)
		for {
			n, err := remote.Read(buf)
			if n > 0 {
				cl.countDown(n)
				data := buf[:n]
				cc.tgDec.XORKeyStream(data, data)
				cc.cltEnc.XORKeyStream(data, data)
				if _, werr := cl.w.Write(data); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()
	<-done
	remote.Close()
	cl.interrupt()
	cl.raw.Close()
	wg.Wait()
}

type fallbackArgs struct {
	relayInit []byte
	dc        int
	test      bool
	media     bool
	cc        *cryptoCtx
	splitter  *msgSplitter
	tag       [4]byte
}

// doFallback tries the CF worker, Cloudflare domains (HTTP/2 for media) and direct TCP
func (s *Server) doFallback(cl *clientConn, a fallbackArgs) bool {
	ipTable := DCDefaultIPs
	if a.test {
		ipTable = DCTestIPs
	}
	dst, hasDst := ipTable[a.dc]
	useCF := s.rc.cfProxy && !a.test
	var methods []string
	if len(s.rc.workerDomains) > 0 && hasDst {
		methods = append(methods, "cf_worker")
	}
	if useCF {
		methods = append(methods, "cf")
	}
	if hasDst {
		methods = append(methods, "tcp")
	}
	for _, method := range methods {
		if cl.setupCtx.Err() != nil {
			return false
		}
		switch method {
		case "cf_worker":
			if s.workerFallback(cl, a, dst) {
				return true
			}
		case "cf":
			if a.media && s.h2 != nil && s.cf.prefersTLS() {
				if ch := s.h2.open(cl.setupCtx, a.dc, cl.label); ch != nil {
					stats.ConnectionsH2.Add(1)
					stats.ConnectionsCFProxy.Add(1)
					s.bridgeH2(cl, ch, a.cc, a.tag)
					return true
				}
			}
			if s.cfProxyFallback(cl, a) {
				return true
			}
		case "tcp":
			if s.tcpFallback(cl, dst, tcpFallbackPort, a.relayInit, a.cc) {
				return true
			}
		}
	}
	return false
}

func (s *Server) workerFallback(cl *clientConn, a fallbackArgs, dst string) bool {
	var ws *wsConn
	var domain string
	if !a.test {
		ws, domain = s.workers.get(a.dc, dst)
	}
	if ws != nil {
		logx.Infof("[%s] DC%d%s -> CF worker pool hit via %s for %s", cl.label, a.dc, mediaTag(a.media), domain, dst)
	} else {
		ws, domain = s.workers.connectOne(cl.setupCtx, a.dc, dst, 10*time.Second)
		if ws == nil {
			return false
		}
		logx.Infof("[%s] DC%d%s -> CF worker %s for %s", cl.label, a.dc, mediaTag(a.media), domain, dst)
	}
	stats.ConnectionsCFProxy.Add(1)
	if err := ws.SendInit(a.relayInit); err != nil {
		return false
	}
	s.bridgeWS(cl, ws, a.cc, a.dc, a.media, nil)
	return true
}

func (s *Server) cfProxyFallback(cl *clientConn, a fallbackArgs) bool {
	logx.Infof("[%s] DC%d%s -> trying CF proxy", cl.label, a.dc, mediaTag(a.media))
	started := time.Now()
	var candidates []cfCandidate
	for _, base := range s.balancer.domainsForDC(a.dc) {
		candidates = append(candidates, cfCandidate{name: base, host: "kws" + itoa(a.dc) + "." + base})
	}
	ws, chosen, err := s.connectCFDomains(cl.setupCtx, candidates, cl.label, !s.rc.noSecure, wsPath, 10*time.Second)
	if err != nil {
		return false
	}
	logx.Debugf("[%s] WS CONNECT DC%d%s route=cf setup_ms=%d", cl.label, a.dc, mediaTag(a.media), time.Since(started).Milliseconds())
	if chosen != "" && s.balancer.updateDomainForDC(a.dc, chosen) {
		logx.Infof("[%s] Switched active CF domain", cl.label)
	}
	stats.ConnectionsCFProxy.Add(1)
	if err := ws.SendInit(a.relayInit); err != nil {
		return false
	}
	s.bridgeWS(cl, ws, a.cc, a.dc, a.media, a.splitter)
	return true
}

func (s *Server) tcpFallback(cl *clientConn, dst, port string, relayInit []byte, cc *cryptoCtx) bool {
	key := net.JoinHostPort(dst, port)
	s.tcpMu.Lock()
	if s.tcpConnecting[key] || time.Now().Before(s.tcpRetryAfter[key]) {
		s.tcpMu.Unlock()
		logx.Debugf("[%s] TCP fallback to %s skipped (backoff or connecting)", cl.label, key)
		return false
	}
	s.tcpConnecting[key] = true
	s.tcpMu.Unlock()

	logx.Infof("[%s] TCP fallback to %s", cl.label, key)
	ctx, cancel := context.WithTimeout(cl.setupCtx, 10*time.Second)
	remote, err := (&net.Dialer{}).DialContext(ctx, "tcp", key)
	cancel()
	if err == nil {
		setSockOpts(remote, s.rc.bufferSize)
		remote.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err = remote.Write(relayInit)
		remote.SetWriteDeadline(time.Time{})
		if err != nil {
			remote.Close()
		}
	}

	s.tcpMu.Lock()
	delete(s.tcpConnecting, key)
	if err != nil {
		if cl.setupCtx.Err() != nil {
			s.tcpMu.Unlock()
			return false
		}
		s.tcpFailures[key]++
		failures := s.tcpFailures[key]
		if failures > 8 {
			failures = 8
		}
		delay := tcpBackoffInitial * time.Duration(1<<(failures-1))
		if delay > tcpBackoffMax {
			delay = tcpBackoffMax
		}
		s.tcpRetryAfter[key] = time.Now().Add(delay)
		s.tcpMu.Unlock()
		logx.Warnf("[%s] TCP fallback to %s failed: %v; retry in %.0fs", cl.label, key, err, delay.Seconds())
		return false
	}
	delete(s.tcpFailures, key)
	delete(s.tcpRetryAfter, key)
	s.tcpMu.Unlock()

	stats.ConnectionsTCPFallback.Add(1)
	s.bridgeTCP(cl, remote, cc)
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
