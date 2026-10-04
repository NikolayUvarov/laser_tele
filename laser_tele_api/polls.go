package laser_tele_api

type Poll struct {
	ID                    string          `json:"id"`
	Question              string          `json:"question"`
	QuestionEntities      []MessageEntity `json:"question_entities"`
	Options               []PollOption    `json:"options"`
	TotalVoterCount       int             `json:"total_voter_count"`
	IsClosed              bool            `json:"is_closed"`
	IsAnonymous           bool            `json:"is_anonymous"`
	Type                  string          `json:"type"` // "regular" or "quiz"
	AllowsMultipleAnswers bool            `json:"allows_multiple_answers"`
	AllowsRevoting        bool            `json:"allows_revoting"`
	MembersOnly           bool            `json:"members_only"`
	CorrectOptionIDs      []int           `json:"correct_option_ids"` // for quizzes sent by the bot or closed
	Explanation           string          `json:"explanation"`
	ExplanationEntities   []MessageEntity `json:"explanation_entities"`
	OpenPeriod            int             `json:"open_period"`
	CloseDate             int             `json:"close_date"`
	Description           string          `json:"description"`
}

type PollOption struct {
	PersistentID string          `json:"persistent_id"`
	Text         string          `json:"text"`
	TextEntities []MessageEntity `json:"text_entities"`
	VoterCount   int             `json:"voter_count"`
	AddedByUser  User            `json:"added_by_user"`
	AddedByChat  Chat            `json:"added_by_chat"`
	AdditionDate int             `json:"addition_date"`
}

// PollAnswer is sent when a user voted in a non-anonymous poll sent by the bot
type PollAnswer struct {
	PollID              string   `json:"poll_id"`
	VoterChat           Chat     `json:"voter_chat"` // for votes of anonymous chat administrators
	User                User     `json:"user"`
	OptionIDs           []int    `json:"option_ids"` // 0-based indexes of chosen options, empty if the vote was retracted
	OptionPersistentIDs []string `json:"option_persistent_ids"`
}

// PollConfig contains optional parameters of a poll for SendPoll
type PollConfig struct {
	Quiz             bool   // quiz with correct answers instead of a regular poll
	CorrectOptionIDs []int  // for a quiz: 0-based indexes of the correct options in increasing order
	Explanation      string // for a quiz: shown when the user chose a wrong answer
	// NotAnonymous makes votes visible; the bot receives them as poll_answer updates only for such polls
	NotAnonymous          bool
	AllowsMultipleAnswers bool
	OpenPeriod            int // seconds the poll is active after creation, 5-600
	CloseDate             int // unix time when the poll is closed
	IsClosed              bool
	ReplyMarkup           *InlineKeyboard
}

// SendPoll sends a poll with 1-12 options and returns the sent message (its Poll.ID identifies the poll)
func (b *Bot) SendPoll(chatID int, question string, options []string, config PollConfig) (Message, error) {
	pollOptions := make([]map[string]string, len(options))
	for i, option := range options {
		pollOptions[i] = map[string]string{"text": option}
	}
	params := map[string]interface{}{"chat_id": chatID, "question": question, "options": pollOptions}
	if config.Quiz {
		params["type"] = "quiz"
		params["correct_option_ids"] = config.CorrectOptionIDs
		if config.Explanation != "" {
			params["explanation"] = config.Explanation
		}
	}
	if config.NotAnonymous {
		params["is_anonymous"] = false
	}
	if config.AllowsMultipleAnswers {
		params["allows_multiple_answers"] = true
	}
	if config.OpenPeriod > 0 {
		params["open_period"] = config.OpenPeriod
	}
	if config.CloseDate > 0 {
		params["close_date"] = config.CloseDate
	}
	if config.IsClosed {
		params["is_closed"] = true
	}
	if config.ReplyMarkup != nil {
		params["reply_markup"] = config.ReplyMarkup
	}
	var message Message
	err := b.callInto("polls", "sendPoll", params, &message)
	return message, err
}

// StopPoll closes the poll sent by the bot and returns its final results
func (b *Bot) StopPoll(chatID, messageID int) (Poll, error) {
	var poll Poll
	err := b.callInto("polls", "stopPoll", map[string]interface{}{"chat_id": chatID, "message_id": messageID}, &poll)
	return poll, err
}

// SendPoll sends a poll with 1-12 options and returns the sent message (its Poll.ID identifies the poll)
func SendPoll(chatID int, question string, options []string, config PollConfig) (Message, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return Message{}, err
	}
	return bot.SendPoll(chatID, question, options, config)
}

// StopPoll closes the poll sent by the bot and returns its final results
func StopPoll(chatID, messageID int) (Poll, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return Poll{}, err
	}
	return bot.StopPoll(chatID, messageID)
}
