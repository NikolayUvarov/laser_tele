package laser_tele_api

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// fakeProxy accepts one connection, remembers its request and refuses it
func fakeProxy(t *testing.T) (string, chan *http.Request) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	requests := make(chan *http.Request, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			return
		}
		requests <- req
		io.WriteString(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
	}()
	return ln.Addr().String(), requests
}

func TestProxyIsUsed(t *testing.T) {
	addr, requests := fakeProxy(t)
	bot, err := NewBot(LaserTeleConfigT{APIKEY: testAPIKEY, LogMode: LogOff, Proxy: "http://bot:pr0xy-pass@" + addr})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(bot.Proxy(), "pr0xy-pass") || !strings.Contains(bot.Proxy(), addr) {
		t.Fatalf("Proxy() = %q", bot.Proxy())
	}
	err = bot.SendMessage(1, "text")
	if err == nil {
		t.Fatal("Expected error from the refusing proxy")
	}
	if strings.Contains(err.Error(), "pr0xy-pass") || strings.Contains(err.Error(), testAPIKEY) {
		t.Fatalf("Secret in error: %v", err)
	}
	select {
	case req := <-requests:
		if req.Method != http.MethodConnect || req.Host != "api.telegram.org:443" {
			t.Fatalf("Expected CONNECT api.telegram.org:443, got %s %s", req.Method, req.Host)
		}
		want := "Basic " + base64.StdEncoding.EncodeToString([]byte("bot:pr0xy-pass"))
		if req.Header.Get("Proxy-Authorization") != want {
			t.Fatalf("Proxy-Authorization = %q", req.Header.Get("Proxy-Authorization"))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("The proxy got no request")
	}
}

func TestProxyFromEnv(t *testing.T) {
	t.Setenv("TG_PROXY", "socks5://127.0.0.1:1080")
	bot, err := NewBot(LaserTeleConfigT{APIKEY: testAPIKEY, LogMode: LogOff})
	if err != nil || bot.Proxy() != "socks5://127.0.0.1:1080" {
		t.Fatalf("Proxy() = %q, err = %v", bot.Proxy(), err)
	}
	bot, err = NewBot(LaserTeleConfigT{APIKEY: testAPIKEY, LogMode: LogOff, Proxy: "http://10.0.0.1:3128"})
	if err != nil || bot.Proxy() != "http://10.0.0.1:3128" {
		t.Fatalf("Config must win over env: %q, %v", bot.Proxy(), err)
	}
}

func TestWrongProxy(t *testing.T) {
	for _, proxy := range []string{"tg://proxy?server=1.2.3.4&port=443&secret=dd00", "ftp://host:21", "http://user:hidden-pass@:bad port"} {
		_, err := NewBot(LaserTeleConfigT{APIKEY: testAPIKEY, LogMode: LogOff, Proxy: proxy})
		if err == nil {
			t.Fatalf("Proxy %q accepted", proxy)
		}
		if strings.Contains(err.Error(), "hidden-pass") {
			t.Fatalf("Password in error: %v", err)
		}
	}
}
