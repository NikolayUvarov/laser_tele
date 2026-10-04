package laser_tele_api

// InlineQuery is sent when the user typed "@your_bot query" in any chat.
// Inline mode must be enabled for the bot with /setinline in @BotFather
type InlineQuery struct {
	ID       string    `json:"id"`
	From     User      `json:"from"`
	Query    string    `json:"query"`
	Offset   string    `json:"offset"`    // NextOffset of the previous answer, for paging
	ChatType string    `json:"chat_type"` // "sender", "private", "group", "supergroup" or "channel"
	Location *Location `json:"location"`
}

// ChosenInlineResult is sent when the user chose a result of an inline query.
// These updates must be enabled for the bot with /setinlinefeedback in @BotFather
type ChosenInlineResult struct {
	ResultID        string    `json:"result_id"`
	From            User      `json:"from"`
	Location        *Location `json:"location"`
	InlineMessageID string    `json:"inline_message_id"` // set if the message has an inline keyboard
	Query           string    `json:"query"`
}

// InlineQueryResult is one result for AnswerInlineQuery or AnswerGuestQuery.
// Create it with NewInlineArticle, NewInlinePhoto or NewInlineCached and add optional fields,
// e.g. result["description"] = "...", or fill the fields of any result type from
// https://core.telegram.org/bots/api#inlinequeryresult
type InlineQueryResult map[string]interface{}

// NewInlineArticle creates a result, which sends a text message
func NewInlineArticle(id, title, messageText string) InlineQueryResult {
	return InlineQueryResult{
		"type":                  "article",
		"id":                    id,
		"title":                 title,
		"input_message_content": map[string]interface{}{"message_text": messageText},
	}
}

// NewInlinePhoto creates a result, which sends a JPEG photo by URL
func NewInlinePhoto(id, photoURL, thumbnailURL string) InlineQueryResult {
	return InlineQueryResult{"type": "photo", "id": id, "photo_url": photoURL, "thumbnail_url": thumbnailURL}
}

// NewInlineCached creates a result, which sends a file already stored on Telegram servers by its file_id.
// fileType is "photo", "gif", "mpeg4_gif", "sticker", "document", "video", "voice" or "audio",
// title is required for "document", "video" and "voice"
func NewInlineCached(fileType, id, fileID, title string) InlineQueryResult {
	field := fileType + "_file_id"
	if fileType == "mpeg4_gif" {
		field = "mpeg4_file_id"
	}
	result := InlineQueryResult{"type": fileType, "id": id, field: fileID}
	if title != "" {
		result["title"] = title
	}
	return result
}

// InlineQueryConfig contains optional parameters of AnswerInlineQuery
type InlineQueryConfig struct {
	CacheTime  int    // seconds the results may be cached on the server, default 300
	IsPersonal bool   // results are cached only for the user, who sent the query
	NextOffset string // passed in the next query of the user (InlineQuery.Offset) to get more results
	// ButtonText and ButtonStartParameter show a button above the results, which starts a private chat with the bot
	ButtonText           string
	ButtonStartParameter string
}

// AnswerInlineQuery sends up to 50 results for the inline query
func (b *Bot) AnswerInlineQuery(inlineQueryID string, results []InlineQueryResult, config InlineQueryConfig) error {
	if results == nil {
		results = []InlineQueryResult{}
	}
	params := map[string]interface{}{"inline_query_id": inlineQueryID, "results": results}
	if config.CacheTime > 0 {
		params["cache_time"] = config.CacheTime
	}
	if config.IsPersonal {
		params["is_personal"] = true
	}
	if config.NextOffset != "" {
		params["next_offset"] = config.NextOffset
	}
	if config.ButtonText != "" {
		params["button"] = map[string]interface{}{"text": config.ButtonText, "start_parameter": config.ButtonStartParameter}
	}
	return b.callInto("answerInline", "answerInlineQuery", params, nil)
}

// AnswerGuestQuery answers a guest message (Message.GuestQueryID) with result
func (b *Bot) AnswerGuestQuery(guestQueryID string, result InlineQueryResult) error {
	params := map[string]interface{}{"guest_query_id": guestQueryID, "result": result}
	return b.callInto("answerInline", "answerGuestQuery", params, nil)
}

// AnswerInlineQuery sends up to 50 results for the inline query
func AnswerInlineQuery(inlineQueryID string, results []InlineQueryResult, config InlineQueryConfig) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerInlineQuery(inlineQueryID, results, config)
}

// AnswerGuestQuery answers a guest message (Message.GuestQueryID) with result
func AnswerGuestQuery(guestQueryID string, result InlineQueryResult) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerGuestQuery(guestQueryID, result)
}
