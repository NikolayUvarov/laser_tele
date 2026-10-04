package laser_tele_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const testAPIKEY = "123456:TEST-KEY"

// Tests write files to the current directory, so they are run in a temporary one
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
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type request struct {
	Method      string                 // Bot API method
	Params      url.Values             // query or form params
	JSON        map[string]interface{} // params sent as JSON
	FileField   string
	FileName    string
	FileContent string
}

// fakeTelegram imitates Telegram Bot API server for one bot.
// getUpdates returns queued updates and confirms them by offset, getFile returns path "files/<file_id>",
// files are downloaded from files map, other methods get response with status and body
type fakeTelegram struct {
	apiKey   string
	mu       sync.Mutex
	updates  []map[string]interface{}
	requests []request
	status   int
	body     string
	files    map[string]string
}

func newFakeTelegram(apiKey string) *fakeTelegram {
	return &fakeTelegram{apiKey: apiKey, status: 200, body: `{"ok":true,"result":{}}`, files: map[string]string{}}
}

func (tg *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tg.mu.Lock()
	defer tg.mu.Unlock()

	if filePath := strings.TrimPrefix(r.URL.Path, "/file/bot"+tg.apiKey+"/"); filePath != r.URL.Path {
		content, ok := tg.files[filePath]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(content))
		return
	}
	method := strings.TrimPrefix(r.URL.Path, "/bot"+tg.apiKey+"/")
	if method == r.URL.Path {
		w.WriteHeader(401)
		w.Write([]byte(`{"ok":false,"error_code":401,"description":"Unauthorized"}`))
		return
	}

	req := request{Method: method, Params: r.URL.Query()}
	if r.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(r.Body).Decode(&req.JSON)
	} else if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			req.Params = r.MultipartForm.Value
			for field, files := range r.MultipartForm.File {
				file, _ := files[0].Open()
				data, _ := io.ReadAll(file)
				req.FileField, req.FileName, req.FileContent = field, files[0].Filename, string(data)
			}
		}
	}
	tg.requests = append(tg.requests, req)

	switch method {
	case "getUpdates":
		tg.getUpdates(w, req.Params)
	case "getFile":
		fileID := req.Params.Get("file_id")
		if fileID == "wrong" {
			w.WriteHeader(400)
			w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: invalid file_id"}`))
			return
		}
		fmt.Fprintf(w, `{"ok":true,"result":{"file_id":%q,"file_path":"files/%s"}}`, fileID, fileID)
	default:
		w.WriteHeader(tg.status)
		w.Write([]byte(tg.body))
	}
}

func (tg *fakeTelegram) getUpdates(w http.ResponseWriter, params url.Values) {
	if offsetStr := params.Get("offset"); offsetStr != "" {
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
	body, _ := json.Marshal(map[string]interface{}{"ok": true, "result": tg.updates})
	w.Write(body)
}

func (tg *fakeTelegram) addUpdates(updates ...map[string]interface{}) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.updates = append(tg.updates, updates...)
}

func (tg *fakeTelegram) addFile(filePath, content string) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.files[filePath] = content
}

func (tg *fakeTelegram) setResponse(status int, body string) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.status, tg.body = status, body
}

func (tg *fakeTelegram) lastRequest(t *testing.T) request {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if len(tg.requests) == 0 {
		t.Fatal("No requests")
	}
	return tg.requests[len(tg.requests)-1]
}

// connectToFake sends all requests of the bot to a new server with fake Telegram
func connectToFake(t *testing.T, bot *Bot, tg *fakeTelegram) {
	server := httptest.NewServer(tg)
	t.Cleanup(server.Close)
	bot.apiLink = server.URL + "/bot" + bot.apiKey
	bot.fileLink = server.URL + "/file/bot" + bot.apiKey
	bot.httpGet = server.Client().Get
	bot.httpPost = server.Client().Post
	bot.fileClient = server.Client()
}

func newTestBot(t *testing.T, config LaserTeleConfigT) (*Bot, *fakeTelegram) {
	if config.APIKEY == "" {
		config.APIKEY = testAPIKEY
	}
	if config.LogDir == "" {
		config.LogDir = t.TempDir()
	}
	if config.DownloadDir == "" {
		config.DownloadDir = filepath.Join(t.TempDir(), "downloads")
	}
	config.Timeout = time.Second
	bot, err := NewBot(config)
	if err != nil {
		t.Fatal(err)
	}
	tg := newFakeTelegram(bot.apiKey)
	connectToFake(t, bot, tg)
	return bot, tg
}

