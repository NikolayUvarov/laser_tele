package laser_tele_api

import (
	"encoding/json"
	"fmt"
	"time"
)

// Telegram Bot API objects (https://core.telegram.org/bots/api#available-types).
// Optional objects are values: if the object is absent, its fields are empty,
// e.g. a message has a video if Video.FileID != "".
// Only ReplyToMessage, PinnedMessage and Location are pointers and are nil if absent.
// Fields not described here can be read from Update.Raw.

// User is a Telegram user or bot
type User struct {
	ID           int    `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// Chat is a private chat, group, supergroup or channel
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

// PhotoSize is one size of a photo or a thumbnail
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int    `json:"file_size"`
}

// Video is a video file
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

// Animation is a GIF or H.264/MPEG-4 AVC video without sound
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

// Audio is a music file
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

// Document is a general file
type Document struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileName     string `json:"file_name"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

// Voice is a voice note
type Voice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type"`
	FileSize     int    `json:"file_size"`
}

// VideoNote is a round video message
type VideoNote struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Length       int    `json:"length"`
	Duration     int    `json:"duration"`
	FileSize     int    `json:"file_size"`
}

// Sticker is a sticker
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

// Contact is a phone contact
type Contact struct {
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserID      int    `json:"user_id"`
	VCard       string `json:"vcard"`
}

// Location is a point on the map
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

// Venue is a place with a name and an address
type Venue struct {
	Location        Location `json:"location"`
	Title           string   `json:"title"`
	Address         string   `json:"address"`
	FoursquareID    string   `json:"foursquare_id"`
	FoursquareType  string   `json:"foursquare_type"`
	GooglePlaceID   string   `json:"google_place_id"`
	GooglePlaceType string   `json:"google_place_type"`
}

// Dice is an animated emoji with a random value
type Dice struct {
	Emoji string `json:"emoji"`
	Value int    `json:"value"`
}

// WebAppData is data sent by a Web App to the bot
type WebAppData struct {
	Data       string `json:"data"`
	ButtonText string `json:"button_text"`
}

// ReplyMarkup is the inline keyboard attached to a message
type ReplyMarkup struct {
	InlineKeyboard []Row `json:"inline_keyboard"`
}

// Message is a message of any kind: text, photo, poll, payment, service message...
type Message struct {
	MessageID       int  `json:"message_id"`
	MessageThreadID int  `json:"message_thread_id"` // topic in forum supergroups
	From            User `json:"from"`              // empty for messages in channels
	SenderChat      Chat `json:"sender_chat"`       // for messages sent on behalf of a chat
	Chat            Chat `json:"chat"`
	Date            int  `json:"date"`
	EditDate        int  `json:"edit_date"`
	// BusinessConnectionID is set for messages of a connected business account,
	// pass it to SendMessageWithConfig to answer on behalf of the account
	BusinessConnectionID string `json:"business_connection_id"`
	// GuestQueryID is set for guest messages, answer them with AnswerGuestQuery
	GuestQueryID        string          `json:"guest_query_id"`
	ForwardOrigin       MessageOrigin   `json:"forward_origin"`   // ForwardOrigin.Type != "" for forwarded messages
	ReplyToMessage      *Message        `json:"reply_to_message"` // nil if the message is not a reply
	ViaBot              User            `json:"via_bot"`          // bot, via which the message was sent in inline mode
	IsTopicMessage      bool            `json:"is_topic_message"`
	HasProtectedContent bool            `json:"has_protected_content"`
	MediaGroupID        string          `json:"media_group_id"` // the same for photos and videos sent as an album
	AuthorSignature     string          `json:"author_signature"`
	EffectID            string          `json:"effect_id"`
	Text                string          `json:"text"`
	Entities            []MessageEntity `json:"entities"`
	Caption             string          `json:"caption"` // text of a photo, video, document...
	CaptionEntities     []MessageEntity `json:"caption_entities"`
	Photo               []PhotoSize     `json:"photo"` // sizes of the photo, the last one is the largest
	Video               Video           `json:"video"`
	Animation           Animation       `json:"animation"`
	Audio               Audio           `json:"audio"`
	Document            Document        `json:"document"`
	Voice               Voice           `json:"voice"`
	VideoNote           VideoNote       `json:"video_note"`
	Sticker             Sticker         `json:"sticker"`
	Contact             Contact         `json:"contact"`
	Dice                Dice            `json:"dice"`
	Game                Game            `json:"game"` // Game.Title != "" for a game
	Poll                Poll            `json:"poll"` // Poll.ID != "" for a poll
	Venue               Venue           `json:"venue"`
	Location            *Location       `json:"location"` // nil if the message has no location
	NewChatMembers      []User          `json:"new_chat_members"`
	LeftChatMember      User            `json:"left_chat_member"`
	NewChatTitle        string          `json:"new_chat_title"`
	NewChatPhoto        []PhotoSize     `json:"new_chat_photo"`
	DeleteChatPhoto     bool            `json:"delete_chat_photo"`
	// the group is upgraded to a supergroup with MigrateToChatID, use it instead of Chat.ID
	MigrateToChatID   int      `json:"migrate_to_chat_id"`
	MigrateFromChatID int      `json:"migrate_from_chat_id"`
	PinnedMessage     *Message `json:"pinned_message"`
	// Invoice.Currency != "" for an invoice
	Invoice Invoice `json:"invoice"`
	// SuccessfulPayment.Currency != "" when the user paid an invoice, deliver the goods after it
	SuccessfulPayment SuccessfulPayment `json:"successful_payment"`
	RefundedPayment   RefundedPayment   `json:"refunded_payment"`
	WebAppData        WebAppData        `json:"web_app_data"`
	ConnectedWebsite  string            `json:"connected_website"`
	ReplyMarkup       ReplyMarkup       `json:"reply_markup"`
}

