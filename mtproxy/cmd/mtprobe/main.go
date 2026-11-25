// mtprobe checks an MTProto proxy like a Telegram client does: it connects with the secret,
// sends req_pq_multi to the chosen datacenters and waits for resPQ from Telegram.
//
//	go run ./cmd/mtprobe -addr 192.168.1.10:1443 -secret dd0123...
package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const resPQ = 0x05162463

func main() {
	addr := flag.String("addr", "127.0.0.1:1443", "proxy address host:port")
	secretArg := flag.String("secret", "", "secret: 32 hex, dd+32 hex or ee+32 hex+domain hex (as in the tg:// link)")
	dcList := flag.String("dc", "1,2,3,4,5", "datacenters, negative numbers are media DCs (for example -2)")
	timeout := flag.Duration("timeout", 15*time.Second, "timeout of one check")
	flag.Parse()

	secret, domain, err := parseSecret(*secretArg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "secret:", err)
		os.Exit(2)
	}
	failed := 0
	for _, item := range strings.Split(*dcList, ",") {
		dc, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil || dc == 0 {
			fmt.Fprintln(os.Stderr, "wrong DC:", item)
			os.Exit(2)
		}
		started := time.Now()
		err = probe(*addr, secret, domain, int16(dc), *timeout)
		name := fmt.Sprintf("DC%d", abs(dc))
		if dc < 0 {
			name += " media"
		}
		if err != nil {
			failed++
			fmt.Printf("%-10s FAIL  %v\n", name, err)
		} else {
			fmt.Printf("%-10s OK    resPQ in %d ms\n", name, time.Since(started).Milliseconds())
		}
	}
	if failed > 0 {
		os.Exit(1)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func parseSecret(s string) (secret []byte, domain string, err error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case len(s) == 32:
	case strings.HasPrefix(s, "dd") && len(s) == 34:
		s = s[2:]
	case strings.HasPrefix(s, "ee") && len(s) > 34:
		d, err := hex.DecodeString(s[34:])
		if err != nil {
			return nil, "", err
		}
		domain = string(d)
		s = s[2:34]
	default:
		return nil, "", errors.New("expected 32 hex characters with optional dd/ee prefix")
	}
	secret, err = hex.DecodeString(s)
	return secret, domain, err
}

func newCTR(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key)
	return cipher.NewCTR(block, iv)
}

func reversed(b []byte) []byte {
	r := make([]byte, len(b))
	for i := range b {
		r[len(b)-1-i] = b[i]
	}
	return r
}

