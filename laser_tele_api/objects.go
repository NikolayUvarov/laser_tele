package laser_tele_api

import (
	"fmt"
	"time"
)

// Telegram Bot API objects (https://core.telegram.org/bots/api#available-types).
// Optional objects are values: if the object is absent, its fields are empty,
// e.g. a message has a video if Video.FileID != "".
// Only ReplyToMessage and Location are pointers and are nil if absent.

type User struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`  // "private", "group", "supergroup" or "channel"
	Title     string `json:"title"` // for groups and channels
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsForum   bool   `json:"is_forum"`
}

// MessageEntity is a special part of the text: command, link, bold text...
// Offset and Length are in UTF-16 code units, not in bytes
type MessageEntity struct {
	Type          string `json:"type"` // "bot_command", "mention", "url", "bold", "text_link"...
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url"`  // for "text_link"
	User          User   `json:"user"` // for "text_mention"
	Language      string `json:"language"`
	CustomEmojiID string `json:"custom_emoji_id"`
}

type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
}

type Video struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

type Animation struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Duration     int    `json:"duration"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

type Audio struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	Performer    string `json:"performer"`
	Title        string `json:"title"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

type Document struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

type VideoNote struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Length       int    `json:"length"`
	Duration     int    `json:"duration"`
	FileSize     int    `json:"file_size"`
}

type Sticker struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Type         string `json:"type"` // "regular", "mask" or "custom_emoji"
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	IsAnimated   bool   `json:"is_animated"`
	IsVideo      bool   `json:"is_video"`
	Emoji        string `json:"emoji"`
	SetName      string `json:"set_name"`
	FileSize     int    `json:"file_size"`
}

type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserID      int    `json:"user_id"`
	VCard       string `json:"vcard"`
}

type Location struct {
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	HorizontalAccuracy float64 `json:"horizontal_accuracy"`
	LivePeriod         int     `json:"live_period"`
}

// MessageOrigin describes the origin of a forwarded message
type MessageOrigin struct {
	Type            string `json:"type"` // "user", "hidden_user", "chat" or "channel"
	Date            int    `json:"date"`
	SenderUser      User   `json:"sender_user"`      // for "user"
	SenderUserName  string `json:"sender_user_name"` // for "hidden_user"
	SenderChat      Chat   `json:"sender_chat"`      // for "chat"
	Chat            Chat   `json:"chat"`             // for "channel"
	MessageID       int    `json:"message_id"`       // for "channel"
	AuthorSignature string `json:"author_signature"`
}

// ReplyMarkup is the inline keyboard attached to a message
type ReplyMarkup struct {
	InlineKeyboard []Row `json:"inline_keyboard"`
}

type Message struct {
	MessageID       int             `json:"message_id"`
	MessageThreadID int             `json:"message_thread_id"` // topic in forum supergroups
	From            User            `json:"from"`              // empty for messages in channels
	SenderChat      Chat            `json:"sender_chat"`       // for messages sent on behalf of a chat
	Chat            Chat            `json:"chat"`
	Date            int             `json:"date"`
	EditDate        int             `json:"edit_date"`
	ForwardOrigin   MessageOrigin   `json:"forward_origin"`   // ForwardOrigin.Type != "" for forwarded messages
	ReplyToMessage  *Message        `json:"reply_to_message"` // nil if the message is not a reply
	MediaGroupID    string          `json:"media_group_id"`   // the same for photos and videos sent as an album
	Text            string          `json:"text"`
	Entities        []MessageEntity `json:"entities"`
	Caption         string          `json:"caption"` // text of a photo, video, document...
	CaptionEntities []MessageEntity `json:"caption_entities"`
	Photo           []PhotoSize     `json:"photo"` // sizes of the photo, the last one is the largest
	Video           Video           `json:"video"`
	Animation       Animation       `json:"animation"`
	Audio           Audio           `json:"audio"`
	Document        Document        `json:"document"`
	Voice           Voice           `json:"voice"`
	VideoNote       VideoNote       `json:"video_note"`
	Sticker         Sticker         `json:"sticker"`
	Contact         Contact         `json:"contact"`
	Location        *Location       `json:"location"` // nil if the message has no location
	NewChatMembers  []User          `json:"new_chat_members"`
	LeftChatMember  User            `json:"left_chat_member"`
	ReplyMarkup     ReplyMarkup     `json:"reply_markup"`
}

