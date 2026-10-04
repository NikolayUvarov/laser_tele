package laser_tele_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testAPIKEY = "123456:TEST-KEY"

// Tests write log files to the current directory, so they are run in a temporary one
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "laser_tele_test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := os.Chdir(dir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	APIKEY = testAPIKEY
	tgApiLinkKEY = tgApiLink + APIKEY
	tgApiFileLinkKEY = tgApiFileLink + APIKEY
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func mockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
}

func testMessage(updateID int, text string) map[string]interface{} {
	return map[string]interface{}{
		"update_id": updateID,
		"message": map[string]interface{}{
			"message_id": updateID - 794872255,
			"from":       map[string]interface{}{"id": 137511897, "is_bot": false, "first_name": "Nikolos", "username": "nikolosu", "language_code": "ru"},
			"chat":       map[string]interface{}{"id": 137511897, "first_name": "Nikolos", "username": "nikolosu", "type": "private"},
			"date":       1692347648,
			"text":       text,
		},
	}
}

// fakeTelegram imitates getUpdates of Telegram: keeps queue of updates and confirms them by offset.
// Other methods get response with status and body
type fakeTelegram struct {
	updates  []map[string]interface{}
	requests []*url.URL
	status   int
	body     string
}

func (tg *fakeTelegram) get(link string) (*http.Response, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	tg.requests = append(tg.requests, u)
	if !strings.HasSuffix(u.Path, "/getUpdates") {
		return mockResponse(tg.status, tg.body), nil
	}

	if offsetStr := u.Query().Get("offset"); offsetStr != "" {
		offset, _ := strconv.Atoi(offsetStr)
		var left []map[string]interface{}
		for num, upd := range tg.updates {
			id := upd["update_id"].(int)
			if (offset < 0 && num >= len(tg.updates)+offset) || (offset >= 0 && id >= offset) {
				left = append(left, upd)
			}
		}
		tg.updates = left
	}
	body, _ := json.Marshal(map[string]interface{}{"ok": true, "result": append([]map[string]interface{}{}, tg.updates...)})
	return mockResponse(200, string(body)), nil
}

func useFakeTelegram(t *testing.T, updates ...map[string]interface{}) *fakeTelegram {
	tg := &fakeTelegram{updates: updates, status: 200, body: `{"ok":true,"result":{}}`}
	originalGet := httpGet
	httpGet = tg.get
	newUpdate = Update{}
	isUpdatesInitialized = false
	t.Cleanup(func() { httpGet = originalGet })
	return tg
}

// useTestServer sends all requests of the library to the test server with handler
func useTestServer(t *testing.T, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	originalLink, originalFileLink, originalGet, originalClient := tgApiLinkKEY, tgApiFileLinkKEY, httpGet, fileClient
	tgApiLinkKEY, tgApiFileLinkKEY = server.URL+"/bot"+APIKEY, server.URL+"/file/bot"+APIKEY
	httpGet, fileClient = server.Client().Get, server.Client()
	t.Cleanup(func() {
		server.Close()
		tgApiLinkKEY, tgApiFileLinkKEY, httpGet, fileClient = originalLink, originalFileLink, originalGet, originalClient
	})
}

func collectUpdates(t *testing.T) []Update {
	var got []Update
	UpdateRequest(func(u Update) { got = append(got, u) })
	return got
}

func TestMakeRequest(t *testing.T) {
	useFakeTelegram(t,
		testMessage(794872550, "ddd"),
		testMessage(794872551, "asd"),
		testMessage(794872552, "asdfa"),
	)

	// updates sent before start of the bot are skipped
	if got := collectUpdates(t); len(got) != 0 {
		t.Errorf("Expected no updates on first request, but got %d", len(got))
	}
	if newUpdate.UpdateID != 794872552 {
		t.Errorf("Expected newUpdate.UpdateID to be 794872552, but got %d", newUpdate.UpdateID)
	}
}

func TestUpdateRequestNewUpdates(t *testing.T) {
	tg := useFakeTelegram(t, testMessage(794872550, "old"))
	collectUpdates(t)

	// no new updates: must not panic and must not repeat old updates
	if got := collectUpdates(t); len(got) != 0 {
		t.Fatalf("Expected no updates, but got %d", len(got))
	}

	tg.updates = append(tg.updates, testMessage(794872551, "first"), testMessage(794872552, "second"))
	got := collectUpdates(t)
	if len(got) != 2 || got[0].UpdateMessage.Text != "first" || got[1].UpdateMessage.Text != "second" {
		t.Fatalf("Expected updates 'first' and 'second', but got %+v", got)
	}
	if got[1].UpdateID != 794872552 || got[1].UpdateMessage.Chat.ID != 137511897 {
		t.Errorf("Wrong update fields: %+v", got[1])
	}

	// processed updates must be confirmed
	if got := collectUpdates(t); len(got) != 0 {
		t.Errorf("Expected no repeated updates, but got %d", len(got))
	}
	if offset := tg.requests[len(tg.requests)-1].Query().Get("offset"); offset != "794872553" {
		t.Errorf("Expected offset 794872553, but got %q", offset)
	}
}

