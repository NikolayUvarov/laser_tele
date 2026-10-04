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

var APIKEY string                     // token of the default bot
var OnUpdateCallbackFunc func(Update) // called for every new update of the default bot
var TgChan chan Update                // channel for updates of the default bot, created by MakeChan
var isChan bool

var defaultBot *Bot
var defaultBotMutex sync.Mutex

// Function,that creates channel for sending updates.
// Updates must be read from TgChan, otherwise processing of updates is blocked
func MakeChan() {
	if TgChan == nil {
		TgChan = make(chan Update)
	}
	isChan = true
}

// Initializing of the default bot, setting api key, timeout and logs via config.
// Exits the program if APIKEY is not set
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

// Running bot, returning updates with callback. Exits the program if APIKEY is not set
func LaserTeleRun(callback Callback) {
	bot, err := getDefaultBot()
	if err != nil {
		fmt.Println(err)
		fmt.Println("Can't read APIKEY, exiting")
		os.Exit(1)
	}
	bot.run(defaultHandler(callback))
}

func UpdateRequest(callback Callback) {
	bot, err := getDefaultBot()
	if err != nil {
		fmt.Println("Can't get updates:", err)
		return
	}
	bot.requestUpdates(defaultHandler(callback))
}

// Sends message to chat
func SendMessage(chatID int, text string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendMessage(chatID, text)
}

// Edit inline keyboard by message id
func EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.EditMessageReplyMarkup(chatID, messageID, keyboard)
}

// Sending prepared inline keyboard to chat. With text(optional)
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

// Sending photo to chat
func SendPhoto(chatID int, text, photo string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendPhoto(chatID, text, photo)
}

// Sending video to chat
func SendVideo(chatID int, text, video string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendVideo(chatID, text, video)
}

// Sending document to chat
func SendDocument(chatID int, text, document string) error {
	bot, err := getDefaultBot()
	if err != nil {
		return err
	}
	return bot.SendDocument(chatID, text, document)
}

// Loading a file from user message. Returns path to downloaded file
func LoadFile(chatID int, fileID string) (string, error) {
	bot, err := getDefaultBot()
	if err != nil {
		return "", err
	}
	return bot.LoadFile(fileID)
}

// Downloads file by link reqString and saves it to downloadedFiles/filePath.
// On error resp is nil and data contains error message
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
