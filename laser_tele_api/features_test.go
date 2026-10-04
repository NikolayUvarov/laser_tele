package laser_tele_api

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// checkJSON checks that value sent as JSON is equal to want
func checkJSON(t *testing.T, name string, value interface{}, want string) {
	t.Helper()
	data, _ := json.Marshal(value)
	var got, expected interface{}
	json.Unmarshal(data, &got)
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatalf("Wrong expected JSON for %s: %v", name, err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("%s: expected %s, but got %s", name, want, data)
	}
}

// checkRequest checks the method and JSON params of the last request
func checkRequest(t *testing.T, tg *fakeTelegram, method, wantParams string) {
	t.Helper()
	req := tg.lastRequest(t)
	if req.Method != method {
		t.Fatalf("Expected method %s, but got %s", method, req.Method)
	}
	checkJSON(t, method, req.JSON, wantParams)
}

func TestAllUpdateKinds(t *testing.T) {
	user := testUser
	chat := testChat
	message := testMessage(1, "text")
	member := map[string]interface{}{"status": "member", "user": user}
	boostSource := map[string]interface{}{"source": "premium", "user": user}
	objects := map[string]map[string]interface{}{
		"message":                    message,
		"edited_message":             message,
		"channel_post":               message,
		"edited_channel_post":        message,
		"business_connection":        {"id": "bc1", "user": user, "user_chat_id": 5, "date": 1, "is_enabled": true, "rights": map[string]interface{}{"can_reply": true}},
		"business_message":           message,
		"edited_business_message":    message,
		"deleted_business_messages":  {"business_connection_id": "bc1", "chat": chat, "message_ids": []int{3, 4}},
		"guest_message":              message,
		"message_reaction":           {"chat": chat, "message_id": 7, "user": user, "date": 1, "old_reaction": []interface{}{}, "new_reaction": []map[string]string{{"type": "emoji", "emoji": "👍"}}},
		"message_reaction_count":     {"chat": chat, "message_id": 7, "date": 1, "reactions": []map[string]interface{}{{"type": map[string]string{"type": "emoji", "emoji": "🔥"}, "total_count": 3}}},
		"inline_query":               {"id": "iq1", "from": user, "query": "cats", "offset": ""},
		"chosen_inline_result":       {"result_id": "r1", "from": user, "query": "cats"},
		"callback_query":             {"id": "cq1", "from": user, "chat_instance": "1", "game_short_name": "mygame"},
		"shipping_query":             {"id": "sq1", "from": user, "invoice_payload": "order1", "shipping_address": map[string]string{"country_code": "RU", "city": "Moscow"}},
		"pre_checkout_query":         {"id": "pq1", "from": user, "currency": "XTR", "total_amount": 50, "invoice_payload": "order1"},
		"purchased_paid_media":       {"from": user, "paid_media_payload": "media1"},
		"poll":                       {"id": "p1", "question": "Tea?", "options": []map[string]interface{}{{"persistent_id": "o1", "text": "Yes", "voter_count": 2}}, "total_voter_count": 2, "is_closed": true, "type": "quiz", "correct_option_ids": []int{0}},
		"poll_answer":                {"poll_id": "p1", "user": user, "option_ids": []int{1}, "option_persistent_ids": []string{"o2"}},
		"my_chat_member":             {"chat": chat, "from": user, "date": 1, "old_chat_member": member, "new_chat_member": map[string]interface{}{"status": "kicked", "user": user}},
		"chat_member":                {"chat": chat, "from": user, "date": 1, "old_chat_member": member, "new_chat_member": map[string]interface{}{"status": "administrator", "user": user, "can_pin_messages": true}},
		"chat_join_request":          {"chat": chat, "from": user, "user_chat_id": 5, "date": 1, "bio": "hello"},
		"chat_boost":                 {"chat": chat, "boost": map[string]interface{}{"boost_id": "b1", "add_date": 1, "expiration_date": 2, "source": boostSource}},
		"removed_chat_boost":         {"chat": chat, "boost_id": "b1", "remove_date": 2, "source": boostSource},
		"managed_bot":                {"user": user, "bot": map[string]interface{}{"id": 777, "is_bot": true, "first_name": "Managed"}},
		"subscription":               {"user": user, "invoice_payload": "sub1", "state": "canceled"},
		"stopped_message_generation": {"chat": chat, "draft_id": 42},
	}
	checks := map[string]func(u Update) bool{
		"message":                    func(u Update) bool { return u.UpdateMessage.Text == "text" },
		"edited_message":             func(u Update) bool { return u.EditedMessage.Text == "text" },
		"channel_post":               func(u Update) bool { return u.ChannelPost.Text == "text" },
		"edited_channel_post":        func(u Update) bool { return u.EditedChannelPost.Text == "text" },
		"business_connection":        func(u Update) bool { return u.BusinessConnection.ID == "bc1" && u.BusinessConnection.Rights.CanReply },
		"business_message":           func(u Update) bool { return u.BusinessMessage.Text == "text" },
		"edited_business_message":    func(u Update) bool { return u.EditedBusinessMessage.Text == "text" },
		"deleted_business_messages":  func(u Update) bool { return reflect.DeepEqual(u.DeletedBusinessMessages.MessageIDs, []int{3, 4}) },
		"guest_message":              func(u Update) bool { return u.GuestMessage.Text == "text" },
		"message_reaction":           func(u Update) bool { return u.MessageReaction.NewReaction[0].Emoji == "👍" },
		"message_reaction_count":     func(u Update) bool { return u.MessageReactionCount.Reactions[0].TotalCount == 3 },
		"inline_query":               func(u Update) bool { return u.InlineQuery.Query == "cats" },
		"chosen_inline_result":       func(u Update) bool { return u.ChosenInlineResult.ResultID == "r1" },
		"callback_query":             func(u Update) bool { return u.CallbackQuery.GameShortName == "mygame" },
		"shipping_query":             func(u Update) bool { return u.ShippingQuery.ShippingAddress.City == "Moscow" },
		"pre_checkout_query":         func(u Update) bool { return u.PreCheckoutQuery.TotalAmount == 50 },
		"purchased_paid_media":       func(u Update) bool { return u.PurchasedPaidMedia.PaidMediaPayload == "media1" },
		"poll":                       func(u Update) bool { return u.Poll.Options[0].VoterCount == 2 && u.Poll.CorrectOptionIDs[0] == 0 },
		"poll_answer":                func(u Update) bool { return u.PollAnswer.OptionIDs[0] == 1 },
		"my_chat_member":             func(u Update) bool { return u.MyChatMember.NewChatMember.Status == "kicked" },
		"chat_member":                func(u Update) bool { return u.ChatMember.NewChatMember.CanPinMessages },
		"chat_join_request":          func(u Update) bool { return u.ChatJoinRequest.Bio == "hello" },
		"chat_boost":                 func(u Update) bool { return u.ChatBoost.Boost.Source.Source == "premium" },
		"removed_chat_boost":         func(u Update) bool { return u.RemovedChatBoost.BoostID == "b1" },
		"managed_bot":                func(u Update) bool { return u.ManagedBot.Bot.ID == 777 },
		"subscription":               func(u Update) bool { return u.Subscription.State == "canceled" },
		"stopped_message_generation": func(u Update) bool { return u.StoppedMessageGeneration.DraftID == 42 },
	}
	if len(objects) != len(AllUpdateTypes) || len(checks) != len(AllUpdateTypes) {
		t.Fatalf("Test must check all %d kinds of updates", len(AllUpdateTypes))
	}

	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)
	for num, kind := range AllUpdateTypes {
		tg.addUpdates(testUpdate(num+1, kind, objects[kind]))
	}
	got := collect(bot)
	if len(got) != len(AllUpdateTypes) {
		t.Fatalf("Expected %d updates, but got %d", len(AllUpdateTypes), len(got))
	}
	for num, kind := range AllUpdateTypes {
		update := got[num]
		if update.Type() != kind {
			t.Errorf("Expected type %s, but got %s", kind, update.Type())
		}
		if !checks[kind](update) {
			t.Errorf("Update %s is parsed wrong: %s", kind, update.Raw)
		}
		// updates created by the application have the same type
		created := update
		created.kind = ""
		if created.Type() != kind {
			t.Errorf("Expected type %s of created update, but got %q", kind, created.Type())
		}
	}
}

