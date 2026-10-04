package laser_tele_api

import "encoding/json"

// MessageConfig contains optional parameters of a message for SendMessageWithConfig
type MessageConfig struct {
	ParseMode           string // "HTML", "MarkdownV2" or "Markdown"
	ReplyToMessageID    int    // the message is a reply to this message
	MessageThreadID     int    // topic in a forum supergroup
	DisableNotification bool
	ProtectContent      bool // the message can't be forwarded and saved
	// BusinessConnectionID (Message.BusinessConnectionID) sends the message on behalf of a connected business account
	BusinessConnectionID string
	ReplyMarkup          *InlineKeyboard
}

// SendMessageWithConfig sends message with optional parameters and returns the sent message
func (b *Bot) SendMessageWithConfig(chatID int, text string, config MessageConfig) (Message, error) {
	params := map[string]interface{}{"chat_id": chatID, "text": text}
	if config.ParseMode != "" {
		params["parse_mode"] = config.ParseMode
	}
	if config.ReplyToMessageID != 0 {
		params["reply_parameters"] = map[string]interface{}{"message_id": config.ReplyToMessageID}
	}
	if config.MessageThreadID != 0 {
		params["message_thread_id"] = config.MessageThreadID
	}
	if config.DisableNotification {
		params["disable_notification"] = true
	}
	if config.ProtectContent {
		params["protect_content"] = true
	}
	if config.BusinessConnectionID != "" {
		params["business_connection_id"] = config.BusinessConnectionID
	}
	if config.ReplyMarkup != nil {
		params["reply_markup"] = config.ReplyMarkup
	}
	var message Message
	err := b.callInto("sendMessage", "sendMessage", params, &message)
	return message, err
}

// EditMessageText changes text of a message sent by the bot
func (b *Bot) EditMessageText(chatID, messageID int, text string) error {
	params := map[string]interface{}{"chat_id": chatID, "message_id": messageID, "text": text}
	return b.callInto("editMessage", "editMessageText", params, nil)
}

// CallbackAnswerConfig contains optional parameters of AnswerCallbackQueryWithConfig
type CallbackAnswerConfig struct {
	Text      string // notification for the user
	ShowAlert bool   // show the text as an alert instead of a notification at the top of the chat
	URL       string // URL opened by the user's client, e.g. the game for the "Play" button of a game
	CacheTime int    // seconds the result may be cached on the client
}

// AnswerCallbackQueryWithConfig answers the pressed inline button like AnswerCallbackQuery, with optional parameters
func (b *Bot) AnswerCallbackQueryWithConfig(callbackQueryID string, config CallbackAnswerConfig) error {
	params := map[string]interface{}{"callback_query_id": callbackQueryID}
	if config.Text != "" {
		params["text"] = config.Text
	}
	if config.ShowAlert {
		params["show_alert"] = true
	}
	if config.URL != "" {
		params["url"] = config.URL
	}
	if config.CacheTime > 0 {
		params["cache_time"] = config.CacheTime
	}
	return b.callInto("answerCallback", "answerCallbackQuery", params, nil)
}

// SendMessageWithConfig sends message with optional parameters and returns the sent message
func SendMessageWithConfig(chatID int, text string, config MessageConfig) (Message, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return Message{}, err
	}
	return bot.SendMessageWithConfig(chatID, text, config)
}

// EditMessageText changes text of a message sent by the bot
func EditMessageText(chatID, messageID int, text string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.EditMessageText(chatID, messageID, text)
}

// AnswerCallbackQueryWithConfig answers the pressed inline button like AnswerCallbackQuery, with optional parameters
func AnswerCallbackQueryWithConfig(callbackQueryID string, config CallbackAnswerConfig) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerCallbackQueryWithConfig(callbackQueryID, config)
}

// Call calls any Bot API method (https://core.telegram.org/bots/api#available-methods) with params
// (a struct or a map, sent as JSON) and returns the "result" field of the response.
// Use it for methods that have no own function in the library
func Call(method string, params interface{}) (json.RawMessage, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return nil, err
	}
	return bot.Call(method, params)
}