// CallbackQuery is sent when the user pressed an inline button.
// It must be answered with AnswerCallbackQuery, otherwise Telegram shows progress on the button
type CallbackQuery struct {
	ID              string  `json:"id"`
	From            User    `json:"from"`
	Message         Message `json:"message"` // message with the pressed button
	InlineMessageID string  `json:"inline_message_id"`
	ChatInstance    string  `json:"chat_instance"`
	Data            string  `json:"data"` // CallbackData of the pressed button
}

type ChatMember struct {
	Status string `json:"status"` // "creator", "administrator", "member", "restricted", "left" or "kicked"
	User   User   `json:"user"`
}

// ChatMemberUpdated is sent when status of the bot in a chat is changed,
// e.g. the user blocked the bot (NewChatMember.Status == "kicked") or the bot was added to a group
type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	Date          int        `json:"date"`
	OldChatMember ChatMember `json:"old_chat_member"`
	NewChatMember ChatMember `json:"new_chat_member"`
}

type Update struct {
	UpdateID          int               `json:"update_id"`
	UpdateMessage     Message           `json:"message"`
	EditedMessage     Message           `json:"edited_message"`
	ChannelPost       Message           `json:"channel_post"`
	EditedChannelPost Message           `json:"edited_channel_post"`
	CallbackQuery     CallbackQuery     `json:"callback_query"`
	MyChatMember      ChatMemberUpdated `json:"my_chat_member"`
}

// Type returns the kind of the update: "message", "edited_message", "channel_post",
// "edited_channel_post", "callback_query", "my_chat_member" or "" for other kinds
func (u Update) Type() string {
	switch {
	case u.UpdateMessage.MessageID != 0:
		return "message"
	case u.EditedMessage.MessageID != 0:
		return "edited_message"
	case u.ChannelPost.MessageID != 0:
		return "channel_post"
	case u.EditedChannelPost.MessageID != 0:
		return "edited_channel_post"
	case u.CallbackQuery.ID != "":
		return "callback_query"
	case u.MyChatMember.Date != 0:
		return "my_chat_member"
	}
	return ""
}

// Old names of the types, kept for compatibility
type UpdateMessageT = Message
type UpdateCallBackQueryT = CallbackQuery

type UpdateJSON struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type File struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	} `json:"result"`
}
type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"` // link opened by the button instead of sending CallbackData
}
type Row []Button
type InlineKeyboard struct {
	Keyboard []Row `json:"inline_keyboard"`
}

type LaserTeleConfigT struct {
	APIKEY  string        // bot token, if empty it is taken from TG_API_KEY env or .APIKEY file
	Timeout time.Duration // interval between requests of updates, if 0 it is taken from TIMEOUT env (default 10s)
	// CallbackOnUpdate (optional) is called for every new update,
	// in addition to the callback passed to LaserTeleRun or Run
	CallbackOnUpdate Callback
	LogMode          LogMode // what is written to log files, default LogWithoutContent
	LogDir           string  // directory of log files, default current directory
	// LogMaxSize is max size of a log file in bytes (default 10 MB).
	// The larger file is renamed to <name>.log.1, the previous <name>.log.1 is deleted
	LogMaxSize  int64
	DownloadDir string // directory for files downloaded by LoadFile, default "downloadedFiles"
}

// APIError is returned when Telegram refused the request,
// e.g. ErrorCode 403 if the bot was blocked by the user
type APIError struct {
	Method      string
	ErrorCode   int
	Description string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.ErrorCode, e.Description)
}

// common part of all Bot API responses
type apiResponse struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func AddButton(text, callback string) Button {
	var newButton Button
	newButton.Text = text
	newButton.CallbackData = callback

	return newButton

}
