package proxy

import (
	"bufio"
	"container/list"
	"context"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	mrand "math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/logx"
)

const (
	tlsRecordHandshake = 0x16
	tlsRecordCCS       = 0x14
	tlsRecordAppData   = 0x17

	clientRandomOffset = 11
	clientRandomLen    = 32
	sessionIDOffset    = 44
	sessionIDLen       = 32

	timestampTolerance = 120 // seconds
	tlsAppDataMax      = 16384

	extSupportedVersions = 0x002b
	extKeyShare          = 0x0033
	groupX25519          = 0x001d
	groupX25519MLKEM768  = 0x11ec

	x25519MLKEM768ClientShareLen = 1216
	mlkem768CiphertextLen        = 1088

	appDataMin         = 64
	fallbackAppDataMin = 64
	fallbackAppDataMax = 160

	serverHelloDelayMin = 8 * time.Millisecond
	serverHelloDelayMax = 24 * time.Millisecond

	replayWindow    = (2*timestampTolerance + 1) * time.Second
	replayCacheSize = 65536
)

// TLS 1.3 suites in the order a typical server picks them from the client list
var cipherPreference = []uint16{0x1301, 0x1303, 0x1302}

var ccsFrame = []byte{0x14, 0x03, 0x03, 0x00, 0x01, 0x01}

// size of the first encrypted record of the masking site, 0 = unknown
var profiledAppDataLen atomic.Int32

// replayCache remembers client randoms of accepted ClientHellos; a repeat is a probe.
// It is global, so a hello accepted just before a restart can't be replayed after it
type replayCacheT struct {
	mu      sync.Mutex
	order   *list.List // of replayItem, insertion order is expiry order
	entries map[[32]byte]*list.Element
}

type replayItem struct {
	random  [32]byte
	expires time.Time
}

var replayCache = &replayCacheT{order: list.New(), entries: map[[32]byte]*list.Element{}}

func (c *replayCacheT) firstSeen(random []byte) bool {
	var key [32]byte
	copy(key[:], random)
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for front := c.order.Front(); front != nil; front = c.order.Front() {
		item := front.Value.(replayItem)
		if item.expires.After(now) {
			break
		}
		c.order.Remove(front)
		delete(c.entries, item.random)
	}
	if _, ok := c.entries[key]; ok {
		return false
	}
	if c.order.Len() >= replayCacheSize {
		front := c.order.Front()
		c.order.Remove(front)
		delete(c.entries, front.Value.(replayItem).random)
	}
	c.entries[key] = c.order.PushBack(replayItem{random: key, expires: now.Add(replayWindow)})
	return true
}

type clientHelloInfo struct {
	cipherSuites map[uint16]bool
	keyShares    map[uint16]int // named group -> key length
}

// verifyClientHello checks HMAC of the secret in client random and the timestamp
func verifyClientHello(data, secret []byte) (clientRandom, sessionID []byte, ok bool) {
	if len(data) < 43 || data[0] != tlsRecordHandshake || data[5] != 0x01 {
		return nil, nil, false
	}
	clientRandom = append([]byte(nil), data[clientRandomOffset:clientRandomOffset+clientRandomLen]...)
	zeroed := append([]byte(nil), data...)
	for i := 0; i < clientRandomLen; i++ {
		zeroed[clientRandomOffset+i] = 0
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(zeroed)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected[:28], clientRandom[:28]) {
		return nil, nil, false
	}
	var tsXor [4]byte
	for i := 0; i < 4; i++ {
		tsXor[i] = clientRandom[28+i] ^ expected[28+i]
	}
	timestamp := int64(binary.LittleEndian.Uint32(tsXor[:]))
	if diff := time.Now().Unix() - timestamp; diff > timestampTolerance || diff < -timestampTolerance {
		return nil, nil, false
	}
	sessionID = make([]byte, sessionIDLen)
	if len(data) >= sessionIDOffset+sessionIDLen && data[43] == 0x20 {
		copy(sessionID, data[sessionIDOffset:sessionIDOffset+sessionIDLen])
	}
	return clientRandom, sessionID, true
}