// CallbackQuery is sent when the user pressed an inline button.
// It must be answered with AnswerCallbackQuery, otherwise Telegram shows progress on the button
type CallbackQuery struct {
	ID              string  `json:"id"`
	From            User    `json:"from"`
	Message         Message `json:"message"` // message with the pressed button
	InlineMessageID string  `json:"inline_message_id"`
	ChatInstance    string  `json:"chat_instance"`
	Data            string  `json:"data"`            // CallbackData of the pressed button
	GameShortName   string  `json:"game_short_name"` // the "Play" button of a game was pressed
}

// ChatMember is a member of a chat; fields of other statuses are empty
type ChatMember struct {
	Status             string `json:"status"` // "creator", "administrator", "member", "restricted", "left" or "kicked"
	User               User   `json:"user"`
	IsAnonymous        bool   `json:"is_anonymous"`
	CustomTitle        string `json:"custom_title"`
	UntilDate          int    `json:"until_date"`
	IsMember           bool   `json:"is_member"` // for "restricted"
	CanBeEdited        bool   `json:"can_be_edited"`
	CanManageChat      bool   `json:"can_manage_chat"`
	CanDeleteMessages  bool   `json:"can_delete_messages"`
	CanRestrictMembers bool   `json:"can_restrict_members"`
	CanPromoteMembers  bool   `json:"can_promote_members"`
	CanChangeInfo      bool   `json:"can_change_info"`
	CanInviteUsers     bool   `json:"can_invite_users"`
	CanPinMessages     bool   `json:"can_pin_messages"`
	CanPostMessages    bool   `json:"can_post_messages"`
	CanEditMessages    bool   `json:"can_edit_messages"`
	CanSendMessages    bool   `json:"can_send_messages"`
}

// ChatMemberUpdated is sent when status of a member in a chat is changed,
// e.g. the user blocked the bot (NewChatMember.Status == "kicked") or the bot was added to a group
type ChatMemberUpdated struct {
	Chat                    Chat           `json:"chat"`
	From                    User           `json:"from"`
	Date                    int            `json:"date"`
	OldChatMember           ChatMember     `json:"old_chat_member"`
	NewChatMember           ChatMember     `json:"new_chat_member"`
	InviteLink              ChatInviteLink `json:"invite_link"`
	ViaJoinRequest          bool           `json:"via_join_request"`
	ViaChatFolderInviteLink bool           `json:"via_chat_folder_invite_link"`
}

// AllUpdateTypes are all kinds of updates, the bot receives them by default (LaserTeleConfigT.AllowedUpdates)
var AllUpdateTypes = []string{
	"message", "edited_message", "channel_post", "edited_channel_post",
	"business_connection", "business_message", "edited_business_message", "deleted_business_messages",
	"guest_message", "message_reaction", "message_reaction_count", "inline_query", "chosen_inline_result",
	"callback_query", "shipping_query", "pre_checkout_query", "purchased_paid_media", "poll", "poll_answer",
	"my_chat_member", "chat_member", "chat_join_request", "chat_boost", "removed_chat_boost",
	"managed_bot", "subscription", "stopped_message_generation",
}

