package laser_tele_api

// Games: https://core.telegram.org/bots/games. A game is created with /newgame in @BotFather.
// When the user presses the "Play" button, a callback query with GameShortName is sent,
// answer it with AnswerCallbackQueryWithConfig and the URL of the game

type Game struct {
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Photo        []PhotoSize     `json:"photo"`
	Text         string          `json:"text"`
	TextEntities []MessageEntity `json:"text_entities"`
	Animation    Animation       `json:"animation"`
}

type GameHighScore struct {
	Position int  `json:"position"`
	User     User `json:"user"`
	Score    int  `json:"score"`
}

// GameMessage is the message with a game: ChatID and MessageID,
// or InlineMessageID (CallbackQuery.InlineMessageID) for a game sent in inline mode
type GameMessage struct {
	ChatID          int
	MessageID       int
	InlineMessageID string
}

func (m GameMessage) params(userID int) map[string]interface{} {
	params := map[string]interface{}{"user_id": userID}
	if m.InlineMessageID != "" {
		params["inline_message_id"] = m.InlineMessageID
	} else {
		params["chat_id"] = m.ChatID
		params["message_id"] = m.MessageID
	}
	return params
}

// SendGame sends the game created in @BotFather and returns the sent message
func (b *Bot) SendGame(chatID int, gameShortName string) (Message, error) {
	var message Message
	err := b.callInto("games", "sendGame", map[string]interface{}{"chat_id": chatID, "game_short_name": gameShortName}, &message)
	return message, err
}

// SetGameScore sets the score of the user in the game. A score lower than the current one is set only with force
func (b *Bot) SetGameScore(userID, score int, message GameMessage, force bool) error {
	params := message.params(userID)
	params["score"] = score
	if force {
		params["force"] = true
	}
	return b.callInto("games", "setGameScore", params, nil)
}

// GetGameHighScores returns the high scores of the user and several of their neighbors in the game
func (b *Bot) GetGameHighScores(userID int, message GameMessage) ([]GameHighScore, error) {
	var scores []GameHighScore
	err := b.callInto("games", "getGameHighScores", message.params(userID), &scores)
	return scores, err
}

// SendGame sends the game created in @BotFather and returns the sent message
func SendGame(chatID int, gameShortName string) (Message, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return Message{}, err
	}
	return bot.SendGame(chatID, gameShortName)
}

// SetGameScore sets the score of the user in the game. A score lower than the current one is set only with force
func SetGameScore(userID, score int, message GameMessage, force bool) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SetGameScore(userID, score, message, force)
}

// GetGameHighScores returns the high scores of the user and several of their neighbors in the game
func GetGameHighScores(userID int, message GameMessage) ([]GameHighScore, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return nil, err
	}
	return bot.GetGameHighScores(userID, message)
}