// parseClientHello extracts offered cipher suites and key_share groups, empty info on malformed input
func parseClientHello(data []byte) (info clientHelloInfo) {
	info = clientHelloInfo{cipherSuites: map[uint16]bool{}, keyShares: map[uint16]int{}}
	defer func() {
		if recover() != nil {
			info = clientHelloInfo{cipherSuites: map[uint16]bool{}, keyShares: map[uint16]int{}}
		}
	}()
	u16 := func(pos int) int { return int(binary.BigEndian.Uint16(data[pos : pos+2])) }

	pos := 5 + 4 + 2 + clientRandomLen
	pos += 1 + int(data[pos])
	suitesEnd := pos + 2 + u16(pos)
	pos += 2
	if suitesEnd > len(data) {
		return clientHelloInfo{cipherSuites: map[uint16]bool{}, keyShares: map[uint16]int{}}
	}
	for ; pos+1 < suitesEnd; pos += 2 {
		info.cipherSuites[uint16(u16(pos))] = true
	}
	pos = suitesEnd
	pos += 1 + int(data[pos])
	extEnd := pos + 2 + u16(pos)
	pos += 2
	if extEnd > len(data) {
		extEnd = len(data)
	}
	for pos+4 <= extEnd {
		extType, extLen := u16(pos), u16(pos+2)
		pos += 4
		if extType == extKeyShare && pos+2 <= extEnd {
			sharesEnd := pos + 2 + u16(pos)
			if sharesEnd > extEnd {
				sharesEnd = extEnd
			}
			for off := pos + 2; off+4 <= sharesEnd; {
				group, length := uint16(u16(off)), u16(off+2)
				off += 4 + length
				if off <= sharesEnd {
					if _, ok := info.keyShares[group]; !ok {
						info.keyShares[group] = length
					}
				}
			}
		}
		pos += extLen
	}
	return info
}

func selectCipherSuite(info clientHelloInfo) uint16 {
	for _, suite := range cipherPreference {
		if info.cipherSuites[suite] {
			return suite
		}
	}
	return cipherPreference[0]
}

func x25519PublicKey() []byte {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		b := make([]byte, 32)
		rand.Read(b)
		b[31] &= 0x7f
		return b
	}
	return key.PublicKey().Bytes()
}

func serverKeyShare(info clientHelloInfo) (uint16, []byte) {
	// Mirror the hybrid post-quantum group when the client offers it, as browsers and servers do;
	// the client never derives keys from it
	if info.keyShares[groupX25519MLKEM768] == x25519MLKEM768ClientShareLen {
		share := make([]byte, mlkem768CiphertextLen)
		rand.Read(share)
		return groupX25519MLKEM768, append(share, x25519PublicKey()...)
	}
	return groupX25519, x25519PublicKey()
}

func appDataLen() int {
	if n := profiledAppDataLen.Load(); n > 0 {
		return int(n)
	}
	return fallbackAppDataMin + mrand.IntN(fallbackAppDataMax-fallbackAppDataMin+1)
}

func put16(b []byte, v int) []byte { return binary.BigEndian.AppendUint16(b, uint16(v)) }

// buildServerHello returns ServerHello + CCS + one ApplicationData record.
// Telegram clients parse exactly these records and check the HMAC over them
func buildServerHello(secret, clientRandom, sessionID []byte, info clientHelloInfo) []byte {
	group, keyExchange := serverKeyShare(info)
	ext := put16(nil, extSupportedVersions)
	ext = put16(ext, 2)
	ext = put16(ext, 0x0304)
	ext = put16(ext, extKeyShare)
	ext = put16(ext, 4+len(keyExchange))
	ext = put16(ext, int(group))
	ext = put16(ext, len(keyExchange))
	ext = append(ext, keyExchange...)

	body := []byte{0x03, 0x03}
	body = append(body, make([]byte, 32)...)
	body = append(body, byte(len(sessionID)))
	body = append(body, sessionID...)
	body = put16(body, int(selectCipherSuite(info)))
	body = append(body, 0x00)
	body = put16(body, len(ext))
	body = append(body, ext...)

	message := []byte{0x02, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}
	message = append(message, body...)
	response := []byte{tlsRecordHandshake, 0x03, 0x03}
	response = put16(response, len(message))
	response = append(response, message...)
	response = append(response, ccsFrame...)

	n := appDataLen()
	response = append(response, tlsRecordAppData, 0x03, 0x03)
	response = put16(response, n)
	appData := make([]byte, n)
	rand.Read(appData)
	response = append(response, appData...)

	mac := hmac.New(sha256.New, secret)
	mac.Write(clientRandom)
	mac.Write(response)
	copy(response[11:43], mac.Sum(nil))
	return response
}

// setProfiledAppDataLen stores the record size of the masking site, 0 resets it
func setProfiledAppDataLen(n int) {
	if n <= 0 {
		profiledAppDataLen.Store(0)
		return
	}
	if n < appDataMin {
		n = appDataMin
	}
	if n > tlsAppDataMax {
		n = tlsAppDataMax
	}
	profiledAppDataLen.Store(int32(n))
}

