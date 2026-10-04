package laser_tele_api

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

// Package-level functions work with the default bot. It is created by DoLaserTeleInit
// or on the first call (with APIKEY, TG_API_KEY env or .APIKEY file).
// To work with several bots in one program use NewBot

// APIKEY is the token of the default bot
var APIKEY string

// OnUpdateCallbackFunc is called for every new update of the default bot (CallbackOnUpdate of the config)
var OnUpdateCallbackFunc func(Update)

// TgChan is the channel for updates of the default bot, it is created by MakeChan
var TgChan chan Update

var isChan bool

var defaultBot *Bot
var defaultBotMutex sync.Mutex

// MakeChan creates the channel TgChan, to which updates of the default bot are sent.
// It must be called before LaserTeleRun. Updates must be read from TgChan, otherwise processing of updates is blocked
func MakeChan() {
	if TgChan == nil {
		TgChan = make(chan Update)
	}
	isChan = true
}

// DoLaserTeleInit configures the default bot with config. It exits the program if the token is not found
func DoLaserTeleInit(config LaserTeleConfigT) {
	if config.CallbackOnUpdate != nil {
		OnUpdateCallbackFunc = config.CallbackOnUpdate
	}
	// the default bot calls CallbackOnUpdate via OnUpdateCallbackFunc
	config.CallbackOnUpdate = nil

	if config.APIKEY == "" && os.Getenv("TG_API_KEY") == "" {
		fmt.Println("ENV APIKEY is empty, try to read from .APIKEY file")
	}
	bot, err := NewBot(config)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Can't read APIKEY, exiting")
		os.Exit(1)
	}

	defaultBotMutex.Lock()
	defaultBot = bot
	APIKEY = bot.apiKey
	defaultBotMutex.Unlock()

	fmt.Println("APIKEY is set")
	fmt.Println("TIMEOUT: ", bot.timeout)
}

// getDefaultBot returns the default bot, it is created if it doesn't exist yet
func getDefaultBot() (*Bot, error) {
	defaultBotMutex.Lock()
	defer defaultBotMutex.Unlock()
	if defaultBot == nil {
		bot, err := NewBot(LaserTeleConfigT{APIKEY: APIKEY})
		if err != nil {
			return nil, err
		}
		defaultBot = bot
		APIKEY = bot.apiKey
	}
	return defaultBot, nil
}

// defaultHandler passes update of the default bot to OnUpdateCallbackFunc, TgChan and callback
func defaultHandler(callback Callback) Callback {
	return func(update Update) {
		if OnUpdateCallbackFunc != nil {
			OnUpdateCallbackFunc(update)
		}
		if isChan {
			TgChan <- update
		}
		if callback != nil {
			callback(update)
		}
	}
}

// LaserTeleRun requests updates of the default bot every Timeout and passes each of them
// to OnUpdateCallbackFunc, TgChan (if MakeChan was called) and callback.
// It never returns and exits the program if the token is not found
func LaserTeleRun(callback Callback) {
	bot, err := getDefaultBot()
	if err != nil {
		fmt.Println(err)
		fmt.Println("Can't read APIKEY, exiting")
		os.Exit(1)
	}
	bot.run(defaultHandler(callback))
}

// UpdateRequest requests new updates of the default bot once and passes each of them
// to OnUpdateCallbackFunc, TgChan (if MakeChan was called) and callback
func UpdateRequest(callback Callback) {
	bot, err := getDefaultBot()
	if err != nil {
		fmt.Println("Can't get updates:", err)
		return
	}
	bot.requestUpdates(defaultHandler(callback))
}

// SendMessage sends a text message to the chat
func SendMessage(chatID int, text string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendMessage(chatID, text)
}

// EditMessageReplyMarkup changes the inline keyboard of a message sent by the bot, an empty keyboard removes it
func EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.EditMessageReplyMarkup(chatID, messageID, keyboard)
}

// SendKeyboard sends a message with text and an inline keyboard
func SendKeyboard(chatID int, text string, keyboard InlineKeyboard) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendKeyboard(chatID, text, keyboard)
}

// AnswerCallbackQuery must be called when the user pressed an inline button
// (update.CallbackQuery.ID != ""), otherwise Telegram shows progress on the button.
// text (optional) is shown to the user as a notification
func AnswerCallbackQuery(callbackQueryID, text string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.AnswerCallbackQuery(callbackQueryID, text)
}

// SendPhoto uploads a local photo file to the chat, text is the caption
func SendPhoto(chatID int, text, photo string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendPhoto(chatID, text, photo)
}

// SendVideo uploads a local video file to the chat, text is the caption
func SendVideo(chatID int, text, video string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendVideo(chatID, text, video)
}

// SendDocument uploads a local file to the chat as a document, text is the caption
func SendDocument(chatID int, text, document string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendDocument(chatID, text, document)
}

// LoadFile downloads a file sent by a user (its file_id) to DownloadDir and returns its path.
// chatID is not used
func LoadFile(chatID int, fileID string) (string, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return "", err
	}
	return bot.LoadFile(fileID)
}

// FileDownload downloads the file by link reqString and saves it to filePath in DownloadDir of the default bot.
// On error resp is nil and data contains the error message
func FileDownload(reqString, filePath string) (resp *http.Response, data []byte, contentType string) {
	bot, err := getDefaultBot()
	if err == nil {
		resp, data, contentType, err = bot.downloadFile(reqString, filePath)
	}
	if err != nil {
		return nil, []byte(err.Error()), ""
	}
	return resp, data, contentType
}
