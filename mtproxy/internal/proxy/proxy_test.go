package proxy

import (
	"bufio"
	"bytes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
)

var (
	testSecret   = bytes.Repeat([]byte{0x11}, 16)
	testExternal = bytes.Repeat([]byte{0x22}, 16)
)

// makeClientInit builds the obfuscated2 init of a Telegram client and its streams
func makeClientInit(secret []byte, tag [4]byte, dcIdx int16) (init []byte, enc, dec cipher.Stream) {
	rnd := make([]byte, handshakeLen)
	rand.Read(rnd)
	rnd[0] = 0x01
	copy(rnd[protoTagPos:], tag[:])
	binary.LittleEndian.PutUint16(rnd[dcIdxPos:], uint16(dcIdx))
	encKey := sha256.Sum256(append(append([]byte(nil), rnd[8:40]...), secret...))
	enc = newCTR(encKey[:], rnd[40:56])
	rev := reversed(rnd[8:56])
	decKey := sha256.Sum256(append(append([]byte(nil), rev[:32]...), secret...))
	dec = newCTR(decKey[:], rev[32:48])
	encrypted := make([]byte, handshakeLen)
	enc.XORKeyStream(encrypted, rnd)
	init = append(append([]byte(nil), rnd[:56]...), encrypted[56:]...)
	return init, enc, dec
}

// dcSide is the Telegram side of a relay connection: plain obfuscation without secret
type dcSide struct {
	dec, enc cipher.Stream
	dc       int16
	tag      [4]byte
}

func newDCSide(t *testing.T, relayInit []byte) *dcSide {
	dec := newCTR(relayInit[8:40], relayInit[40:56])
	plain := make([]byte, handshakeLen)
	dec.XORKeyStream(plain, relayInit)
	var tag [4]byte
	copy(tag[:], plain[56:60])
	rev := reversed(relayInit[8:56])
	return &dcSide{dec: dec, enc: newCTR(rev[:32], rev[32:48]), tag: tag,
		dc: int16(binary.LittleEndian.Uint16(plain[60:62]))}
}

func TestHandshakeAndCrypto(t *testing.T) {
	init, cEnc, cDec := makeClientInit(testSecret, tagIntermediate, -2)
	if _, ok := tryHandshake(init, testExternal); ok {
		t.Fatal("handshake accepted a wrong secret")
	}
	info, ok := tryHandshake(init, testSecret)
	if !ok || info.dc != 2 || !info.media || info.tag != tagIntermediate {
		t.Fatalf("handshake: %+v ok=%v", info, ok)
	}
	relay := generateRelayInit(info.tag, -2)
	dc := newDCSide(t, relay)
	if dc.tag != tagIntermediate || dc.dc != -2 {
		t.Fatalf("relay init: tag=%x dc=%d", dc.tag, dc.dc)
	}
	cc := buildCryptoCtx(info.prekeyIV, testSecret, relay)

	msg := []byte("hello telegram, this is a test packet")
	wire := make([]byte, len(msg))
	cEnc.XORKeyStream(wire, msg)
	cc.cltDec.XORKeyStream(wire, wire)
	cc.tgEnc.XORKeyStream(wire, wire)
	dc.dec.XORKeyStream(wire, wire)
	if !bytes.Equal(wire, msg) {
		t.Fatal("client -> DC data mismatch")
	}
	reply := []byte("answer of the datacenter")
	wire = make([]byte, len(reply))
	dc.enc.XORKeyStream(wire, reply)
	cc.tgDec.XORKeyStream(wire, wire)
	cc.cltEnc.XORKeyStream(wire, wire)
	cDec.XORKeyStream(wire, wire)
	if !bytes.Equal(wire, reply) {
		t.Fatal("DC -> client data mismatch")
	}
}