func TestAllowedUpdates(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)
	var got []string
	if err := json.Unmarshal([]byte(tg.lastRequest(t).Params.Get("allowed_updates")), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, AllUpdateTypes) {
		t.Errorf("Expected all kinds of updates, but got %v", got)
	}

	bot, tg = newTestBot(t, LaserTeleConfigT{AllowedUpdates: []string{"message", "poll_answer"}})
	collect(bot)
	if got := tg.lastRequest(t).Params.Get("allowed_updates"); got != `["message","poll_answer"]` {
		t.Errorf("Wrong allowed_updates: %s", got)
	}
}

func TestMessageFields(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	collect(bot)
	message := testMessage(5, "")
	message["dice"] = map[string]interface{}{"emoji": "🎲", "value": 4}
	message["venue"] = map[string]interface{}{"location": map[string]float64{"latitude": 1, "longitude": 2}, "title": "Office", "address": "Street 1"}
	message["successful_payment"] = map[string]interface{}{"currency": "XTR", "total_amount": 50, "invoice_payload": "order1", "telegram_payment_charge_id": "ch1"}
	message["pinned_message"] = testMessage(4, "pinned")
	message["business_connection_id"] = "bc1"
	message["via_bot"] = map[string]interface{}{"id": 9, "is_bot": true, "first_name": "Other"}
	message["poll"] = map[string]interface{}{"id": "p1", "question": "Tea?"}
	message["game"] = map[string]interface{}{"title": "Race", "description": "Fast"}
	tg.addUpdates(testUpdate(1, "message", message))

	msg := collect(bot)[0].UpdateMessage
	if msg.Dice.Value != 4 || msg.Venue.Title != "Office" || msg.Venue.Location.Longitude != 2 ||
		msg.SuccessfulPayment.TelegramPaymentChargeID != "ch1" || msg.PinnedMessage == nil || msg.PinnedMessage.Text != "pinned" ||
		msg.BusinessConnectionID != "bc1" || msg.ViaBot.ID != 9 || msg.Poll.Question != "Tea?" || msg.Game.Title != "Race" {
		t.Errorf("Wrong message: %+v", msg)
	}
}