func probe(addr string, secret []byte, domain string, dcIdx int16, timeout time.Duration) error {
	raw, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer raw.Close()
	raw.SetDeadline(time.Now().Add(timeout))
	var rw io.ReadWriter = raw
	if domain != "" {
		if rw, err = fakeTLS(raw, secret, domain); err != nil {
			return fmt.Errorf("fake TLS: %w", err)
		}
	}

	// obfuscated2 init with the intermediate protocol
	init := make([]byte, 64)
	for {
		rand.Read(init)
		if init[0] != 0xef && string(init[:4]) != "GET " && string(init[:4]) != "POST" && string(init[:4]) != "HEAD" &&
			binary.LittleEndian.Uint32(init[4:8]) != 0 && init[0] != 0x16 {
			break
		}
	}
	binary.LittleEndian.PutUint32(init[56:60], 0xeeeeeeee)
	binary.LittleEndian.PutUint16(init[60:62], uint16(dcIdx))
	encKey := sha256.Sum256(append(append([]byte(nil), init[8:40]...), secret...))
	enc := newCTR(encKey[:], init[40:56])
	rev := reversed(init[8:56])
	decKey := sha256.Sum256(append(append([]byte(nil), rev[:32]...), secret...))
	dec := newCTR(decKey[:], rev[32:48])
	encrypted := make([]byte, 64)
	enc.XORKeyStream(encrypted, init)
	copy(init[56:], encrypted[56:])

	// req_pq_multi#be7e8ef1 nonce:int128 as an unencrypted message
	msg := make([]byte, 0, 44)
	msg = binary.LittleEndian.AppendUint32(msg, 40)
	msg = append(msg, make([]byte, 8)...) // auth_key_id = 0
	msgID := uint64(time.Now().Unix())<<32 | uint64(time.Now().Nanosecond())&^3
	msg = binary.LittleEndian.AppendUint64(msg, msgID)
	msg = binary.LittleEndian.AppendUint32(msg, 20)
	msg = binary.LittleEndian.AppendUint32(msg, 0xbe7e8ef1)
	nonce := make([]byte, 16)
	rand.Read(nonce)
	msg = append(msg, nonce...)
	enc.XORKeyStream(msg, msg)
	if _, err := rw.Write(append(init, msg...)); err != nil {
		return err
	}

	var lenBuf [4]byte
	if _, err := io.ReadFull(rw, lenBuf[:]); err != nil {
		return fmt.Errorf("no answer: %w", err)
	}
	dec.XORKeyStream(lenBuf[:], lenBuf[:])
	n := binary.LittleEndian.Uint32(lenBuf[:])
	if n == 4 {
		var code [4]byte
		io.ReadFull(rw, code[:])
		dec.XORKeyStream(code[:], code[:])
		return fmt.Errorf("transport error %d", int32(binary.LittleEndian.Uint32(code[:])))
	}
	if n < 40 || n > 1<<20 {
		return fmt.Errorf("unexpected answer length %d", n)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(rw, body); err != nil {
		return err
	}
	dec.XORKeyStream(body, body)
	if c := binary.LittleEndian.Uint32(body[20:24]); c != resPQ {
		return fmt.Errorf("unexpected constructor 0x%08x", c)
	}
	if string(body[24:40]) != string(nonce) {
		return errors.New("nonce mismatch")
	}
	return nil
}

// fakeTLS performs the Fake TLS handshake of ee-secrets and returns the stream of ApplicationData records
func fakeTLS(conn net.Conn, secret []byte, domain string) (io.ReadWriter, error) {
	body := []byte{0x03, 0x03}
	body = append(body, make([]byte, 32)...)
	sid := make([]byte, 32)
	rand.Read(sid)
	body = append(append(body, 32), sid...)
	body = append(body, 0x00, 0x06, 0x13, 0x01, 0x13, 0x02, 0x13, 0x03, 0x01, 0x00)
	key := make([]byte, 32)
	rand.Read(key)
	var ext []byte
	sni := append([]byte{0x00}, binary.BigEndian.AppendUint16(nil, uint16(len(domain)))...)
	sni = append(sni, domain...)
	ext = append(ext, 0x00, 0x00)
	ext = binary.BigEndian.AppendUint16(ext, uint16(len(sni)+2))
	ext = binary.BigEndian.AppendUint16(ext, uint16(len(sni)))
	ext = append(ext, sni...)
	ext = append(ext, 0x00, 0x33, 0x00, 0x26, 0x00, 0x24, 0x00, 0x1d, 0x00, 0x20)
	ext = append(ext, key...)
	ext = append(ext, 0x00, 0x2b, 0x00, 0x03, 0x02, 0x03, 0x04)
	body = binary.BigEndian.AppendUint16(body, uint16(len(ext)))
	body = append(body, ext...)
	hs := append([]byte{0x01, byte(len(body) >> 16), byte(len(body) >> 8), byte(len(body))}, body...)
	hello := append([]byte{0x16, 0x03, 0x01}, binary.BigEndian.AppendUint16(nil, uint16(len(hs)))...)
	hello = append(hello, hs...)
	mac := hmac.New(sha256.New, secret)
	mac.Write(hello)
	sum := mac.Sum(nil)
	var ts [4]byte
	binary.LittleEndian.PutUint32(ts[:], uint32(time.Now().Unix()))
	for i := 0; i < 4; i++ {
		sum[28+i] ^= ts[i]
	}
	copy(hello[11:43], sum)
	clientRandom := append([]byte(nil), sum...)
	if _, err := conn.Write(hello); err != nil {
		return nil, err
	}
	br := bufio.NewReader(conn)
	var response []byte
	for i := 0; i < 3; i++ {
		var hdr [5]byte
		if _, err := io.ReadFull(br, hdr[:]); err != nil {
			return nil, fmt.Errorf("server hello: %w", err)
		}
		rec := make([]byte, binary.BigEndian.Uint16(hdr[3:5]))
		if _, err := io.ReadFull(br, rec); err != nil {
			return nil, err
		}
		response = append(append(response, hdr[:]...), rec...)
	}
	got := append([]byte(nil), response[11:43]...)
	copy(response[11:43], make([]byte, 32))
	check := hmac.New(sha256.New, secret)
	check.Write(clientRandom)
	check.Write(response)
	if !hmac.Equal(check.Sum(nil), got) {
		return nil, errors.New("server hello HMAC mismatch: wrong secret or not a proxy")
	}
	return &tlsStream{conn: conn, r: br}, nil
}

type tlsStream struct {
	conn net.Conn
	r    *bufio.Reader
	left int
}

func (s *tlsStream) Read(p []byte) (int, error) {
	for s.left == 0 {
		var hdr [5]byte
		if _, err := io.ReadFull(s.r, hdr[:]); err != nil {
			return 0, err
		}
		n := int(binary.BigEndian.Uint16(hdr[3:5]))
		if hdr[0] == 0x14 {
			s.r.Discard(n)
			continue
		}
		s.left = n
	}
	if len(p) > s.left {
		p = p[:s.left]
	}
	n, err := s.r.Read(p)
	s.left -= n
	return n, err
}

func (s *tlsStream) Write(p []byte) (int, error) {
	rec := append([]byte{0x17, 0x03, 0x03}, binary.BigEndian.AppendUint16(nil, uint16(len(p)))...)
	_, err := s.conn.Write(append(rec, p...))
	return len(p), err
}