func TestRelayInitAvoidsReservedStarts(t *testing.T) {
	for i := 0; i < 2000; i++ {
		r := generateRelayInit(tagAbridged, 3)
		if r[0] == 0xEF {
			t.Fatal("reserved first byte")
		}
		var start [4]byte
		copy(start[:], r[:4])
		for _, reserved := range reservedStarts {
			if start == reserved {
				t.Fatal("reserved start")
			}
		}
	}
}

func TestSplitter(t *testing.T) {
	// abridged: 1-byte header with length/4, then 0x7f with 3-byte length
	p1 := append([]byte{0x02}, bytes.Repeat([]byte{1}, 8)...)
	p2 := append([]byte{0x7f, 0x01, 0x00, 0x00}, bytes.Repeat([]byte{2}, 4)...)
	stream := append(append([]byte(nil), p1...), p2...)
	s := newMsgSplitter(protoAbridged)
	var parts [][]byte
	for _, cut := range [][2]int{{0, 3}, {3, 10}, {10, len(stream)}} {
		chunk := stream[cut[0]:cut[1]]
		parts = append(parts, s.split(chunk, chunk)...)
	}
	if len(parts) != 2 || !bytes.Equal(parts[0], p1) || !bytes.Equal(parts[1], p2) {
		t.Fatalf("abridged parts: %x", parts)
	}
	s = newMsgSplitter(protoIntermediate)
	pkt := append([]byte{8, 0, 0, 0}, bytes.Repeat([]byte{3}, 8)...)
	both := append(append([]byte(nil), pkt...), pkt...)
	parts = s.split(both[:7], both[:7])
	parts = append(parts, s.split(both[7:], both[7:])...)
	if len(parts) != 2 || !bytes.Equal(parts[0], pkt) {
		t.Fatalf("intermediate parts: %x", parts)
	}
	if tail := s.flush(); tail != nil {
		t.Fatalf("unexpected tail %x", tail)
	}
}

func TestDecodeCFDomain(t *testing.T) {
	for enc, want := range map[string]string{
		"virkgj.com":       "pclead.co.uk",
		"vmmzovy.com":      "offshor.co.uk",
		"zaftuzsftqdq.com": "nothingthere.co.uk",
		"example.org":      "example.org",
	} {
		if got := decodeCFDomain(enc); got != want {
			t.Errorf("decode(%s) = %s, want %s", enc, got, want)
		}
	}
	if len(DefaultCFDomains()) != len(cfProxyEncoded) {
		t.Fatal("default domains")
	}
}

func TestReplayCache(t *testing.T) {
	r := make([]byte, 32)
	rand.Read(r)
	if !replayCache.firstSeen(r) || replayCache.firstSeen(r) {
		t.Fatal("replay cache")
	}
}

func TestEncodeReadNativePacket(t *testing.T) {
	body := make([]byte, 40)
	rand.Read(body)
	for _, tag := range [][4]byte{tagAbridged, tagIntermediate, tagPadded} {
		frame, err := encodeReply(body, tag)
		if err != nil {
			t.Fatal(err)
		}
		key := bytes.Repeat([]byte{7}, 32)
		iv := bytes.Repeat([]byte{9}, 16)
		enc := make([]byte, len(frame))
		newCTR(key, iv).XORKeyStream(enc, frame)
		got, _, err := readNativePacket(bytes.NewReader(enc), newCTR(key, iv), tag)
		if tag == tagPadded {
			// encrypted packets of the padded protocol are cut to 24+16*k bytes
			if err != nil || !bytes.Equal(got, body[:len(got)]) {
				t.Fatalf("padded: %v", err)
			}
			continue
		}
		if err != nil || !bytes.Equal(got, body) {
			t.Fatalf("tag %x: %v", tag, err)
		}
	}
}

// --- end-to-end tests through a real listener ---

func testRuntime() *Runtime {
	return &Runtime{host: "127.0.0.1", port: 0, dcRedirects: map[int]string{}, bufferSize: 64 * 1024, poolSize: 0}
}