func TestSendMessageWithConfig(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.setResponse(200, `{"ok":true,"result":{"message_id":55,"chat":{"id":1},"text":"<b>Hi</b>"}}`)
	keyboard := &InlineKeyboard{Keyboard: []Row{{AddButton("OK", "ok")}}}
	message, err := bot.SendMessageWithConfig(1, "<b>Hi</b>", MessageConfig{
		ParseMode: "HTML", ReplyToMessageID: 10, BusinessConnectionID: "bc1", ProtectContent: true, ReplyMarkup: keyboard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if message.MessageID != 55 {
		t.Errorf("Expected sent message 55, but got %+v", message)
	}
	checkRequest(t, tg, "sendMessage", `{"chat_id":1,"text":"<b>Hi</b>","parse_mode":"HTML","reply_parameters":{"message_id":10},
		"business_connection_id":"bc1","protect_content":true,"reply_markup":{"inline_keyboard":[[{"text":"OK","callback_data":"ok"}]]}}`)

	if err := bot.EditMessageText(1, 55, "edited"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "editMessageText", `{"chat_id":1,"message_id":55,"text":"edited"}`)
}

func TestAnswerCallbackQueryWithConfig(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	if err := bot.AnswerCallbackQueryWithConfig("cq1", CallbackAnswerConfig{URL: "https://example.com/game", ShowAlert: true, CacheTime: 5}); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerCallbackQuery", `{"callback_query_id":"cq1","url":"https://example.com/game","show_alert":true,"cache_time":5}`)
}

func TestPolls(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.setResponse(200, `{"ok":true,"result":{"message_id":7,"chat":{"id":1},"poll":{"id":"p1","question":"2+2?","type":"quiz"}}}`)
	message, err := bot.SendPoll(1, "2+2?", []string{"3", "4"}, PollConfig{Quiz: true, CorrectOptionIDs: []int{1}, Explanation: "Math", NotAnonymous: true, OpenPeriod: 60})
	if err != nil {
		t.Fatal(err)
	}
	if message.MessageID != 7 || message.Poll.ID != "p1" {
		t.Errorf("Wrong sent message: %+v", message)
	}
	checkRequest(t, tg, "sendPoll", `{"chat_id":1,"question":"2+2?","options":[{"text":"3"},{"text":"4"}],
		"type":"quiz","correct_option_ids":[1],"explanation":"Math","is_anonymous":false,"open_period":60}`)

	if _, err := bot.SendPoll(1, "Tea?", []string{"Yes", "No"}, PollConfig{}); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "sendPoll", `{"chat_id":1,"question":"Tea?","options":[{"text":"Yes"},{"text":"No"}]}`)

	tg.setResponse(200, `{"ok":true,"result":{"id":"p1","question":"Tea?","options":[{"text":"Yes","voter_count":3},{"text":"No","voter_count":1}],"total_voter_count":4,"is_closed":true}}`)
	poll, err := bot.StopPoll(1, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !poll.IsClosed || poll.TotalVoterCount != 4 || poll.Options[0].VoterCount != 3 {
		t.Errorf("Wrong poll: %+v", poll)
	}
	checkRequest(t, tg, "stopPoll", `{"chat_id":1,"message_id":7}`)
}

func TestInlineQueries(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	article := NewInlineArticle("1", "Cat", "Meow")
	article["description"] = "A cat"
	results := []InlineQueryResult{
		article,
		NewInlinePhoto("2", "https://example.com/cat.jpg", "https://example.com/thumb.jpg"),
		NewInlineCached("mpeg4_gif", "3", "file3", ""),
		NewInlineCached("document", "4", "file4", "Doc"),
	}
	if err := bot.AnswerInlineQuery("iq1", results, InlineQueryConfig{CacheTime: 10, IsPersonal: true, NextOffset: "20", ButtonText: "Settings", ButtonStartParameter: "settings"}); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerInlineQuery", `{"inline_query_id":"iq1","cache_time":10,"is_personal":true,"next_offset":"20",
		"button":{"text":"Settings","start_parameter":"settings"},"results":[
		{"type":"article","id":"1","title":"Cat","description":"A cat","input_message_content":{"message_text":"Meow"}},
		{"type":"photo","id":"2","photo_url":"https://example.com/cat.jpg","thumbnail_url":"https://example.com/thumb.jpg"},
		{"type":"mpeg4_gif","id":"3","mpeg4_file_id":"file3"},
		{"type":"document","id":"4","document_file_id":"file4","title":"Doc"}]}`)

	// no results must be sent as an empty list
	if err := bot.AnswerInlineQuery("iq2", nil, InlineQueryConfig{}); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerInlineQuery", `{"inline_query_id":"iq2","results":[]}`)

	if err := bot.AnswerGuestQuery("gq1", NewInlineArticle("1", "Answer", "42")); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerGuestQuery", `{"guest_query_id":"gq1","result":{"type":"article","id":"1","title":"Answer","input_message_content":{"message_text":"42"}}}`)
}

func TestPayments(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	invoice := InvoiceConfig{
		Title: "Coffee", Description: "Big cup", Payload: "order1", Currency: "XTR",
		Prices: []LabeledPrice{{Label: "Coffee", Amount: 50}}, StartParameter: "coffee", SubscriptionPeriod: 2592000,
		ReplyMarkup: &InlineKeyboard{Keyboard: []Row{{{Text: "Pay 50 XTR", Pay: true}}}},
	}
	if _, err := bot.SendInvoice(1, invoice); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "sendInvoice", `{"chat_id":1,"title":"Coffee","description":"Big cup","payload":"order1","currency":"XTR",
		"prices":[{"label":"Coffee","amount":50}],"start_parameter":"coffee","reply_markup":{"inline_keyboard":[[{"text":"Pay 50 XTR","pay":true}]]}}`)

	tg.setResponse(200, `{"ok":true,"result":"https://t.me/$invoice"}`)
	link, err := bot.CreateInvoiceLink(invoice)
	if err != nil || link != "https://t.me/$invoice" {
		t.Fatalf("Wrong invoice link: %q %v", link, err)
	}
	checkRequest(t, tg, "createInvoiceLink", `{"title":"Coffee","description":"Big cup","payload":"order1","currency":"XTR",
		"prices":[{"label":"Coffee","amount":50}],"subscription_period":2592000}`)

	tg.setResponse(200, `{"ok":true,"result":true}`)
	options := []ShippingOption{{ID: "post", Title: "Post", Prices: []LabeledPrice{{Label: "Delivery", Amount: 300}}}}
	if err := bot.AnswerShippingQuery("sq1", options, ""); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerShippingQuery", `{"shipping_query_id":"sq1","ok":true,"shipping_options":[{"id":"post","title":"Post","prices":[{"label":"Delivery","amount":300}]}]}`)
	if err := bot.AnswerShippingQuery("sq2", nil, "No delivery to Mars"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerShippingQuery", `{"shipping_query_id":"sq2","ok":false,"error_message":"No delivery to Mars"}`)

	if err := bot.AnswerPreCheckoutQuery("pq1", ""); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerPreCheckoutQuery", `{"pre_checkout_query_id":"pq1","ok":true}`)
	if err := bot.AnswerPreCheckoutQuery("pq2", "Out of stock"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "answerPreCheckoutQuery", `{"pre_checkout_query_id":"pq2","ok":false,"error_message":"Out of stock"}`)

	if err := bot.RefundStarPayment(5, "ch1"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "refundStarPayment", `{"user_id":5,"telegram_payment_charge_id":"ch1"}`)
	if err := bot.EditUserStarSubscription(5, "ch1", true); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "editUserStarSubscription", `{"user_id":5,"telegram_payment_charge_id":"ch1","is_canceled":true}`)
}

func TestGames(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	if _, err := bot.SendGame(1, "race"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "sendGame", `{"chat_id":1,"game_short_name":"race"}`)

	if err := bot.SetGameScore(5, 100, GameMessage{ChatID: 1, MessageID: 7}, true); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "setGameScore", `{"user_id":5,"score":100,"force":true,"chat_id":1,"message_id":7}`)
	if err := bot.SetGameScore(5, 100, GameMessage{InlineMessageID: "im1"}, false); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "setGameScore", `{"user_id":5,"score":100,"inline_message_id":"im1"}`)

	tg.setResponse(200, `{"ok":true,"result":[{"position":1,"user":{"id":5,"first_name":"Ann"},"score":100},{"position":2,"user":{"id":6},"score":90}]}`)
	scores, err := bot.GetGameHighScores(5, GameMessage{ChatID: 1, MessageID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(scores) != 2 || scores[0].User.FirstName != "Ann" || scores[1].Score != 90 {
		t.Errorf("Wrong scores: %+v", scores)
	}
	checkRequest(t, tg, "getGameHighScores", `{"user_id":5,"chat_id":1,"message_id":7}`)
}

func TestReactionsAndChats(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	if err := bot.SetMessageReaction(1, 7, "👍"); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "setMessageReaction", `{"chat_id":1,"message_id":7,"reaction":[{"type":"emoji","emoji":"👍"}]}`)
	if err := bot.SetMessageReaction(1, 7, ""); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "setMessageReaction", `{"chat_id":1,"message_id":7,"reaction":[]}`)

	if err := bot.ApproveChatJoinRequest(-100, 5); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "approveChatJoinRequest", `{"chat_id":-100,"user_id":5}`)
	if err := bot.DeclineChatJoinRequest(-100, 6); err != nil {
		t.Fatal(err)
	}
	checkRequest(t, tg, "declineChatJoinRequest", `{"chat_id":-100,"user_id":6}`)

	tg.setResponse(200, `{"ok":true,"result":"777:MANAGED"}`)
	if token, err := bot.GetManagedBotToken(777); err != nil || token != "777:MANAGED" {
		t.Errorf("Wrong token: %q %v", token, err)
	}
	checkRequest(t, tg, "getManagedBotToken", `{"user_id":777}`)
}

func TestCall(t *testing.T) {
	bot, tg := newTestBot(t, LaserTeleConfigT{})
	tg.setResponse(200, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"LaserBot"}}`)
	result, err := bot.Call("getMe", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	var me User
	if err := json.Unmarshal(result, &me); err != nil || me.FirstName != "LaserBot" {
		t.Errorf("Wrong result: %s %v", result, err)
	}
	if tg.lastRequest(t).Method != "getMe" {
		t.Errorf("Wrong method: %s", tg.lastRequest(t).Method)
	}

	tg.setResponse(429, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 5","parameters":{"retry_after":5}}`)
	_, err = bot.Call("sendDice", map[string]interface{}{"chat_id": 1})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != 429 || apiErr.RetryAfter != 5 {
		t.Errorf("Expected APIError with RetryAfter 5, but got %v", err)
	}
}

func TestJSONLogWithoutContent(t *testing.T) {
	bot, _ := newTestBot(t, LaserTeleConfigT{})
	if _, err := bot.SendPoll(137, "secret question", []string{"secret option", "B"}, PollConfig{}); err != nil {
		t.Fatal(err)
	}
	log := readLog(t, bot, "polls")
	if strings.Contains(log, "secret") || !strings.Contains(log, "REQV: POST sendPoll chat_id=137") || !strings.Contains(log, "RESP: 200 OK") {
		t.Errorf("Wrong log: %s", log)
	}
	if strings.Contains(log, testAPIKEY) {
		t.Errorf("Log contains APIKEY")
	}
}

func TestPackageLevelFeatures(t *testing.T) {
	DoLaserTeleInit(LaserTeleConfigT{APIKEY: testAPIKEY, Timeout: time.Second, LogMode: LogOff})
	defer func() { defaultBot, APIKEY = nil, "" }()
	tg := newFakeTelegram(testAPIKEY)
	connectToFake(t, defaultBot, tg)

	calls := map[string]func() error{
		"sendMessage":              func() error { _, err := SendMessageWithConfig(1, "text", MessageConfig{}); return err },
		"editMessageText":          func() error { return EditMessageText(1, 2, "text") },
		"answerCallbackQuery":      func() error { return AnswerCallbackQueryWithConfig("cq1", CallbackAnswerConfig{}) },
		"getMe":                    func() error { _, err := Call("getMe", nil); return err },
		"sendPoll":                 func() error { _, err := SendPoll(1, "Q", []string{"A", "B"}, PollConfig{}); return err },
		"stopPoll":                 func() error { _, err := StopPoll(1, 2); return err },
		"answerInlineQuery":        func() error { return AnswerInlineQuery("iq1", nil, InlineQueryConfig{}) },
		"answerGuestQuery":         func() error { return AnswerGuestQuery("gq1", NewInlineArticle("1", "T", "M")) },
		"sendInvoice":              func() error { _, err := SendInvoice(1, InvoiceConfig{}); return err },
		"answerShippingQuery":      func() error { return AnswerShippingQuery("sq1", nil, "") },
		"answerPreCheckoutQuery":   func() error { return AnswerPreCheckoutQuery("pq1", "") },
		"refundStarPayment":        func() error { return RefundStarPayment(1, "ch1") },
		"editUserStarSubscription": func() error { return EditUserStarSubscription(1, "ch1", false) },
		"sendGame":                 func() error { _, err := SendGame(1, "race"); return err },
		"setGameScore":             func() error { return SetGameScore(1, 10, GameMessage{ChatID: 1, MessageID: 2}, false) },
		"setMessageReaction":       func() error { return SetMessageReaction(1, 2, "👍") },
		"approveChatJoinRequest":   func() error { return ApproveChatJoinRequest(1, 2) },
		"declineChatJoinRequest":   func() error { return DeclineChatJoinRequest(1, 2) },
	}
	for method, call := range calls {
		if err := call(); err != nil {
			t.Errorf("%s failed: %v", method, err)
		} else if got := tg.lastRequest(t).Method; got != method {
			t.Errorf("Expected method %s, but got %s", method, got)
		}
	}

	tg.setResponse(200, `{"ok":true,"result":"text"}`)
	if link, err := CreateInvoiceLink(InvoiceConfig{}); err != nil || link != "text" || tg.lastRequest(t).Method != "createInvoiceLink" {
		t.Errorf("CreateInvoiceLink failed: %q %v", link, err)
	}
	if token, err := GetManagedBotToken(1); err != nil || token != "text" || tg.lastRequest(t).Method != "getManagedBotToken" {
		t.Errorf("GetManagedBotToken failed: %q %v", token, err)
	}
	tg.setResponse(200, `{"ok":true,"result":[]}`)
	if _, err := GetGameHighScores(1, GameMessage{InlineMessageID: "im1"}); err != nil || tg.lastRequest(t).Method != "getGameHighScores" {
		t.Errorf("GetGameHighScores failed: %v", err)
	}
}
