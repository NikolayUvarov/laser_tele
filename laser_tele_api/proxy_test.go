package laser_tele_api

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// startHTTPProxy starts a fake HTTP proxy: it sends the request line and the Proxy-Authorization header
// of every connection to the channel and refuses the connection
func startHTTPProxy(t *testing.T) (string, chan string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	requests := make(chan string, 10)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			reader := bufio.NewReader(conn)
			request := ""
			for {
				line, err := reader.ReadString('\n')
				line = strings.TrimRight(line, "\r\n")
				if err != nil || line == "" {
					break
				}
				if request == "" {
					request = line
				} else if strings.HasPrefix(strings.ToLower(line), "proxy-authorization") {
					request += " | " + line
				}
			}
			conn.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n"))
			conn.Close()
			requests <- request
		}
	}()
	return listener.Addr().String(), requests
}

// startSOCKS5Proxy starts a fake SOCKS5 proxy with authentication by username and password:
// it sends "user:password host:port" of every connection to the channel and refuses the connection
func startSOCKS5Proxy(t *testing.T) (string, chan string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	requests := make(chan string, 10)
	read := func(conn net.Conn, n int) []byte {
		buf := make([]byte, n)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return make([]byte, n)
		}
		return buf
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			// greeting: version, methods; choose username/password
			greeting := read(conn, 2)
			read(conn, int(greeting[1]))
			conn.Write([]byte{5, 2})
			// authentication: version, username, password
			version := read(conn, 2)
			user := read(conn, int(version[1]))
			password := read(conn, int(read(conn, 1)[0]))
			conn.Write([]byte{1, 0})
			// request: version, command, reserved, a domain name and a port
			header := read(conn, 4)
			address := fmt.Sprint("address type ", header[3])
			if header[3] == 3 {
				host := read(conn, int(read(conn, 1)[0]))
				port := read(conn, 2)
				address = net.JoinHostPort(string(host), strconv.Itoa(int(port[0])<<8|int(port[1])))
			}
			// the connection is not allowed by the rules of the proxy
			conn.Write([]byte{5, 2, 0, 1, 0, 0, 0, 0, 0, 0})
			conn.Close()
			requests <- string(user) + ":" + string(password) + " " + address
		}
	}()
	return listener.Addr().String(), requests
}

func newProxyBot(t *testing.T, proxy string) *Bot {
	bot, err := NewBot(LaserTeleConfigT{APIKEY: testAPIKEY, LogMode: LogOff, Timeout: time.Second, Proxy: proxy})
	if err != nil {
		t.Fatal(err)
	}
	return bot
}

func receive(t *testing.T, requests chan string) string {
	select {
	case request := <-requests:
		return request
	case <-time.After(5 * time.Second):
		t.Fatal("The proxy got no request")
		return ""
	}
}

func checkProxyError(t *testing.T, err error) {
	if err == nil {
		t.Fatal("The proxy refused the request, but there is no error")
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), testAPIKEY) {
		t.Errorf("The error shows the password or the token: %v", err)
	}
}

func TestHTTPProxy(t *testing.T) {
	address, requests := startHTTPProxy(t)
	bot := newProxyBot(t, "http://user:secret@"+address)

	checkProxyError(t, bot.SendMessage(1, "text"))
	if request := receive(t, requests); request != "CONNECT api.telegram.org:443 HTTP/1.1 | Proxy-Authorization: Basic dXNlcjpzZWNyZXQ=" {
		t.Errorf("Wrong request to the proxy: %q", request)
	}

	// files are downloaded through the proxy too
	_, _, _, err := bot.downloadFile(bot.fileLink+"/photos/file_1.jpg", "photos/file_1.jpg")
	checkProxyError(t, err)
	if request := receive(t, requests); !strings.HasPrefix(request, "CONNECT api.telegram.org:443") {
		t.Errorf("Wrong request to the proxy: %q", request)
	}

	// an address without a scheme is an HTTP proxy
	checkProxyError(t, newProxyBot(t, address).SendMessage(1, "text"))
	if request := receive(t, requests); request != "CONNECT api.telegram.org:443 HTTP/1.1" {
		t.Errorf("Wrong request to the proxy: %q", request)
	}
}

func TestSOCKS5Proxy(t *testing.T) {
	address, requests := startSOCKS5Proxy(t)
	for _, scheme := range []string{"socks5", "socks5h"} {
		checkProxyError(t, newProxyBot(t, scheme+"://user:secret@"+address).SendMessage(1, "text"))
		if request := receive(t, requests); request != "user:secret api.telegram.org:443" {
			t.Errorf("Wrong request to the %s proxy: %q", scheme, request)
		}
	}
}

func TestWrongProxyAndAPIURL(t *testing.T) {
	for _, config := range []LaserTeleConfigT{
		{Proxy: "http://user:secret@[wrong"},
		{Proxy: "ftp://user:secret@proxy:21"},
		{Proxy: "http://user:secret@"},
		{APIURL: "api.telegram.org"},
		{APIURL: "ftp://api.telegram.org"},
	} {
		config.APIKEY = testAPIKEY
		config.LogMode = LogOff
		_, err := NewBot(config)
		if err == nil {
			t.Errorf("No error for Proxy %q, APIURL %q", config.Proxy, config.APIURL)
			continue
		}
		if strings.Contains(err.Error(), "secret") {
			t.Errorf("The error shows the password: %v", err)
		}
	}
}

func TestAPIURL(t *testing.T) {
	tg := newFakeTelegram(testAPIKEY)
	server := httptest.NewServer(tg)
	defer server.Close()
	tg.addFile("files/file_7", "file content")

	// the default bot with its own server, without connectToFake
	DoLaserTeleInit(LaserTeleConfigT{
		APIKEY:      testAPIKEY,
		Timeout:     time.Second,
		LogMode:     LogOff,
		DownloadDir: filepath.Join(t.TempDir(), "downloads"),
		APIURL:      server.URL + "/",
	})
	defer func() { defaultBot, APIKEY = nil, "" }()

	if err := SendMessage(137, "Hello"); err != nil {
		t.Fatal(err)
	}
	if request := tg.lastRequest(t); request.Method != "sendMessage" {
		t.Errorf("Wrong request: %+v", request)
	}

	path, err := LoadFile(137, "file_7")
	if err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "file content" {
		t.Errorf("Wrong file %s: %q, %v", path, data, err)
	}
}

func TestHidePassword(t *testing.T) {
	for link, hidden := range map[string]string{
		"http://user:secret@proxy:8080": "http://user:***@proxy:8080",
		"socks5://user@proxy:1080":      "socks5://user@proxy:1080",
		"socks5://proxy:1080":           "socks5://proxy:1080",
		"proxy:3128":                    "proxy:3128",
	} {
		if result := hidePassword(link); result != hidden {
			t.Errorf("hidePassword(%q) = %q, want %q", link, result, hidden)
		}
	}
}