// testUsers: "lan" works only from trusted networks, "ext" from everywhere
func testUsers() []config.User {
	return []config.User{
		{Name: "lan", Secret: hex.EncodeToString(testSecret), MaxConnections: 100},
		{Name: "ext", Secret: hex.EncodeToString(testExternal), MaxConnections: 100, AllowExternal: true},
	}
}

func startServer(t *testing.T, rc *Runtime, trusted []string) (*Server, string) {
	users := NewUsers("")
	users.Update(testUsers())
	return startServerWithUsers(t, rc, trusted, users)
}

func startServerWithUsers(t *testing.T, rc *Runtime, trusted []string, users *Users) (*Server, string) {
	t.Helper()
	acl, err := access.New(trusted)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	rc.port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	s := NewServer(rc, acl, users)
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s, net.JoinHostPort("127.0.0.1", strconv.Itoa(rc.port))
}

// echoDC serves relay connections: decrypts data and sends it back encrypted
func echoDC(t *testing.T, conn net.Conn, readInit func() ([]byte, error), read func([]byte) (int, error), write func([]byte) error) {
	defer conn.Close()
	init, err := readInit()
	if err != nil {
		return
	}
	dc := newDCSide(t, init)
	buf := make([]byte, 64*1024)
	for {
		n, err := read(buf)
		if n > 0 {
			data := append([]byte(nil), buf[:n]...)
			dc.dec.XORKeyStream(data, data)
			dc.enc.XORKeyStream(data, data)
			if write(data) != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func startTCPDC(t *testing.T) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go echoDC(t, conn, func() ([]byte, error) {
				b := make([]byte, handshakeLen)
				_, err := io.ReadFull(conn, b)
				return b, err
			}, conn.Read, func(b []byte) error { _, err := conn.Write(b); return err })
		}
	}()
	return ln.Addr().String()
}

func roundTrip(t *testing.T, conn io.ReadWriter, enc, dec cipher.Stream) {
	t.Helper()
	packet := append([]byte{0x04}, bytes.Repeat([]byte("ABCD"), 4)...)
	wire := make([]byte, len(packet))
	enc.XORKeyStream(wire, packet)
	if _, err := conn.Write(wire); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(packet))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal("read answer:", err)
	}
	dec.XORKeyStream(got, got)
	if !bytes.Equal(got, packet) {
		t.Fatalf("echo mismatch: %x", got)
	}
}

func useTCPDC(t *testing.T, addr string) {
	host, port, _ := net.SplitHostPort(addr)
	oldPort, oldIP := tcpFallbackPort, DCDefaultIPs[2]
	tcpFallbackPort, DCDefaultIPs[2] = port, host
	t.Cleanup(func() { tcpFallbackPort, DCDefaultIPs[2] = oldPort, oldIP })
}

func dialClient(t *testing.T, addr string) net.Conn {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestEndToEndTCPFallback(t *testing.T) {
	useTCPDC(t, startTCPDC(t))
	_, addr := startServer(t, testRuntime(), []string{"loopback"})
	conn := dialClient(t, addr)
	init, enc, dec := makeClientInit(testSecret, tagAbridged, 2)
	conn.Write(init)
	roundTrip(t, conn, enc, dec)
	roundTrip(t, conn, enc, dec)
}

func TestUntrustedNeedsExternalSecret(t *testing.T) {
	useTCPDC(t, startTCPDC(t))
	// nothing is trusted: only the external secret works
	_, addr := startServer(t, testRuntime(), nil)

	conn := dialClient(t, addr)
	init, _, _ := makeClientInit(testSecret, tagAbridged, 2)
	conn.Write(init)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("trusted secret accepted from an untrusted address")
	}
	if GlobalStats().ConnectionsUntrusted.Load() == 0 {
		t.Fatal("untrusted refusal not counted")
	}

	conn2 := dialClient(t, addr)
	init, enc, dec := makeClientInit(testExternal, tagAbridged, 2)
	conn2.Write(init)
	roundTrip(t, conn2, enc, dec)
}

