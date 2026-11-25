package proxy

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	wsOpCont   = 0x0
	wsOpText   = 0x1
	wsOpBinary = 0x2
	wsOpClose  = 0x8
	wsOpPing   = 0x9
	wsOpPong   = 0xA

	wsMaxMessage   = 16 * 1024 * 1024
	wsCloseTimeout = 2 * time.Second
)

// wsHandshakeError is a non-101 answer to the WebSocket upgrade
type wsHandshakeError struct {
	StatusCode int
	StatusLine string
	Headers    map[string]string
}

func (e *wsHandshakeError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.StatusLine)
}

func (e *wsHandshakeError) isRedirect() bool {
	switch e.StatusCode {
	case 301, 302, 303, 307, 308:
		return true
	}
	return false
}

// Test hooks: port of WebSocket connections and root certificates (nil = system)
var (
	wsPortOverride string
	wsRootCAs      *x509.CertPool
)

// wsConn is a minimal WebSocket client for binary messages
type wsConn struct {
	conn net.Conn
	br   *bufio.Reader

	wmu    sync.Mutex
	closed atomic.Bool
	frag   []byte

	// watcher of an idle connection in a pool
	watchDone chan struct{}
	watchErr  error
}

type wsOptions struct {
	host    string // address to connect to
	domain  string // Host header
	path    string
	sni     string // SNI if differs from domain: domain fronting without certificate check
	secure  bool   // TLS on port 443, otherwise plain on port 80
	timeout time.Duration
	bufSize int
}

func setSockOpts(conn net.Conn, bufSize int) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.SetNoDelay(true)
		if bufSize > 0 {
			tcp.SetReadBuffer(bufSize)
			tcp.SetWriteBuffer(bufSize)
		}
	}
}

// wsConnect opens a WebSocket connection, ctx cancels the setup
func wsConnect(ctx context.Context, o wsOptions) (*wsConn, error) {
	if o.timeout <= 0 {
		o.timeout = 10 * time.Second
	}
	port := "80"
	if o.secure {
		port = "443"
	}
	if wsPortOverride != "" {
		port = wsPortOverride
	}
	deadline := time.Now().Add(o.timeout)
	dialTimeout := o.timeout
	if dialTimeout > 10*time.Second {
		dialTimeout = 10 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	stage := "tcp"
	raw, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort(o.host, port))
	if err != nil {
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) {
			stage = "dns"
		}
		logx.Debugf("WS setup failed: host=%s port=%s stage=%s error=%v", o.host, port, stage, err)
		return nil, err
	}
	setSockOpts(raw, o.bufSize)
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()
	raw.SetDeadline(deadline)

	conn := raw
	if o.secure {
		stage = "tls"
		cfg := &tls.Config{ServerName: o.domain, NextProtos: []string{"http/1.1"}, RootCAs: wsRootCAs}
		if o.sni != "" {
			cfg.ServerName = o.sni
			cfg.InsecureSkipVerify = true
		}
		tlsConn := tls.Client(raw, cfg)
		if err := tlsConn.HandshakeContext(dialCtx); err != nil {
			raw.Close()
			logx.Debugf("WS setup failed: host=%s port=%s stage=%s error=%v", o.host, port, stage, err)
			return nil, ctxErr(ctx, err)
		}
		conn = tlsConn
	}

	stage = "ws_upgrade"
	keyBytes := make([]byte, 16)
	rand.Read(keyBytes)
	req := "GET " + o.path + " HTTP/1.1\r\n" +
		"Host: " + o.domain + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + base64.StdEncoding.EncodeToString(keyBytes) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Protocol: binary\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return nil, ctxErr(ctx, err)
	}

	br := bufio.NewReaderSize(conn, 64*1024)
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if err != nil && line == "" {
			if len(lines) == 0 {
				conn.Close()
				if err == io.EOF {
					return nil, &wsHandshakeError{StatusLine: "empty response"}
				}
				logx.Debugf("WS setup failed: host=%s port=%s stage=%s error=%v", o.host, port, stage, err)
				return nil, ctxErr(ctx, err)
			}
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		conn.Close()
		return nil, &wsHandshakeError{StatusLine: "empty response"}
	}
	parts := strings.SplitN(lines[0], " ", 3)
	status := 0
	if len(parts) >= 2 {
		status, _ = strconv.Atoi(parts[1])
	}
	if status != 101 {
		conn.Close()
		headers := map[string]string{}
		for _, h := range lines[1:] {
			if k, v, ok := strings.Cut(h, ":"); ok {
				headers[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
			}
		}
		return nil, &wsHandshakeError{StatusCode: status, StatusLine: lines[0], Headers: headers}
	}
	conn.SetDeadline(time.Time{})
	return &wsConn{conn: conn, br: br}, nil
}

func ctxErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func maskBytes(dst, src []byte, mask [4]byte) {
	m64 := uint64(binary.LittleEndian.Uint32(mask[:]))
	m64 |= m64 << 32
	i := 0
	for ; i+8 <= len(src); i += 8 {
		binary.LittleEndian.PutUint64(dst[i:], binary.LittleEndian.Uint64(src[i:])^m64)
	}
	for ; i < len(src); i++ {
		dst[i] = src[i] ^ mask[i%4]
	}
}