func TestUpdateRequestEmptyFirstRequest(t *testing.T) {
	tg := useFakeTelegram(t)
	collectUpdates(t)

	// the first message after start of the bot must not be lost
	tg.updates = append(tg.updates, testMessage(794872550, "hello"))
	got := collectUpdates(t)
	if len(got) != 1 || got[0].UpdateMessage.Text != "hello" {
		t.Fatalf("Expected update 'hello', but got %+v", got)
	}
}

func TestUpdateRequestChan(t *testing.T) {
	tg := useFakeTelegram(t)
	MakeChan()
	defer func() { isChan = false }()
	collectUpdates(t)

	tg.updates = append(tg.updates, testMessage(794872550, "to chan"))
	done := make(chan struct{})
	go func() {
		UpdateRequest(nil)
		close(done)
	}()
	if u := <-TgChan; u.UpdateMessage.Text != "to chan" {
		t.Errorf("Expected update 'to chan' in channel, but got %+v", u)
	}
	<-done
}

func TestSendMessageText(t *testing.T) {
	tg := useFakeTelegram(t)
	text := "line1\nline2; 100% C# 1+1=2 tom & jerry"
	if err := SendMessage(-1001234567890, text); err != nil {
		t.Fatal(err)
	}

	if len(tg.requests) != 1 {
		t.Fatalf("Expected 1 request, but got %d", len(tg.requests))
	}
	q := tg.requests[0].Query()
	if q.Get("text") != text {
		t.Errorf("Expected text %q, but got %q", text, q.Get("text"))
	}
	if q.Get("chat_id") != "-1001234567890" {
		t.Errorf("Expected chat_id -1001234567890, but got %q", q.Get("chat_id"))
	}
}

func TestSendKeyboard(t *testing.T) {
	tg := useFakeTelegram(t)
	keyboard := InlineKeyboard{Keyboard: []Row{{AddButton("A & B", "a#b")}}}
	if err := SendKeyboard(1, "choose", keyboard); err != nil {
		t.Fatal(err)
	}

	var got InlineKeyboard
	if err := json.Unmarshal([]byte(tg.requests[0].Query().Get("reply_markup")), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Keyboard) != 1 || got.Keyboard[0][0] != keyboard.Keyboard[0][0] {
		t.Errorf("Expected keyboard %+v, but got %+v", keyboard, got)
	}
}

func TestLogsHideAPIKEY(t *testing.T) {
	useFakeTelegram(t)
	SendMessage(1, "secret check")

	data, err := os.ReadFile("sendMessage.log")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), testAPIKEY) {
		t.Errorf("Log contains APIKEY: %s", data)
	}
	if !strings.Contains(string(data), "secret+check") {
		t.Errorf("Log doesn't contain request: %s", data)
	}
}

func TestSendMessageAPIError(t *testing.T) {
	tg := useFakeTelegram(t)
	tg.status = 400
	tg.body = `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`

	err := SendMessage(1, "text")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, but got %v", err)
	}
	if apiErr.ErrorCode != 400 || apiErr.Description != "Bad Request: chat not found" || apiErr.Method != "sendMessage" {
		t.Errorf("Wrong APIError: %+v", apiErr)
	}

	tg.status = 502
	tg.body = "<html>Bad Gateway</html>"
	if err := SendMessage(1, "text"); !errors.As(err, &apiErr) || apiErr.ErrorCode != 502 {
		t.Errorf("Expected APIError with code 502, but got %v", err)
	}
}

func TestSendMessageConnectionError(t *testing.T) {
	originalGet := httpGet
	defer func() { httpGet = originalGet }()
	httpGet = func(link string) (*http.Response, error) {
		return nil, &url.Error{Op: "Get", URL: link, Err: errors.New("connection refused")}
	}

	err := SendMessage(1, "text")
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("Expected url.Error, but got %v", err)
	}
	if strings.Contains(err.Error(), testAPIKEY) {
		t.Errorf("Error contains APIKEY: %v", err)
	}
}