// --- WebSocket pool with a fake TLS datacenter ---

func selfSignedTLS(t *testing.T, names ...string) (*tls.Config, *x509.CertPool) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: names[0]}, DNSNames: names,
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}, pool
}

func readServerFrame(br *bufio.Reader) (byte, []byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return 0, nil, err
	}
	length := uint64(hdr[1] & 0x7f)
	switch length {
	case 126:
		var b [2]byte
		io.ReadFull(br, b[:])
		length = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		io.ReadFull(br, b[:])
		length = binary.BigEndian.Uint64(b[:])
	}
	var mask [4]byte
	if hdr[1]&0x80 == 0 {
		return 0, nil, fmt.Errorf("client frame not masked")
	}
	io.ReadFull(br, mask[:])
	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return 0, nil, err
	}
	maskBytes(payload, payload, mask)
	return hdr[0] & 0x0f, payload, nil
}

func startWSDC(t *testing.T) (port string, frames chan int) {
	cfg, pool := selfSignedTLS(t, "kws2.web.telegram.org", "kws2-1.web.telegram.org")
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	frames = make(chan int, 100)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				br := bufio.NewReader(conn)
				req, err := http.ReadRequest(br)
				if err != nil || req.URL.Path != wsPath || req.Header.Get("Upgrade") != "websocket" {
					conn.Close()
					return
				}
				io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
				var pending []byte
				readMsg := func() ([]byte, error) {
					for {
						op, payload, err := readServerFrame(br)
						if err != nil {
							return nil, err
						}
						if op == wsOpBinary {
							return payload, nil
						}
						if op == wsOpClose {
							return nil, io.EOF
						}
					}
				}
				echoDC(t, conn, readMsg, func(b []byte) (int, error) {
					if len(pending) == 0 {
						msg, err := readMsg()
						if err != nil {
							return 0, err
						}
						frames <- len(msg)
						pending = msg
					}
					n := copy(b, pending)
					pending = pending[n:]
					return n, nil
				}, func(b []byte) error {
					frame := []byte{0x82}
					switch {
					case len(b) < 126:
						frame = append(frame, byte(len(b)))
					default:
						frame = append(frame, 126, byte(len(b)>>8), byte(len(b)))
					}
					_, err := conn.Write(append(frame, b...))
					return err
				})
			}()
		}
	}()
	_, port, _ = net.SplitHostPort(ln.Addr().String())
	wsRootCAs = pool
	t.Cleanup(func() { wsRootCAs = nil })
	return port, frames
}

func TestEndToEndWebSocketPool(t *testing.T) {
	port, frames := startWSDC(t)
	wsPortOverride = port
	t.Cleanup(func() { wsPortOverride = "" })
	rc := testRuntime()
	rc.dcRedirects = map[int]string{2: "127.0.0.1"}
	rc.poolSize = 2
	s, addr := startServer(t, rc, []string{"loopback"})

	deadline := time.Now().Add(5 * time.Second)
	for {
		s.pool.mu.Lock()
		n := len(s.pool.idle[poolKey{2, false, false}])
		s.pool.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("WS pool not filled")
		}
		time.Sleep(20 * time.Millisecond)
	}

	hits := GlobalStats().PoolHits.Load()
	conn := dialClient(t, addr)
	init, enc, dec := makeClientInit(testSecret, tagAbridged, 2)
	conn.Write(init)
	// two packets in one write must arrive as two WebSocket frames
	p1 := append([]byte{0x02}, bytes.Repeat([]byte{5}, 8)...)
	p2 := append([]byte{0x01}, bytes.Repeat([]byte{6}, 4)...)
	both := append(append([]byte(nil), p1...), p2...)
	wire := make([]byte, len(both))
	enc.XORKeyStream(wire, both)
	conn.Write(wire)
	got := make([]byte, len(both))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	dec.XORKeyStream(got, got)
	if !bytes.Equal(got, both) {
		t.Fatal("echo mismatch through WS")
	}
	if GlobalStats().PoolHits.Load() == hits {
		t.Fatal("connection didn't use the WS pool")
	}
	sizes := []int{<-frames, <-frames}
	if sizes[0] != len(p1) || sizes[1] != len(p2) {
		t.Fatalf("frames %v, want %d and %d", sizes, len(p1), len(p2))
	}
	roundTrip(t, conn, enc, dec)
}