// firstAppDataRecordLen returns the length of the first ApplicationData record in a server's
// first flight, -1 if more data is needed, 0 if the flight can't contain one
func firstAppDataRecordLen(data []byte) int {
	pos := 0
	for pos+5 <= len(data) {
		rtype := data[pos]
		recLen := int(binary.BigEndian.Uint16(data[pos+3 : pos+5]))
		if rtype == tlsRecordAppData {
			return recLen
		}
		if rtype != tlsRecordHandshake && rtype != tlsRecordCCS {
			return 0
		}
		pos += 5 + recLen
	}
	return -1
}

var errProbeDone = errors.New("probe done")

type recordingConn struct {
	net.Conn
	flight []byte
	result int
}

func (c *recordingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.flight = append(c.flight, p[:n]...)
	if r := firstAppDataRecordLen(c.flight); r >= 0 {
		c.result = r
		return n, errProbeDone
	}
	return n, err
}

// probeAppDataLen measures the first encrypted record the masking site sends after ServerHello
func probeAppDataLen(ctx context.Context, domain string) int {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(domain, "443"))
	if err != nil {
		logx.Warnf("Fake TLS profile: can't probe %s:443: %v", domain, err)
		return 0
	}
	defer raw.Close()
	rec := &recordingConn{Conn: raw, result: -1}
	conn := tls.Client(rec, &tls.Config{
		ServerName: domain, InsecureSkipVerify: true, MinVersion: tls.VersionTLS13,
		NextProtos: []string{"h2", "http/1.1"},
	})
	err = conn.HandshakeContext(ctx)
	if rec.result > 0 {
		return rec.result
	}
	logx.Warnf("Fake TLS profile: can't probe %s:443: %v", domain, err)
	return 0
}

// fakeTLSConn reads payload of TLS ApplicationData records and wraps writes into them
type fakeTLSConn struct {
	net.Conn
	r    *bufio.Reader
	left int // unread bytes of the current record
	wmu  sync.Mutex
}

func (c *fakeTLSConn) Read(p []byte) (int, error) {
	for c.left == 0 {
		var hdr [5]byte
		if _, err := io.ReadFull(c.r, hdr[:]); err != nil {
			return 0, err
		}
		recLen := int(binary.BigEndian.Uint16(hdr[3:5]))
		switch hdr[0] {
		case tlsRecordCCS:
			if _, err := c.r.Discard(recLen); err != nil {
				return 0, err
			}
		case tlsRecordAppData:
			c.left = recLen
		default:
			return 0, io.EOF
		}
	}
	if len(p) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= n
	if err == io.EOF && c.left > 0 {
		err = io.ErrUnexpectedEOF
	}
	return n, err
}

func (c *fakeTLSConn) Write(data []byte) (int, error) {
	out := make([]byte, 0, len(data)+5*(len(data)/tlsAppDataMax+1))
	for off := 0; off < len(data); off += tlsAppDataMax {
		end := off + tlsAppDataMax
		if end > len(data) {
			end = len(data)
		}
		out = append(out, tlsRecordAppData, 0x03, 0x03, byte((end-off)>>8), byte(end-off))
		out = append(out, data[off:end]...)
	}
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if _, err := c.Conn.Write(out); err != nil {
		return 0, err
	}
	return len(data), nil
}

// proxyToMaskingDomain relays a client that failed the Fake TLS check to the real site,
// so an active prober sees the site and not a proxy
func proxyToMaskingDomain(client net.Conn, r io.Reader, initial []byte, domain, label string) {
	upstream, err := net.DialTimeout("tcp", net.JoinHostPort(domain, "443"), 10*time.Second)
	if err != nil {
		logx.Warnf("[%s] masking: can't connect to %s:443: %v", label, domain, err)
		return
	}
	defer upstream.Close()
	logx.Debugf("[%s] masking -> %s:443", label, domain)
	stats.ConnectionsMasked.Add(1)
	client.SetDeadline(time.Time{})
	if len(initial) > 0 {
		if _, err := upstream.Write(initial); err != nil {
			return
		}
	}
	pipeBoth(client, r, upstream, upstream, nil, nil)
}

// pipeBoth copies a→b and b→a until one side ends, then closes both.
// aReader and bReader replace reading of a and b (buffered readers), counters may be nil
func pipeBoth(a io.ReadWriteCloser, aReader io.Reader, b io.ReadWriteCloser, bReader io.Reader, up, down *atomic.Int64) {
	if aReader == nil {
		aReader = a
	}
	if bReader == nil {
		bReader = b
	}
	done := make(chan struct{}, 2)
	cp := func(dst io.Writer, src io.Reader, counter *atomic.Int64) {
		buf := make([]byte, 32*1024)
		for {
			n, err := src.Read(buf)
			if n > 0 {
				if counter != nil {
					counter.Add(int64(n))
				}
				if _, werr := dst.Write(buf[:n]); werr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}
	go cp(b, aReader, up)
	go cp(a, bReader, down)
	<-done
	a.Close()
	b.Close()
	<-done
}