var testUser = map[string]interface{}{"id": 137511897, "is_bot": false, "first_name": "Nikolos", "username": "nikolosu", "language_code": "ru"}
var testChat = map[string]interface{}{"id": 137511897, "first_name": "Nikolos", "username": "nikolosu", "type": "private"}

func testMessage(messageID int, text string) map[string]interface{} {
	return map[string]interface{}{"message_id": messageID, "from": testUser, "chat": testChat, "date": 1692347648, "text": text}
}

func testUpdate(updateID int, kind string, object interface{}) map[string]interface{} {
	return map[string]interface{}{"update_id": updateID, kind: object}
}

func textUpdate(updateID int, text string) map[string]interface{} {
	return testUpdate(updateID, "message", testMessage(updateID%1000+1, text))
}

func collect(bot *Bot) []Update {
	var got []Update
	bot.UpdateRequest(func(u Update) { got = append(got, u) })
	return got
}

func readLog(t *testing.T, bot *Bot, logName string) string {
	data, err := os.ReadFile(filepath.Join(bot.logger.dir, logName+".log"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFirstRequestSkipsOldUpdates(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.addUpdates(textUpdate(794872550, "ddd"), textUpdate(794872551, "asd"), textUpdate(794872552, "asdfa"))

	if got := collect(bot); len(got) != 0 {
		t.Errorf("Expected no updates on first request, but got %d", len(got))
	}
	if bot.lastUpdateID != 794872552 {
		t.Errorf("Expected lastUpdateID to be 794872552, but got %d", bot.lastUpdateID)
	}
	if offset := tg.lastRequest(t).Params.Get("offset"); offset != "-1" {
		t.Errorf("Expected offset -1, but got %q", offset)
	}
}

func TestNewUpdatesAreConfirmed(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.addUpdates(textUpdate(794872550, "old"))
	collect(bot)

	// no new updates: must not panic and must not repeat old updates
	if got := collect(bot); len(got) != 0 {
		t.Fatalf("Expected no updates, but got %d", len(got))
	}

	tg.addUpdates(textUpdate(794872551, "first"), textUpdate(794872552, "second"))
	got := collect(bot)
	if len(got) != 2 || got[0].UpdateMessage.Text != "first" || got[1].UpdateMessage.Text != "second" {
		t.Fatalf("Expected updates 'first' and 'second', but got %+v", got)
	}
	if got[1].UpdateID != 794872552 || got[1].UpdateMessage.Chat.ID != 137511897 {
		t.Errorf("Wrong update fields: %+v", got[1])
	}

	if got := collect(bot); len(got) != 0 {
		t.Errorf("Expected no repeated updates, but got %d", len(got))
	}
	if offset := tg.lastRequest(t).Params.Get("offset"); offset != "794872553" {
		t.Errorf("Expected offset 794872553, but got %q", offset)
	}
}

func TestEmptyFirstRequest(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)

	// the first message after start of the bot must not be lost
	tg.addUpdates(textUpdate(794872550, "hello"))
	got := collect(bot)
	if len(got) != 1 || got[0].UpdateMessage.Text != "hello" {
		t.Fatalf("Expected update 'hello', but got %+v", got)
	}
}

func TestUpdateTypes(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)

	photo := testMessage(10, "")
	photo["caption"] = "my photo"
	photo["photo"] = []map[string]interface{}{
		{"file_id": "small", "width": 90, "height": 90},
		{"file_id": "big", "width": 1280, "height": 960},
	}
	photo["reply_to_message"] = testMessage(9, "original")
	photo["forward_origin"] = map[string]interface{}{"type": "user", "date": 1692347000, "sender_user": testUser}
	location := testMessage(11, "")
	location["location"] = map[string]interface{}{"latitude": 55.75, "longitude": 37.62}
	group := testMessage(12, "/start@my_bot")
	group["chat"] = map[string]interface{}{"id": -100555, "type": "supergroup", "title": "Laser team"}
	group["entities"] = []map[string]interface{}{{"type": "bot_command", "offset": 0, "length": 13}}
	post := map[string]interface{}{"message_id": 13, "chat": map[string]interface{}{"id": -100777, "type": "channel", "title": "News"}, "date": 1692347648, "text": "post"}
	button := testMessage(7, "choose")
	button["reply_markup"] = map[string]interface{}{"inline_keyboard": [][]map[string]interface{}{{{"text": "Yes", "callback_data": "yes"}, {"text": "Site", "url": "https://example.com"}}}}
	blocked := map[string]interface{}{
		"chat": testChat, "from": testUser, "date": 1692347648,
		"old_chat_member": map[string]interface{}{"status": "member", "user": map[string]interface{}{"id": 1, "is_bot": true}},
		"new_chat_member": map[string]interface{}{"status": "kicked", "user": map[string]interface{}{"id": 1, "is_bot": true}},
	}
	tg.addUpdates(
		testUpdate(1, "message", photo),
		testUpdate(2, "edited_message", testMessage(8, "edited")),
		testUpdate(3, "channel_post", post),
		testUpdate(4, "callback_query", map[string]interface{}{"id": "query1", "from": testUser, "message": button, "chat_instance": "1", "data": "yes"}),
		testUpdate(5, "my_chat_member", blocked),
		testUpdate(6, "message", location),
		testUpdate(7, "message", group),
		testUpdate(8, "future_update", map[string]interface{}{"id": "unknown"}),
	)

	got := collect(bot)
	var types []string
	for _, u := range got {
		types = append(types, u.Type())
	}
	// unknown kinds of updates are named too and can be read from Raw
	wantTypes := []string{"message", "edited_message", "channel_post", "callback_query", "my_chat_member", "message", "message", "future_update"}
	if !reflect.DeepEqual(types, wantTypes) {
		t.Fatalf("Expected types %v, but got %v", wantTypes, types)
	}

	msg := got[0].UpdateMessage
	if msg.Caption != "my photo" || len(msg.Photo) != 2 || msg.Photo[1].FileID != "big" || msg.Photo[1].Width != 1280 {
		t.Errorf("Wrong photo message: %+v", msg)
	}
	if msg.ReplyToMessage == nil || msg.ReplyToMessage.Text != "original" {
		t.Errorf("Wrong reply_to_message: %+v", msg.ReplyToMessage)
	}
	if msg.ForwardOrigin.Type != "user" || msg.ForwardOrigin.SenderUser.Username != "nikolosu" {
		t.Errorf("Wrong forward_origin: %+v", msg.ForwardOrigin)
	}
	if msg.Location != nil {
		t.Errorf("Expected no location, but got %+v", msg.Location)
	}
	if got[1].EditedMessage.Text != "edited" {
		t.Errorf("Wrong edited message: %+v", got[1].EditedMessage)
	}
	if got[2].ChannelPost.Chat.Title != "News" || got[2].ChannelPost.Text != "post" {
		t.Errorf("Wrong channel post: %+v", got[2].ChannelPost)
	}
	query := got[3].CallbackQuery
	if query.ID != "query1" || query.Data != "yes" || query.Message.Text != "choose" ||
		query.Message.ReplyMarkup.InlineKeyboard[0][1].URL != "https://example.com" {
		t.Errorf("Wrong callback query: %+v", query)
	}
	if got[4].MyChatMember.NewChatMember.Status != "kicked" {
		t.Errorf("Wrong my_chat_member: %+v", got[4].MyChatMember)
	}
	if loc := got[5].UpdateMessage.Location; loc == nil || loc.Latitude != 55.75 || loc.Longitude != 37.62 {
		t.Errorf("Wrong location: %+v", loc)
	}
	if g := got[6].UpdateMessage; g.Chat.Title != "Laser team" || g.Chat.Type != "supergroup" || g.Entities[0].Type != "bot_command" {
		t.Errorf("Wrong group message: %+v", g)
	}
	var future struct {
		FutureUpdate struct {
			ID string `json:"id"`
		} `json:"future_update"`
	}
	if err := json.Unmarshal(got[7].Raw, &future); err != nil || future.FutureUpdate.ID != "unknown" {
		t.Errorf("Wrong raw update: %s %v", got[7].Raw, err)
	}
}

func TestCallbacksAndChan(t *testing.T) {
	var order []string
	bot, tg := newTestBot(t, LaserTeleConfigT{CallbackOnUpdate: func(u Update) { order = append(order, "config:"+u.UpdateMessage.Text) }})
	updates := bot.MakeChan()
	bot.UpdateRequest(nil)

	tg.addUpdates(textUpdate(794872550, "hello"))
	done := make(chan struct{})
	go func() {
		bot.UpdateRequest(func(u Update) { order = append(order, "callback:"+u.UpdateMessage.Text) })
		close(done)
	}()
	if u := <-updates; u.UpdateMessage.Text != "hello" {
		t.Errorf("Expected update 'hello' in channel, but got %+v", u)
	}
	<-done
	if want := []string{"config:hello", "callback:hello"}; !reflect.DeepEqual(order, want) {
		t.Errorf("Expected calls %v, but got %v", want, order)
	}
}

func TestTwoBots(t *testing.T) {
	bot1, tg1 := newTestBot(t, LaserTeleConfigT{APIKEY: "111:KEY-ONE"})
	bot2, tg2 := newTestBot(t, LaserTeleConfigT{APIKEY: "222:KEY-TWO"})
	collect(bot1)
	collect(bot2)

	tg1.addUpdates(textUpdate(100, "to one"))
	tg2.addUpdates(textUpdate(500, "to two"), textUpdate(501, "to two again"))
	if got := collect(bot1); len(got) != 1 || got[0].UpdateMessage.Text != "to one" {
		t.Errorf("Wrong updates of bot1: %+v", got)
	}
	if got := collect(bot2); len(got) != 2 || got[1].UpdateMessage.Text != "to two again" {
		t.Errorf("Wrong updates of bot2: %+v", got)
	}

	if err := bot1.SendMessage(1, "from one"); err != nil {
		t.Fatal(err)
	}
	if err := bot2.SendMessage(2, "from two"); err != nil {
		t.Fatal(err)
	}
	if text := tg1.lastRequest(t).Params.Get("text"); text != "from one" {
		t.Errorf("Expected 'from one' sent by bot1, but got %q", text)
	}
	if text := tg2.lastRequest(t).Params.Get("text"); text != "from two" {
		t.Errorf("Expected 'from two' sent by bot2, but got %q", text)
	}

	// the same file path of different bots is saved to different directories
	tg1.addFile("files/doc1", "one")
	tg2.addFile("files/doc1", "two")
	path1, err1 := bot1.LoadFile("doc1")
	path2, err2 := bot2.LoadFile("doc1")
	if err1 != nil || err2 != nil || path1 == path2 {
		t.Fatalf("Expected files in different directories, but got %q %v, %q %v", path1, err1, path2, err2)
	}

	if log1 := readLog(t, bot1, "sendMessage"); strings.Contains(log1, "111:KEY-ONE") || !strings.Contains(log1, "chat_id=1") {
		t.Errorf("Wrong log of bot1: %s", log1)
	}
	if log2 := readLog(t, bot2, "sendMessage"); strings.Contains(log2, "222:KEY-TWO") || !strings.Contains(log2, "chat_id=2") {
		t.Errorf("Wrong log of bot2: %s", log2)
	}
}

func TestSendMessageText(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	text := "line1\nline2; 100% C# 1+1=2 tom & jerry"
	if err := bot.SendMessage(-1001234567890, text); err != nil {
		t.Fatal(err)
	}
	req := tg.lastRequest(t)
	if req.Method != "sendMessage" || req.Params.Get("text") != text || req.Params.Get("chat_id") != "-1001234567890" {
		t.Errorf("Wrong request: %+v", req)
	}
}

func TestSendKeyboard(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	keyboard := InlineKeyboard{Keyboard: []Row{{AddButton("A & B", "a#b"), {Text: "Site", URL: "https://example.com"}}}}
	if err := bot.SendKeyboard(1, "choose", keyboard); err != nil {
		t.Fatal(err)
	}

	markup := tg.lastRequest(t).Params.Get("reply_markup")
	var got InlineKeyboard
	if err := json.Unmarshal([]byte(markup), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, keyboard) {
		t.Errorf("Expected keyboard %+v, but got %+v", keyboard, got)
	}
	if strings.Contains(markup, `"url":""`) || strings.Contains(markup, `"callback_data":""`) {
		t.Errorf("Empty fields of buttons must not be sent: %s", markup)
	}
}

func TestAnswerCallbackQuery(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	if err := bot.AnswerCallbackQuery("query1", "Done!"); err != nil {
		t.Fatal(err)
	}
	req := tg.lastRequest(t)
	if req.Method != "answerCallbackQuery" || req.Params.Get("callback_query_id") != "query1" || req.Params.Get("text") != "Done!" {
		t.Errorf("Wrong request: %+v", req)
	}

	if err := bot.AnswerCallbackQuery("query2", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := tg.lastRequest(t).Params["text"]; ok {
		t.Errorf("Empty text must not be sent")
	}
}

func TestAPIError(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.setResponse(400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`)

	err := bot.SendMessage(1, "text")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected APIError, but got %v", err)
	}
	if apiErr.ErrorCode != 400 || apiErr.Description != "Bad Request: chat not found" || apiErr.Method != "sendMessage" {
		t.Errorf("Wrong APIError: %+v", apiErr)
	}

	tg.setResponse(502, "<html>Bad Gateway</html>")
	if err := bot.SendMessage(1, "text"); !errors.As(err, &apiErr) || apiErr.ErrorCode != 502 {
		t.Errorf("Expected APIError with code 502, but got %v", err)
	}
}

func TestConnectionErrorHidesKey(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{})
	bot.httpGet = func(link string) (*http.Response, error) {
		return nil, &url.Error{Op: "Get", URL: link, Err: errors.New("connection refused")}
	}

	err := bot.SendMessage(1, "text")
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		t.Fatalf("Expected url.Error, but got %v", err)
	}
	if strings.Contains(err.Error(), testAPIKEY) {
		t.Errorf("Error contains APIKEY: %v", err)
	}
	if log := readLog(t, bot, "sendMessage"); strings.Contains(log, testAPIKEY) || !strings.Contains(log, "CONNECTION_ERROR") {
		t.Errorf("Wrong log: %s", log)
	}
}

func TestSendPhoto(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	fileName := filepath.Join(t.TempDir(), "test.jpg")
	if err := os.WriteFile(fileName, []byte("image data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := bot.SendPhoto(-100123, "Test image & more", fileName); err != nil {
		t.Fatal(err)
	}
	req := tg.lastRequest(t)
	want := request{
		Method:      "sendPhoto",
		Params:      url.Values{"chat_id": {"-100123"}, "caption": {"Test image & more"}},
		FileField:   "photo",
		FileName:    "test.jpg",
		FileContent: "image data",
	}
	if !reflect.DeepEqual(req, want) {
		t.Errorf("Expected request %+v, but got %+v", want, req)
	}
}

func TestSendPhotoMissingFile(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{})
	if err := bot.SendPhoto(1, "caption", "no_such_file.jpg"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected os.ErrNotExist, but got %v", err)
	}
}

func TestLoadFile(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.addFile("files/good", "image data")

	path, err := bot.LoadFile("good")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(bot.downloadDir, "files", "good"); path != want {
		t.Errorf("Expected path %q, but got %q", want, path)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "image data" {
		t.Errorf("Expected file with 'image data', but got %q, %v", data, err)
	}

	var apiErr *APIError
	path, err = bot.LoadFile("wrong")
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 400 || path != "" {
		t.Errorf("Expected APIError 400 for wrong file_id, but got %q, %v", path, err)
	}

	path, err = bot.LoadFile("deleted")
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 404 || path != "" {
		t.Errorf("Expected APIError 404 for failed download, but got %q, %v", path, err)
	}
	if strings.Contains(err.Error(), testAPIKEY) {
		t.Errorf("Error contains APIKEY: %v", err)
	}
}

func TestLogWithoutContent(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)
	tg.addUpdates(textUpdate(794872550, "secret update"))
	collect(bot)
	if err := bot.SendMessage(137, "secret text"); err != nil {
		t.Fatal(err)
	}

	sendLog := readLog(t, bot, "sendMessage")
	if strings.Contains(sendLog, "secret") || !strings.Contains(sendLog, "sendMessage chat_id=137") || !strings.Contains(sendLog, "RESP: 200 OK") {
		t.Errorf("Wrong log: %s", sendLog)
	}
	updatesLog := readLog(t, bot, "updateRequest")
	if strings.Contains(updatesLog, "secret") || !strings.Contains(updatesLog, "UPDATES: 794872550 message") {
		t.Errorf("Wrong log: %s", updatesLog)
	}
	if strings.Contains(sendLog+updatesLog, testAPIKEY) {
		t.Errorf("Log contains APIKEY")
	}
}

func TestLogFull(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{LogMode: LogFull})
	if err := bot.SendMessage(137, "secret text"); err != nil {
		t.Fatal(err)
	}
	if log := readLog(t, bot, "sendMessage"); !strings.Contains(log, "secret+text") || strings.Contains(log, testAPIKEY) {
		t.Errorf("Wrong log: %s", log)
	}
}

func TestLogOff(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{LogMode: LogOff})
	if err := bot.SendMessage(137, "secret text"); err != nil {
		t.Fatal(err)
	}
	if files, _ := os.ReadDir(bot.logger.dir); len(files) != 0 {
		t.Errorf("Expected no log files, but got %d", len(files))
	}
}

func TestLogRotation(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{LogMaxSize: 300})
	for i := 0; i < 20; i++ {
		if err := bot.SendMessage(137, "text"); err != nil {
			t.Fatal(err)
		}
	}
	fileName := filepath.Join(bot.logger.dir, "sendMessage.log")
	if info, err := os.Stat(fileName); err != nil || info.Size() > 300+200 {
		t.Errorf("Log file is not rotated: %v %v", info, err)
	}
	if _, err := os.Stat(fileName + ".1"); err != nil {
		t.Errorf("Expected rotated log file: %v", err)
	}
	if files, _ := os.ReadDir(bot.logger.dir); len(files) != 2 {
		t.Errorf("Expected 2 log files, but got %d", len(files))
	}
}

func TestPackageLevelAPI(t *testing.T) {
	var configGot, callbackGot []string
	DoLaserTeleInit(LaserTeleConfigT{
		APIKEY:           testAPIKEY,
		Timeout:          time.Second,
		LogDir:           t.TempDir(),
		CallbackOnUpdate: func(u Update) { configGot = append(configGot, u.UpdateMessage.Text) },
	})
	defer func() { defaultBot, OnUpdateCallbackFunc, TgChan, isChan, APIKEY = nil, nil, nil, false, "" }()
	tg := newFakeTelegram(testAPIKEY)
	connectToFake(t, defaultBot, tg)
	if APIKEY != testAPIKEY {
		t.Errorf("Expected APIKEY %q, but got %q", testAPIKEY, APIKEY)
	}

	UpdateRequest(nil)
	tg.addUpdates(textUpdate(1, "hello"))
	MakeChan()
	done := make(chan struct{})
	go func() {
		UpdateRequest(func(u Update) { callbackGot = append(callbackGot, u.UpdateMessage.Text) })
		close(done)
	}()
	if u := <-TgChan; u.UpdateMessage.Text != "hello" {
		t.Errorf("Expected update 'hello' in TgChan, but got %+v", u)
	}
	<-done
	// CallbackOnUpdate from config is called once
	if !reflect.DeepEqual(configGot, []string{"hello"}) || !reflect.DeepEqual(callbackGot, []string{"hello"}) {
		t.Errorf("Expected 'hello' in callbacks, but got %v %v", configGot, callbackGot)
	}

	if err := SendMessage(5, "package level"); err != nil || tg.lastRequest(t).Params.Get("text") != "package level" {
		t.Errorf("SendMessage failed: %v", err)
	}
	if err := AnswerCallbackQuery("query1", ""); err != nil || tg.lastRequest(t).Method != "answerCallbackQuery" {
		t.Errorf("AnswerCallbackQuery failed: %v", err)
	}
	tg.addFile("files/doc", "data")
	if path, err := LoadFile(5, "doc"); err != nil || path != filepath.Join("downloadedFiles", "files", "doc") {
		t.Errorf("LoadFile failed: %q %v", path, err)
	}
	// FileDownload keeps old behaviour: nil resp and error message in data
	if resp, data, _ := FileDownload(defaultBot.fileLink+"/files/missing", "files/missing"); resp != nil || !strings.Contains(string(data), "404") {
		t.Errorf("Expected nil resp and error message, but got %v, %q", resp, data)
	}
}

func TestNoAPIKEY(t *testing.T) {
	t.Setenv("TG_API_KEY", "")
	if _, err := NewBot(LaserTeleConfigT{}); err == nil {
		t.Errorf("Expected error without APIKEY")
	}
	// package-level functions return error instead of exiting
	if err := SendMessage(1, "text"); err == nil {
		t.Errorf("Expected error without APIKEY")
	}

	t.Setenv("TG_API_KEY", testAPIKEY)
	t.Setenv("TIMEOUT", "")
	bot, err := NewBot(LaserTeleConfigT{LogMode: LogOff})
	if err != nil || bot.apiKey != testAPIKEY || bot.timeout != 10*time.Second {
		t.Errorf("Expected bot with APIKEY from env and default timeout, but got %+v, %v", bot, err)
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
