package laser_tele_api

// Updates about chats: join requests, boosts, business accounts, managed bots

type ChatInviteLink struct {
	InviteLink              string `json:"invite_link"`
	Creator                 User   `json:"creator"`
	CreatesJoinRequest      bool   `json:"creates_join_request"`
	IsPrimary               bool   `json:"is_primary"`
	IsRevoked               bool   `json:"is_revoked"`
	Name                    string `json:"name"`
	ExpireDate              int    `json:"expire_date"`
	MemberLimit             int    `json:"member_limit"`
	PendingJoinRequestCount int    `json:"pending_join_request_count"`
	SubscriptionPeriod      int    `json:"subscription_period"`
	SubscriptionPrice       int    `json:"subscription_price"`
}

// ChatJoinRequest is sent when a user asked to join the chat, answer it with
// ApproveChatJoinRequest or DeclineChatJoinRequest. The bot must be an administrator with can_invite_users right
type ChatJoinRequest struct {
	Chat       Chat           `json:"chat"`
	From       User           `json:"from"`
	UserChatID int            `json:"user_chat_id"` // private chat with the user, the bot can write there for 5 minutes
	Date       int            `json:"date"`
	Bio        string         `json:"bio"`
	InviteLink ChatInviteLink `json:"invite_link"`
	QueryID    string         `json:"query_id"`
}

type ChatBoostSource struct {
	Source            string `json:"source"` // "premium", "gift_code" or "giveaway"
	User              User   `json:"user"`
	GiveawayMessageID int    `json:"giveaway_message_id"`
	PrizeStarCount    int    `json:"prize_star_count"`
	IsUnclaimed       bool   `json:"is_unclaimed"`
}

type ChatBoost struct {
	BoostID        string          `json:"boost_id"`
	AddDate        int             `json:"add_date"`
	ExpirationDate int             `json:"expiration_date"`
	Source         ChatBoostSource `json:"source"`
}

type ChatBoostUpdated struct {
	Chat  Chat      `json:"chat"`
	Boost ChatBoost `json:"boost"`
}

type ChatBoostRemoved struct {
	Chat       Chat            `json:"chat"`
	BoostID    string          `json:"boost_id"`
	RemoveDate int             `json:"remove_date"`
	Source     ChatBoostSource `json:"source"`
}

type BusinessBotRights struct {
	CanReply                   bool `json:"can_reply"`
	CanReadMessages            bool `json:"can_read_messages"`
	CanDeleteSentMessages      bool `json:"can_delete_sent_messages"`
	CanDeleteAllMessages       bool `json:"can_delete_all_messages"`
	CanEditName                bool `json:"can_edit_name"`
	CanEditBio                 bool `json:"can_edit_bio"`
	CanEditProfilePhoto        bool `json:"can_edit_profile_photo"`
	CanEditUsername            bool `json:"can_edit_username"`
	CanChangeGiftSettings      bool `json:"can_change_gift_settings"`
	CanViewGiftsAndStars       bool `json:"can_view_gifts_and_stars"`
	CanConvertGiftsToStars     bool `json:"can_convert_gifts_to_stars"`
	CanTransferAndUpgradeGifts bool `json:"can_transfer_and_upgrade_gifts"`
	CanTransferStars           bool `json:"can_transfer_stars"`
	CanManageStories           bool `json:"can_manage_stories"`
}

// BusinessConnection is sent when the bot was connected to or disconnected from a business account.
// Messages of the account come as business_message updates, answer them with
// SendMessageWithConfig and MessageConfig.BusinessConnectionID
type BusinessConnection struct {
	ID         string            `json:"id"`
	User       User              `json:"user"`
	UserChatID int               `json:"user_chat_id"`
	Date       int               `json:"date"`
	Rights     BusinessBotRights `json:"rights"`
	IsEnabled  bool              `json:"is_enabled"`
}

type BusinessMessagesDeleted struct {
	BusinessConnectionID string `json:"business_connection_id"`
	Chat                 Chat   `json:"chat"`
	MessageIDs           []int  `json:"message_ids"`
}

// ManagedBotUpdated is sent when a bot managed by this bot was created or its token or owner was changed,
// get its token with GetManagedBotToken
type ManagedBotUpdated struct {
	User User `json:"user"` // user who created the bot
	Bot  User `json:"bot"`
}

// MessageGenerationStopped is sent when the user asked the bot to stop generation of a message
type MessageGenerationStopped struct {
	Chat            Chat `json:"chat"`
	MessageThreadID int  `json:"message_thread_id"`
	DraftID         int  `json:"draft_id"`
}

// ApproveChatJoinRequest lets the user (ChatJoinRequest.From.ID) join the chat
func (b *Bot) ApproveChatJoinRequest(chatID, userID int) error {
	return b.callInto("joinRequests", "approveChatJoinRequest", map[string]interface{}{"chat_id": chatID, "user_id": userID}, nil)
}

// DeclineChatJoinRequest rejects the request of the user (ChatJoinRequest.From.ID) to join the chat
func (b *Bot) DeclineChatJoinRequest(chatID, userID int) error {
	return b.callInto("joinRequests", "declineChatJoinRequest", map[string]interface{}{"chat_id": chatID, "user_id": userID}, nil)
}

// GetManagedBotToken returns the token of the bot (ManagedBotUpdated.Bot.ID) managed by this bot
func (b *Bot) GetManagedBotToken(botUserID int) (string, error) {
	var token string
	err := b.callInto("managedBots", "getManagedBotToken", map[string]interface{}{"user_id": botUserID}, &token)
	return token, err
}

// ApproveChatJoinRequest lets the user (ChatJoinRequest.From.ID) join the chat
func ApproveChatJoinRequest(chatID, userID int) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.ApproveChatJoinRequest(chatID, userID)
}

// DeclineChatJoinRequest rejects the request of the user (ChatJoinRequest.From.ID) to join the chat
func DeclineChatJoinRequest(chatID, userID int) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.DeclineChatJoinRequest(chatID, userID)
}

// GetManagedBotToken returns the token of the bot (ManagedBotUpdated.Bot.ID) managed by this bot
func GetManagedBotToken(botUserID int) (string, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return "", err
	}
	return bot.GetManagedBotToken(botUserID)
}
