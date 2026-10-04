package laser_tele_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
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
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func mockResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
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

// fakeTelegram imitates getUpdates of Telegram: keeps queue of updates and confirms them by offset
type fakeTelegram struct {
	updates  []map[string]interface{}
	requests []*url.URL
}

func (tg *fakeTelegram) get(link string) (*http.Response, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	tg.requests = append(tg.requests, u)
	if !strings.HasSuffix(u.Path, "/getUpdates") {
		return mockResponse(`{"ok":true,"result":{}}`), nil
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
	return mockResponse(string(body)), nil
}

func useFakeTelegram(t *testing.T, updates ...map[string]interface{}) *fakeTelegram {
	tg := &fakeTelegram{updates: updates}
	originalGet := httpGet
	httpGet = tg.get
	newUpdate = Update{}
	isUpdatesInitialized = false
	t.Cleanup(func() { httpGet = originalGet })
	return tg
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
	SendMessage(-1001234567890, text)

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
	SendKeyboard(1, "choose", keyboard)

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

func TestSendPhotoMissingFile(t *testing.T) {
	// must not panic
	SendPhoto(1, "caption", "no_such_file.jpg")
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
