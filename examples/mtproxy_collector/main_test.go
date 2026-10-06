package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	laser_tele "github.com/NikolayUvarov/laser_tele/v2/laser_tele_api"
)

const testToken = "123456:TEST-KEY"

// fakeServer imitates the Bot API server with the MTProxy registry
type fakeServer struct {
	mu        sync.Mutex
	requests  []fakeRequest
	responses map[string]string // method -> response body
}

type fakeRequest struct {
	Method string
	Params map[string]interface{}
}

func (s *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	params := map[string]interface{}{}
	if body, _ := io.ReadAll(r.Body); len(body) > 0 {
		json.Unmarshal(body, &params)
	}
	for key, values := range r.URL.Query() {
		params[key] = values[0]
	}
	s.requests = append(s.requests, fakeRequest{method, params})
	response, ok := s.responses[method]
	if !ok {
		response = `{"ok":true,"result":true}`
	}
	if strings.Contains(response, `"ok":false`) {
		w.WriteHeader(403)
	}
	w.Write([]byte(response))
}

func (s *fakeServer) take() []fakeRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	requests := s.requests
	s.requests = nil
	return requests
}

func newTestCollector(t *testing.T, channels string) (*collector, *fakeServer) {
	server := &fakeServer{responses: map[string]string{
		"addMTProxies": `{"ok":true,"result":{"added":1,"known":0,"ids":[3],"errors":[{"link":"tg://proxy?x","error":"wrong port of the MTProxy"}]}}`,
		"sendMessage":  `{"ok":true,"result":{"message_id":1,"chat":{"id":1}}}`,
	}}
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)
	bot, err := laser_tele.NewBot(laser_tele.LaserTeleConfigT{APIKEY: testToken, APIURL: httpServer.URL,
		LogMode: laser_tele.LogOff, DownloadDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	c, err := newCollector(bot, channels, "137")
	if err != nil {
		t.Fatal(err)
	}
	return c, server
}

func update(t *testing.T, kind, message string) laser_tele.Update {
	var u laser_tele.Update
	if err := json.Unmarshal([]byte(`{"update_id":1,"`+kind+`":`+message+`}`), &u); err != nil {
		t.Fatal(err)
	}
	return u
}

const channel = `"chat":{"id":-100123,"type":"channel","title":"Proxies","username":"FreeProxies"}`
const admin = `"from":{"id":137,"first_name":"Admin"},"chat":{"id":137,"type":"private"}`

func TestChannelPosts(t *testing.T) {
	c, server := newTestCollector(t, "@freeproxies, -100777")

	// links in the text, a hidden link and a button
	c.handle(update(t, "channel_post", `{"message_id":7,`+channel+`,"text":"Fresh: tg://proxy?server=a&port=443&secret=dd00 and Connect",
		"entities":[{"type":"text_link","offset":0,"length":5,"url":"https://t.me/proxy?server=b&port=443&secret=dd11"}],
		"reply_markup":{"inline_keyboard":[[{"text":"Connect","url":"https://t.me/proxy?server=c&port=443&secret=dd22"},{"text":"Site","url":"https://example.com"}]]}}`))
	requests := server.take()
	if len(requests) != 1 || requests[0].Method != "addMTProxies" || requests[0].Params["source"] != "channel:-100123/7" {
		t.Fatalf("Wrong requests: %+v", requests)
	}
	links := requests[0].Params["links"].([]interface{})
	if len(links) != 3 || !strings.Contains(links[1].(string), "server=b") || !strings.Contains(links[2].(string), "server=c") {
		t.Errorf("Wrong links: %v", links)
	}

	// edited posts are collected too, posts without links and other channels are not
	c.handle(update(t, "edited_channel_post", `{"message_id":8,`+channel+`,"caption":"tg://proxy?server=d&port=1&secret=dd33"}`))
	c.handle(update(t, "channel_post", `{"message_id":9,`+channel+`,"text":"Good morning"}`))
	c.handle(update(t, "channel_post", `{"message_id":10,"chat":{"id":-100555,"type":"channel"},"text":"tg://proxy?server=e&port=1&secret=dd44"}`))
	requests = server.take()
	if len(requests) != 1 || requests[0].Params["source"] != "channel:-100123/8" {
		t.Errorf("Wrong requests: %+v", requests)
	}
}

func TestAdminMessages(t *testing.T) {
	c, server := newTestCollector(t, "")

	// a post forwarded from any channel
	c.handle(update(t, "message", `{"message_id":20,`+admin+`,"text":"proxy.example.com:443:dd00",
		"forward_origin":{"type":"channel","date":1,"chat":{"id":-100999,"type":"channel"},"message_id":45}}`))
	requests := server.take()
	if len(requests) != 2 || requests[0].Params["source"] != "channel:-100999/45" || requests[1].Method != "sendMessage" {
		t.Fatalf("Wrong requests: %+v", requests)
	}
	if text := requests[1].Params["text"].(string); !strings.Contains(text, "Added 1, already known 0") || !strings.Contains(text, "wrong port") {
		t.Errorf("Wrong reply: %q", text)
	}

	// other users can't add servers
	c.handle(update(t, "message", `{"message_id":21,"from":{"id":500},"chat":{"id":500,"type":"private"},"text":"tg://proxy?server=a"}`))
	requests = server.take()
	if len(requests) != 1 || !strings.Contains(requests[0].Params["text"].(string), "Your identifier is 500") {
		t.Errorf("Wrong requests: %+v", requests)
	}

	// commands
	c.handle(update(t, "message", `{"message_id":22,`+admin+`,"text":"/use@collector_bot 5"}`))
	c.handle(update(t, "message", `{"message_id":23,`+admin+`,"text":"/remove"}`))
	requests = server.take()
	if len(requests) != 3 || requests[0].Method != "setMTProxy" || requests[0].Params["id"] != float64(5) ||
		requests[1].Params["text"] != "Done" || !strings.Contains(requests[2].Params["text"].(string), "Usage: /remove") {
		t.Errorf("Wrong requests: %+v", requests)
	}

	// errors of the server are shown
	server.responses["checkMTProxies"] = `{"ok":false,"error_code":403,"description":"Forbidden: the bot isn't an MTProxy administrator"}`
	c.handle(update(t, "message", `{"message_id":24,`+admin+`,"text":"/check"}`))
	requests = server.take()
	if len(requests) != 2 || !strings.Contains(requests[1].Params["text"].(string), "isn't an MTProxy administrator") {
		t.Errorf("Wrong requests: %+v", requests)
	}
}

func TestRegistryList(t *testing.T) {
	c, server := newTestCollector(t, "")
	server.responses["getMTProxies"] = `{"ok":true,"result":{"active_id":2,"bot_count":2,"connected_bot_count":1,"proxies":[
		{"id":1,"server":"a.example.com","port":443,"secret_type":"dd","state":"failing","last_error":"Connection closed"},
		{"id":2,"server":"b.example.com","port":443,"secret_type":"ee","domain":"google.com","state":"working","is_active":true,"ping":0.25},
		{"id":3,"server":"c.example.com","port":443,"secret_type":"dd","state":"working","ping":0.1}]}}`
	c.handle(update(t, "message", `{"message_id":30,`+admin+`,"text":"/proxies"}`))
	requests := server.take()
	if len(requests) != 2 {
		t.Fatalf("Wrong requests: %+v", requests)
	}
	want := "Bots connected: 1 of 2. Servers: 3\n✅ #2 b.example.com:443 ee google.com, 250 ms ← active\n" +
		"✅ #3 c.example.com:443 dd, 100 ms\n❌ #1 a.example.com:443 dd, Connection closed"
	if text := requests[1].Params["text"]; text != want {
		t.Errorf("Wrong list:\n%s\nwant:\n%s", text, want)
	}
}

func TestLongRegistryList(t *testing.T) {
	reg := registry{}
	for i := 0; i < 200; i++ {
		reg.Proxies = append(reg.Proxies, registryProxy{ID: int64(i), Server: "proxy.example.com", Port: 443, State: "failing",
			LastError: "Connection closed"})
	}
	text := describeRegistry(reg)
	if len(text) > 4096 || !strings.Contains(text, "more") {
		t.Errorf("Wrong long list of %d bytes: ...%s", len(text), text[len(text)-40:])
	}
}
