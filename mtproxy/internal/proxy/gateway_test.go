package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/access"
	"github.com/NikolayUvarov/laser_tele/v2/mtproxy/internal/config"
)

func startEcho(t *testing.T) (host, port string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); io.Copy(c, c) }()
		}
	}()
	host, port, _ = net.SplitHostPort(ln.Addr().String())
	return
}

func startGateway(t *testing.T, trusted []string, password string, allowed ...string) string {
	cfg := config.Default()
	cfg.Gateway.Host = "127.0.0.1"
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	cfg.Gateway.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	cfg.Gateway.AllowedHosts = allowed
	cfg.Access.Username = "bot"
	if password != "" {
		cfg.Access.PasswordHash = config.HashPassword(password)
	}
	acl, err := access.New(trusted)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGateway(cfg, acl)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(g.Stop)
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Gateway.Port))
}

func httpConnect(t *testing.T, gw, target, auth string) (net.Conn, *bufio.Reader, int) {
	conn, err := net.DialTimeout("tcp", gw, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	req := "CONNECT " + target + " HTTP/1.1\r\nHost: " + target + "\r\n"
	if auth != "" {
		req += "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(auth)) + "\r\n"
	}
	io.WriteString(conn, req+"\r\n")
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	return conn, br, resp.StatusCode
}

func echoCheck(t *testing.T, conn net.Conn, br *bufio.Reader) {
	t.Helper()
	conn.Write([]byte("ping-through-gateway"))
	got := make([]byte, len("ping-through-gateway"))
	if _, err := io.ReadFull(br, got); err != nil || string(got) != "ping-through-gateway" {
		t.Fatalf("echo: %q %v", got, err)
	}
}

func TestGatewayHTTPTrusted(t *testing.T) {
	host, port := startEcho(t)
	target := net.JoinHostPort(host, port)
	gw := startGateway(t, []string{"loopback"}, "", target)
	conn, br, status := httpConnect(t, gw, target, "")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	echoCheck(t, conn, br)

	if _, _, status := httpConnect(t, gw, "127.0.0.1:1", ""); status != http.StatusForbidden {
		t.Fatalf("not allowed destination: status %d", status)
	}
}

func TestGatewayHTTPPassword(t *testing.T) {
	host, port := startEcho(t)
	target := net.JoinHostPort(host, port)

	noPassword := startGateway(t, nil, "", target)
	if _, _, status := httpConnect(t, noPassword, target, ""); status != http.StatusForbidden {
		t.Fatalf("untrusted without configured password: status %d", status)
	}

	gw := startGateway(t, nil, "s3cret-pass", target)
	if _, _, status := httpConnect(t, gw, target, ""); status != http.StatusProxyAuthRequired {
		t.Fatalf("without credentials: status %d", status)
	}
	if _, _, status := httpConnect(t, gw, target, "bot:wrong"); status != http.StatusProxyAuthRequired {
		t.Fatalf("wrong password: status %d", status)
	}
	conn, br, status := httpConnect(t, gw, target, "bot:s3cret-pass")
	if status != 200 {
		t.Fatalf("with password: status %d", status)
	}
	echoCheck(t, conn, br)
}

func socksConnect(t *testing.T, gw, host string, port int, user, pass string) (net.Conn, byte) {
	conn, err := net.DialTimeout("tcp", gw, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	if user == "" {
		conn.Write([]byte{5, 1, 0})
	} else {
		conn.Write([]byte{5, 2, 0, 2})
	}
	var choice [2]byte
	if _, err := io.ReadFull(conn, choice[:]); err != nil {
		t.Fatal(err)
	}
	if choice[1] == 0xFF {
		return conn, 0xFF
	}
	if choice[1] == 2 {
		msg := append([]byte{1, byte(len(user))}, user...)
		msg = append(append(msg, byte(len(pass))), pass...)
		conn.Write(msg)
		var st [2]byte
		io.ReadFull(conn, st[:])
		if st[1] != 0 {
			return conn, 0xFE
		}
	}
	req := []byte{5, 1, 0, 3, byte(len(host))}
	req = append(req, host...)
	req = binary.BigEndian.AppendUint16(req, uint16(port))
	conn.Write(req)
	var rep [10]byte
	if _, err := io.ReadFull(conn, rep[:]); err != nil {
		t.Fatal(err)
	}
	return conn, rep[1]
}

func TestGatewaySOCKS5(t *testing.T) {
	host, port := startEcho(t)
	p, _ := strconv.Atoi(port)
	allowed := "localhost:" + port

	trusted := startGateway(t, []string{"loopback"}, "", allowed)
	conn, code := socksConnect(t, trusted, "localhost", p, "", "")
	if code != 0 {
		t.Fatalf("trusted SOCKS5: code %d", code)
	}
	echoCheck(t, conn, bufio.NewReader(conn))
	if _, code := socksConnect(t, trusted, host, 1, "", ""); code != socksRepNotAllowed {
		t.Fatalf("not allowed: code %d", code)
	}

	gw := startGateway(t, nil, "s3cret-pass", allowed)
	if _, code := socksConnect(t, gw, "localhost", p, "", ""); code != 0xFF {
		t.Fatalf("untrusted without auth: code %d", code)
	}
	if _, code := socksConnect(t, gw, "localhost", p, "bot", "bad"); code != 0xFE {
		t.Fatalf("wrong password: code %d", code)
	}
	conn, code = socksConnect(t, gw, "localhost", p, "bot", "s3cret-pass")
	if code != 0 {
		t.Fatalf("with password: code %d", code)
	}
	echoCheck(t, conn, bufio.NewReader(conn))
}

func TestHostPatterns(t *testing.T) {
	g := NewGateway(&config.Config{Gateway: config.GatewayConfig{AllowedHosts: []string{"api.telegram.org:443", "*.example.com:*"}}}, nil)
	cases := map[string]bool{
		"api.telegram.org:443": true, "API.Telegram.org.:443": true, "api.telegram.org:80": false,
		"evil-api.telegram.org:443": false, "example.com:1": true, "a.b.example.com:8080": true, "badexample.com:1": false,
	}
	for addr, want := range cases {
		h, p, _ := net.SplitHostPort(addr)
		if got := g.isAllowed(h, p); got != want {
			t.Errorf("%s: %v, want %v", addr, got, want)
		}
	}
	if !strings.Contains(g.allowedList(), "*.example.com:*") {
		t.Fatal(g.allowedList())
	}
}

func TestDialRaceSkipsDeadAddress(t *testing.T) {
	host, port := startEcho(t)
	deadAddr := "127.0.0.1:1" // nothing listens on port 1: refused at once
	conn, addr, err := dialRace(context.Background(), []string{deadAddr, net.JoinHostPort(host, port)}, time.Second)
	if err != nil || addr != net.JoinHostPort(host, port) {
		t.Fatal(addr, err)
	}
	conn.Close()
	if _, _, err := dialRace(context.Background(), []string{deadAddr}, time.Second); err == nil {
		t.Fatal("dead address connected")
	}
}

func TestGatewayFallbackIP(t *testing.T) {
	host, port := startEcho(t)
	cfg := config.Default()
	cfg.Gateway.Host = "127.0.0.1"
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	cfg.Gateway.Port = ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	// "unreachable.invalid" doesn't resolve, the fallback IP is the echo server
	cfg.Gateway.AllowedHosts = []string{"unreachable.invalid:" + port}
	cfg.Gateway.FallbackIPs = map[string][]string{"unreachable.invalid": {host}}
	acl, _ := access.New([]string{"loopback"})
	g := NewGateway(cfg, acl)
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(g.Stop)
	conn, br, status := httpConnect(t, net.JoinHostPort("127.0.0.1", strconv.Itoa(cfg.Gateway.Port)), "unreachable.invalid:"+port, "")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	echoCheck(t, conn, br)
}
