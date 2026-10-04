package laser_tele_api

// Telegram sends updates about reactions only if the bot is an administrator in the chat

// ReactionType is a reaction: an emoji, a custom emoji or a paid reaction
type ReactionType struct {
	Type          string `json:"type"` // "emoji", "custom_emoji" or "paid"
	Emoji         string `json:"emoji,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// ReactionCount is the number of reactions of one type
type ReactionCount struct {
	Type       ReactionType `json:"type"`
	TotalCount int          `json:"total_count"`
}

// MessageReactionUpdated is sent when a user changed their reaction to a message
type MessageReactionUpdated struct {
	Chat        Chat           `json:"chat"`
	MessageID   int            `json:"message_id"`
	User        User           `json:"user"`
	ActorChat   Chat           `json:"actor_chat"` // for anonymous reactions on behalf of a chat
	Date        int            `json:"date"`
	OldReaction []ReactionType `json:"old_reaction"`
	NewReaction []ReactionType `json:"new_reaction"`
}

// MessageReactionCountUpdated is sent when anonymous reactions to a message were changed
type MessageReactionCountUpdated struct {
	Chat      Chat            `json:"chat"`
	MessageID int             `json:"message_id"`
	Date      int             `json:"date"`
	Reactions []ReactionCount `json:"reactions"`
}

// SetMessageReaction sets the reaction of the bot to a message, emoji "" removes it.
// Only emoji allowed for reactions can be used: "👍", "👎", "❤", "🔥", "🎉", "👏"...
func (b *Bot) SetMessageReaction(chatID, messageID int, emoji string) error {
	reaction := []ReactionType{}
	if emoji != "" {
		reaction = append(reaction, ReactionType{Type: "emoji", Emoji: emoji})
	}
	params := map[string]interface{}{"chat_id": chatID, "message_id": messageID, "reaction": reaction}
	return b.callInto("reactions", "setMessageReaction", params, nil)
}

// SetMessageReaction sets the reaction of the bot to a message, emoji "" removes it.
// Only emoji allowed for reactions can be used: "👍", "👎", "❤", "🔥", "🎉", "👏"...
func SetMessageReaction(chatID, messageID int, emoji string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SetMessageReaction(chatID, messageID, emoji)
}