// Update is an incoming update. Only the field of its kind (see Type) is filled
type Update struct {
	UpdateID                 int                         `json:"update_id"`
	UpdateMessage            Message                     `json:"message"`
	EditedMessage            Message                     `json:"edited_message"`
	ChannelPost              Message                     `json:"channel_post"`
	EditedChannelPost        Message                     `json:"edited_channel_post"`
	BusinessConnection       BusinessConnection          `json:"business_connection"`
	BusinessMessage          Message                     `json:"business_message"`
	EditedBusinessMessage    Message                     `json:"edited_business_message"`
	DeletedBusinessMessages  BusinessMessagesDeleted     `json:"deleted_business_messages"`
	GuestMessage             Message                     `json:"guest_message"`
	MessageReaction          MessageReactionUpdated      `json:"message_reaction"`
	MessageReactionCount     MessageReactionCountUpdated `json:"message_reaction_count"`
	InlineQuery              InlineQuery                 `json:"inline_query"`
	ChosenInlineResult       ChosenInlineResult          `json:"chosen_inline_result"`
	CallbackQuery            CallbackQuery               `json:"callback_query"`
	ShippingQuery            ShippingQuery               `json:"shipping_query"`
	PreCheckoutQuery         PreCheckoutQuery            `json:"pre_checkout_query"`
	PurchasedPaidMedia       PaidMediaPurchased          `json:"purchased_paid_media"`
	Poll                     Poll                        `json:"poll"`
	PollAnswer               PollAnswer                  `json:"poll_answer"`
	MyChatMember             ChatMemberUpdated           `json:"my_chat_member"`
	ChatMember               ChatMemberUpdated           `json:"chat_member"`
	ChatJoinRequest          ChatJoinRequest             `json:"chat_join_request"`
	ChatBoost                ChatBoostUpdated            `json:"chat_boost"`
	RemovedChatBoost         ChatBoostRemoved            `json:"removed_chat_boost"`
	ManagedBot               ManagedBotUpdated           `json:"managed_bot"`
	Subscription             BotSubscriptionUpdated      `json:"subscription"`
	StoppedMessageGeneration MessageGenerationStopped    `json:"stopped_message_generation"`

	// Raw is the update as received from Telegram, for fields not described in the library
	Raw json.RawMessage `json:"-"`

	kind string
}

// UnmarshalJSON decodes the update and remembers its kind (Type) and the received JSON (Raw)
func (u *Update) UnmarshalJSON(data []byte) error {
	type plainUpdate Update // without methods, so this function is not called recursively
	if err := json.Unmarshal(data, (*plainUpdate)(u)); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	u.kind = ""
	for name := range fields {
		if name != "update_id" {
			u.kind = name
		}
	}
	u.Raw = append(json.RawMessage(nil), data...)
	return nil
}

// Type returns the kind of the update, the name of its field in Telegram Bot API:
// "message", "callback_query", "poll", "inline_query", "pre_checkout_query"... (see AllUpdateTypes)
func (u Update) Type() string {
	if u.kind != "" {
		return u.kind
	}
	// the update is not received from Telegram, but created by the application
	kinds := []struct {
		name   string
		filled bool
	}{
		{"message", u.UpdateMessage.MessageID != 0},
		{"edited_message", u.EditedMessage.MessageID != 0},
		{"channel_post", u.ChannelPost.MessageID != 0},
		{"edited_channel_post", u.EditedChannelPost.MessageID != 0},
		{"business_connection", u.BusinessConnection.ID != ""},
		{"business_message", u.BusinessMessage.MessageID != 0},
		{"edited_business_message", u.EditedBusinessMessage.MessageID != 0},
		{"deleted_business_messages", u.DeletedBusinessMessages.BusinessConnectionID != ""},
		{"guest_message", u.GuestMessage.MessageID != 0},
		{"message_reaction", u.MessageReaction.MessageID != 0},
		{"message_reaction_count", u.MessageReactionCount.MessageID != 0},
		{"inline_query", u.InlineQuery.ID != ""},
		{"chosen_inline_result", u.ChosenInlineResult.ResultID != ""},
		{"callback_query", u.CallbackQuery.ID != ""},
		{"shipping_query", u.ShippingQuery.ID != ""},
		{"pre_checkout_query", u.PreCheckoutQuery.ID != ""},
		{"purchased_paid_media", u.PurchasedPaidMedia.From.ID != 0},
		{"poll", u.Poll.ID != ""},
		{"poll_answer", u.PollAnswer.PollID != ""},
		{"my_chat_member", u.MyChatMember.Date != 0},
		{"chat_member", u.ChatMember.Date != 0},
		{"chat_join_request", u.ChatJoinRequest.Date != 0},
		{"chat_boost", u.ChatBoost.Boost.BoostID != ""},
		{"removed_chat_boost", u.RemovedChatBoost.BoostID != ""},
		{"managed_bot", u.ManagedBot.Bot.ID != 0},
		{"subscription", u.Subscription.User.ID != 0},
		{"stopped_message_generation", u.StoppedMessageGeneration.DraftID != 0},
	}
	for _, kind := range kinds {
		if kind.filled {
			return kind.name
		}
	}
	return ""
}