func appendFrame(out []byte, opcode byte, data []byte) []byte {
	n := len(data)
	out = append(out, 0x80|opcode)
	switch {
	case n < 126:
		out = append(out, 0x80|byte(n))
	case n < 65536:
		out = append(out, 0x80|126, byte(n>>8), byte(n))
	default:
		out = append(out, 0x80|127)
		out = binary.BigEndian.AppendUint64(out, uint64(n))
	}
	var mask [4]byte
	rand.Read(mask[:])
	out = append(out, mask[:]...)
	start := len(out)
	out = append(out, make([]byte, n)...)
	maskBytes(out[start:], data, mask)
	return out
}

func (w *wsConn) writeRaw(frame []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()
	_, err := w.conn.Write(frame)
	return err
}

// Send sends one binary message
func (w *wsConn) Send(data []byte) error {
	if w.closed.Load() {
		return errors.New("WebSocket closed")
	}
	return w.writeRaw(appendFrame(make([]byte, 0, len(data)+14), wsOpBinary, data))
}

// SendBatch sends several binary messages with one write
func (w *wsConn) SendBatch(parts [][]byte) error {
	if w.closed.Load() {
		return errors.New("WebSocket closed")
	}
	size := 0
	for _, p := range parts {
		size += len(p) + 14
	}
	out := make([]byte, 0, size)
	for _, p := range parts {
		out = appendFrame(out, wsOpBinary, p)
	}
	return w.writeRaw(out)
}

// SendInit sends the init packet of the relay with a timeout
func (w *wsConn) SendInit(data []byte) error {
	w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := w.Send(data)
	w.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		w.closed.Store(true)
		w.conn.Close()
	}
	return err
}

func (w *wsConn) readFrame() (opcode byte, payload []byte, fin bool, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(w.br, hdr[:]); err != nil {
		return
	}
	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0F
	length := uint64(hdr[1] & 0x7F)
	switch length {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(w.br, b[:]); err != nil {
			return
		}
		length = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(w.br, b[:]); err != nil {
			return
		}
		length = binary.BigEndian.Uint64(b[:])
	}
	if length > wsMaxMessage {
		err = fmt.Errorf("WS frame too large: %d bytes", length)
		return
	}
	var mask [4]byte
	masked := hdr[1]&0x80 != 0
	if masked {
		if _, err = io.ReadFull(w.br, mask[:]); err != nil {
			return
		}
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(w.br, payload); err != nil {
		return
	}
	if masked {
		maskBytes(payload, payload, mask)
	}
	return
}

// Recv returns the next binary message, io.EOF when the server closed the connection
func (w *wsConn) Recv() ([]byte, error) {
	for !w.closed.Load() {
		opcode, payload, fin, err := w.readFrame()
		if err != nil {
			return nil, err
		}
		switch opcode {
		case wsOpClose:
			w.closed.Store(true)
			code := 0
			if len(payload) >= 2 {
				code = int(binary.BigEndian.Uint16(payload[:2]))
			}
			logx.Debugf("WS OP_CLOSE from upstream: code=%d", code)
			reply := payload
			if len(reply) > 2 {
				reply = reply[:2]
			}
			w.conn.SetWriteDeadline(time.Now().Add(wsCloseTimeout))
			w.writeRaw(appendFrame(nil, wsOpClose, reply))
			return nil, io.EOF
		case wsOpPing:
			w.writeRaw(appendFrame(nil, wsOpPong, payload))
		case wsOpPong:
		case wsOpCont, wsOpText, wsOpBinary:
			if fin && len(w.frag) == 0 {
				return payload, nil
			}
			w.frag = append(w.frag, payload...)
			if len(w.frag) > wsMaxMessage {
				return nil, fmt.Errorf("WS message too large: %d bytes", len(w.frag))
			}
			if !fin {
				continue
			}
			message := w.frag
			w.frag = nil
			return message, nil
		}
	}
	return nil, io.EOF
}

// Close sends a close frame and closes the connection
func (w *wsConn) Close() {
	if w.closed.CompareAndSwap(false, true) {
		w.conn.SetWriteDeadline(time.Now().Add(wsCloseTimeout))
		w.writeRaw(appendFrame(nil, wsOpClose, nil))
	}
	w.conn.Close()
}

// startWatch notices when the server closes an idle connection, like asyncio does by itself
func (w *wsConn) startWatch() {
	w.watchDone = make(chan struct{})
	go func() {
		_, err := w.br.Peek(1)
		w.watchErr = err
		close(w.watchDone)
	}()
}

// watchFailed reports that the idle connection was closed or broken
func (w *wsConn) watchFailed() bool {
	if w.closed.Load() {
		return true
	}
	if w.watchDone == nil {
		return false
	}
	select {
	case <-w.watchDone:
		return w.watchErr != nil && !isTimeout(w.watchErr)
	default:
		return false
	}
}

// stopWatch stops the watcher before the connection is used, it returns false if the connection is dead
func (w *wsConn) stopWatch() bool {
	if w.watchDone == nil {
		return !w.closed.Load()
	}
	w.conn.SetReadDeadline(time.Now())
	<-w.watchDone
	w.conn.SetReadDeadline(time.Time{})
	w.watchDone = nil
	if w.closed.Load() {
		return false
	}
	return w.watchErr == nil || isTimeout(w.watchErr)
}