// --- Fake TLS ---

func buildClientHello(secret []byte, ts uint32) []byte {
	body := []byte{0x03, 0x03}
	body = append(body, make([]byte, 32)...)
	sid := make([]byte, 32)
	rand.Read(sid)
	body = append(body, 32)
	body = append(body, sid...)
	body = append(body, 0x00, 0x04, 0x13, 0x01, 0x13, 0x02)
	body = append(body, 0x01, 0x00)
	key := make([]byte, 32)
	rand.Read(key)
	ext := []byte{0x00, 0x33}
	ext = binary.BigEndian.AppendUint16(ext, uint16(2+4+32))
	ext = binary.BigEndian.AppendUint16(ext, uint16(4+32))
	ext = append(ext, 0x00, 0x1d, 0x00, 32)
	ext = append(ext, key...)
	ext = append(ext, 0x00, 0x2b, 0x00, 0x03, 0x02, 0x03, 0x04)
	body = binary.BigEndian.AppendUint16(body, uint16(len(ext)))
	body = append(body, ext...)
	hs := []byte{0x01, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}
	hs = append(hs, body...)
	record := []byte{0x16, 0x03, 0x01}
	record = binary.BigEndian.AppendUint16(record, uint16(len(hs)))
	record = append(record, hs...)
	mac := hmac.New(sha256.New, secret)
	mac.Write(record)
	sum := mac.Sum(nil)
	var tsb [4]byte
	binary.LittleEndian.PutUint32(tsb[:], ts)
	for i := 0; i < 4; i++ {
		sum[28+i] ^= tsb[i]
	}
	copy(record[11:43], sum[:32])
	return record
}

func readRecord(r io.Reader) (byte, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, err
	}
	body := make([]byte, binary.BigEndian.Uint16(hdr[3:5]))
	_, err := io.ReadFull(r, body)
	return hdr[0], append(hdr[:], body...), err
}

func TestFakeTLSHelloRoundTrip(t *testing.T) {
	hello := buildClientHello(testSecret, uint32(time.Now().Unix()))
	random, sid, ok := verifyClientHello(hello, testSecret)
	if !ok || len(sid) != 32 {
		t.Fatal("hello not verified")
	}
	if _, _, ok := verifyClientHello(hello, testExternal); ok {
		t.Fatal("verified with wrong secret")
	}
	if _, _, ok := verifyClientHello(buildClientHello(testSecret, uint32(time.Now().Unix()-1000)), testSecret); ok {
		t.Fatal("old timestamp accepted")
	}
	info := parseClientHello(hello)
	if !info.cipherSuites[0x1301] || info.keyShares[groupX25519] != 32 {
		t.Fatalf("parse: %+v", info)
	}
	resp := buildServerHello(testSecret, random, sid, info)
	zeroed := append([]byte(nil), resp...)
	copy(zeroed[11:43], make([]byte, 32))
	mac := hmac.New(sha256.New, testSecret)
	mac.Write(random)
	mac.Write(zeroed)
	if !bytes.Equal(mac.Sum(nil), resp[11:43]) {
		t.Fatal("server hello HMAC")
	}
}