// UpdateMessageT is the old name of Message, kept for compatibility
type UpdateMessageT = Message

// UpdateCallBackQueryT is the old name of CallbackQuery, kept for compatibility
type UpdateCallBackQueryT = CallbackQuery

// UpdateJSON is the response of getUpdates
type UpdateJSON struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// File is the response of getFile
type File struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileID   string `json:"file_id"`
		FilePath string `json:"file_path"`
	} `json:"result"`
}

// Button of an inline keyboard. Set one of the fields after Text
type Button struct {
	Text         string `json:"text"`
	Style        string `json:"style,omitempty"` // "danger" (red), "success" (green) or "primary" (blue)
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"` // link opened by the button instead of sending CallbackData
	// SwitchInlineQuery opens a chat chosen by the user with "@your_bot <SwitchInlineQuery>" in the input field
	SwitchInlineQuery string `json:"switch_inline_query,omitempty"`
	// SwitchInlineQueryCurrentChat inserts "@your_bot <SwitchInlineQueryCurrentChat>" in the current chat
	SwitchInlineQueryCurrentChat string `json:"switch_inline_query_current_chat,omitempty"`
	// CallbackGame launches the game sent by SendGame, the button must be the first one: &CallbackGame{}
	CallbackGame *CallbackGame `json:"callback_game,omitempty"`
	Pay          bool          `json:"pay,omitempty"` // Pay button of an invoice, it must be the first one
}

// CallbackGame is a placeholder for Button.CallbackGame
type CallbackGame struct{}

// Row is a row of buttons of an inline keyboard
type Row []Button

// InlineKeyboard is a keyboard attached to a message, see SendKeyboard
type InlineKeyboard struct {
	Keyboard []Row `json:"inline_keyboard"`
}

// LaserTeleConfigT is the configuration of a bot for NewBot and DoLaserTeleInit. All fields are optional
type LaserTeleConfigT struct {
	APIKEY  string        // bot token, if empty it is taken from TG_API_KEY env or .APIKEY file
	Timeout time.Duration // interval between requests of updates, if 0 it is taken from TIMEOUT env (default 10s)
	// CallbackOnUpdate (optional) is called for every new update,
	// in addition to the callback passed to LaserTeleRun or Run
	CallbackOnUpdate Callback
	// AllowedUpdates are kinds of updates the bot receives, default AllUpdateTypes.
	// Note that Telegram sends some of them only if the bot is an administrator in the chat
	AllowedUpdates []string
	LogMode        LogMode // what is written to log files, default LogWithoutContent
	LogDir         string  // directory of log files, default current directory
	// LogMaxSize is max size of a log file in bytes (default 10 MB).
	// The larger file is renamed to <name>.log.1, the previous <name>.log.1 is deleted
	LogMaxSize  int64
	DownloadDir string // directory for files downloaded by LoadFile, default "downloadedFiles"
}

// APIError is returned when Telegram refused the request,
// e.g. ErrorCode 403 if the bot was blocked by the user
type APIError struct {
	Method          string
	ErrorCode       int
	Description     string
	RetryAfter      int // for ErrorCode 429 (too many requests): seconds to wait before the next request
	MigrateToChatID int // the group was upgraded to a supergroup with this ID
}

// Error returns the description of the error with the method and the code
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.ErrorCode, e.Description)
}

// common part of all Bot API responses
type apiResponse struct {
	Ok          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	Parameters  struct {
		MigrateToChatID int `json:"migrate_to_chat_id"`
		RetryAfter      int `json:"retry_after"`
	} `json:"parameters"`
}

// AddButton creates a button, which sends callback (CallbackQuery.Data) to the bot when pressed
func AddButton(text, callback string) Button {
	var newButton Button
	newButton.Text = text
	newButton.CallbackData = callback

	return newButton

}