func TestSendPhotoMissingFile(t *testing.T) {
	if err := SendPhoto(1, "caption", "no_such_file.jpg"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, but got %v", err)
	}
}

func TestSendPhoto(t *testing.T) {
	type request struct {
		path, chatID, caption, fileName, content string
	}
	var got request
	useTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.chatID = r.FormValue("chat_id")
		got.caption = r.FormValue("caption")
		if file, header, err := r.FormFile("photo"); err == nil {
			data, _ := io.ReadAll(file)
			got.fileName, got.content = header.Filename, string(data)
		}
		w.Write([]byte(`{"ok":true,"result":{}}`))
	})

	if err := os.WriteFile("test.jpg", []byte("image data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SendPhoto(-100123, "Test image & more", "test.jpg"); err != nil {
		t.Fatal(err)
	}
	want := request{"/bot" + testAPIKEY + "/sendPhoto", "-100123", "Test image & more", "test.jpg", "image data"}
	if got != want {
		t.Errorf("Expected request %+v, but got %+v", want, got)
	}
}

func TestLoadFile(t *testing.T) {
	useTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bot" + testAPIKEY + "/getFile":
			switch fileID := r.URL.Query().Get("file_id"); fileID {
			case "good":
				w.Write([]byte(`{"ok":true,"result":{"file_id":"good","file_path":"photos/file_1.jpg"}}`))
			case "deleted":
				w.Write([]byte(`{"ok":true,"result":{"file_id":"deleted","file_path":"photos/deleted.jpg"}}`))
			default:
				w.WriteHeader(400)
				w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: invalid file_id"}`))
			}
		case "/file/bot" + testAPIKEY + "/photos/file_1.jpg":
			w.Write([]byte("image data"))
		default:
			http.NotFound(w, r)
		}
	})

	path, err := LoadFile(1, "good")
	if err != nil {
		t.Fatal(err)
	}
	if path != "downloadedFiles/photos/file_1.jpg" {
		t.Errorf("Expected path downloadedFiles/photos/file_1.jpg, but got %q", path)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "image data" {
		t.Errorf("Expected file with 'image data', but got %q, %v", data, err)
	}

	var apiErr *APIError
	path, err = LoadFile(1, "wrong")
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 400 || path != "" {
		t.Errorf("Expected APIError 400 for wrong file_id, but got %q, %v", path, err)
	}

	path, err = LoadFile(1, "deleted")
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 404 || path != "" {
		t.Errorf("Expected APIError 404 for failed download, but got %q, %v", path, err)
	}
	if strings.Contains(err.Error(), testAPIKEY) {
		t.Errorf("Error contains APIKEY: %v", err)
	}

	// FileDownload keeps old behaviour: nil resp and error message in data
	if resp, data, _ := FileDownload(tgApiFileLinkKEY+"/photos/deleted.jpg", "photos/deleted.jpg"); resp != nil || !strings.Contains(string(data), "404") {
		t.Errorf("Expected nil resp and error message, but got %v, %q", resp, data)
	}
}

func TestConfigCallbackOnUpdate(t *testing.T) {
	tg := useFakeTelegram(t)
	defer func() { OnUpdateCallbackFunc = nil }()

	var got []string
	DoLaserTeleInit(LaserTeleConfigT{
		APIKEY:           testAPIKEY,
		Timeout:          time.Second,
		CallbackOnUpdate: func(u Update) { got = append(got, u.UpdateMessage.Text) },
	})
	UpdateRequest(nil)

	tg.updates = append(tg.updates, testMessage(794872550, "hello"))
	UpdateRequest(nil)
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("Expected config callback to get 'hello', but got %v", got)
	}
}

func TestLoadApiKeyFromFile(t *testing.T) {
	if err := os.WriteFile(".APIKEY_test", []byte("  "+testAPIKEY+"\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if key := loadApiKeyFromFile(".APIKEY_test"); key != testAPIKEY {
		t.Errorf("Expected %q, but got %q", testAPIKEY, key)
	}
	if key := loadApiKeyFromFile(".APIKEY_missing"); key != "" {
		t.Errorf("Expected empty key, but got %q", key)
	}
}

func TestCheckPath(t *testing.T) {
	if err := checkPath("a/b/c.txt"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat("a/b"); err != nil || !info.IsDir() {
		t.Errorf("Expected directory a/b to be created")
	}
	if err := checkPath("a/b"); err == nil {
		t.Errorf("Expected error for directory")
	}
}