func TestEndToEndFakeTLS(t *testing.T) {
	useTCPDC(t, startTCPDC(t))
	rc := testRuntime()
	rc.fakeTLSDomain = "localhost"
	_, addr := startServer(t, rc, []string{"loopback"})

	conn := dialClient(t, addr)
	hello := buildClientHello(testSecret, uint32(time.Now().Unix()))
	conn.Write(hello)
	for _, want := range []byte{tlsRecordHandshake, tlsRecordCCS, tlsRecordAppData} {
		typ, _, err := readRecord(conn)
		if err != nil || typ != want {
			t.Fatalf("record %x: got %x %v", want, typ, err)
		}
	}
	tlsConn := &fakeTLSConn{Conn: conn, r: bufio.NewReader(conn)}
	init, enc, dec := makeClientInit(testSecret, tagAbridged, 2)
	tlsConn.Write(init)
	roundTrip(t, tlsConn, enc, dec)

	// the same hello again is a replay and must not get a ServerHello
	conn2 := dialClient(t, addr)
	conn2.Write(hello)
	conn2.SetReadDeadline(time.Now().Add(time.Second))
	if typ, _, err := readRecord(conn2); err == nil && typ == tlsRecordHandshake {
		t.Fatal("replayed hello answered")
	}
}

// --- users: limits, disabling, changes and counters ---

func waitClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("connection is still open")
	} else if isTimeout(err) {
		t.Fatal("connection was not closed")
	}
}

func userReport(u *Users, name string) UserReport {
	for _, r := range u.Report().Users {
		if r.Name == name {
			return r
		}
	}
	return UserReport{}
}

func TestUserLimitAndChanges(t *testing.T) {
	useTCPDC(t, startTCPDC(t))
	users := NewUsers(filepath.Join(t.TempDir(), "stats.json"))
	list := testUsers()
	list[0].MaxConnections = 1
	users.Update(list)
	_, addr := startServerWithUsers(t, testRuntime(), []string{"loopback"}, users)

	first := dialClient(t, addr)
	init, enc, dec := makeClientInit(testSecret, tagAbridged, 2)
	first.Write(init)
	roundTrip(t, first, enc, dec)

	// the second simultaneous connection of "lan" is over the limit
	second := dialClient(t, addr)
	init2, _, _ := makeClientInit(testSecret, tagAbridged, 2)
	second.Write(init2)
	waitClosed(t, second)
	r := userReport(users, "lan")
	if r.Active != 1 || r.RefusedLimit != 1 || r.Connections != 1 || r.Peak != 1 {
		t.Fatalf("report: %+v", r)
	}
	if r.BytesUp == 0 || r.BytesDown == 0 || r.LastIP != "127.0.0.1" {
		t.Fatalf("traffic and address not counted: %+v", r)
	}

	// another user isn't affected by the limit of "lan"
	other := dialClient(t, addr)
	init3, enc3, dec3 := makeClientInit(testExternal, tagAbridged, 2)
	other.Write(init3)
	roundTrip(t, other, enc3, dec3)

	// disabling closes the open connection and refuses new ones
	list[0].MaxConnections = 0
	users.Update(list)
	waitClosed(t, first)
	third := dialClient(t, addr)
	init4, _, _ := makeClientInit(testSecret, tagAbridged, 2)
	third.Write(init4)
	waitClosed(t, third)
	if r := userReport(users, "lan"); r.RefusedDisabled != 1 {
		t.Fatalf("disabled refusal not counted: %+v", r)
	}

	// a new secret closes connections with the old one
	list[1].Secret = hex.EncodeToString(bytes.Repeat([]byte{0x33}, 16))
	users.Update(list)
	waitClosed(t, other)

	// counters survive in the statistics file
	if err := users.Save(); err != nil {
		t.Fatal(err)
	}
	again := NewUsers(users.statsPath)
	again.Update(list)
	if r := userReport(again, "lan"); r.Connections != 1 || r.RefusedLimit != 1 || r.BytesUp == 0 {
		t.Fatalf("saved counters: %+v", r)
	}
}
