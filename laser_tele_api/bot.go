package laser_tele_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const tgApiLink = "https://api.telegram.org/bot"
const tgApiFileLink = "https://api.telegram.org/file/bot"
const defaultDownloadDir = "downloadedFiles"

// Clients with timeouts, so a hung connection can't stall the bot forever
var httpClient = &http.Client{Timeout: 60 * time.Second}
var fileClient = &http.Client{Timeout: 10 * time.Minute}

type Callback func(Update)

// Bot is a Telegram bot, it must be created with NewBot.
// Several bots with different tokens can work in one program.
// Methods sending messages can be called from several goroutines
type Bot struct {
	apiKey      string
	timeout     time.Duration
	apiLink     string
	fileLink    string
	downloadDir string
	onUpdate    Callback
	updates     chan Update
	logger      *logger

	allowedUpdates string // JSON list of update types for getUpdates

	httpGet    func(string) (*http.Response, error)
	httpPost   func(url, contentType string, body io.Reader) (*http.Response, error)
	fileClient *http.Client

	// used only by the goroutine requesting updates
	lastUpdateID         int
	isUpdatesInitialized bool
}

// NewBot creates a bot. If config.APIKEY is empty, it is taken from TG_API_KEY env or .APIKEY file,
// if config.Timeout is 0, it is taken from TIMEOUT env (seconds, default 10)
func NewBot(config LaserTeleConfigT) (*Bot, error) {
	apiKey := config.APIKEY
	if apiKey == "" {
		apiKey = getEnvD("TG_API_KEY", "")
	}
	if apiKey == "" {
		apiKey = loadApiKeyFromFile(".APIKEY")
	}
	if apiKey == "" {
		return nil, errors.New("APIKEY is not set: set it in config, in TG_API_KEY env or in .APIKEY file")
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeoutInt, err := strconv.Atoi(getEnvD("TIMEOUT", "10"))
		if err != nil || timeoutInt <= 0 {
			fmt.Println("Wrong TIMEOUT value, using default 10 seconds")
			timeoutInt = 10
		}
		timeout = time.Duration(timeoutInt) * time.Second
	}

	logger, err := newLogger(config, apiKey)
	if err != nil {
		return nil, err
	}

	downloadDir := config.DownloadDir
	if downloadDir == "" {
		downloadDir = defaultDownloadDir
	}

	allowedUpdates := config.AllowedUpdates
	if len(allowedUpdates) == 0 {
		allowedUpdates = AllUpdateTypes
	}
	allowedUpdatesJSON, err := json.Marshal(allowedUpdates)
	if err != nil {
		return nil, err
	}

	return &Bot{
		apiKey:         apiKey,
		timeout:        timeout,
		apiLink:        tgApiLink + apiKey,
		fileLink:       tgApiFileLink + apiKey,
		downloadDir:    downloadDir,
		onUpdate:       config.CallbackOnUpdate,
		logger:         logger,
		allowedUpdates: string(allowedUpdatesJSON),
		httpGet:        httpClient.Get,
		httpPost:       httpClient.Post,
		fileClient:     fileClient,
	}, nil
}

// MakeChan creates the channel, to which updates are sent, and returns it. It must be called before Run.
// Updates must be read from the channel, otherwise processing of updates is blocked
func (b *Bot) MakeChan() chan Update {
	if b.updates == nil {
		b.updates = make(chan Update)
	}
	return b.updates
}

// Run requests updates every timeout interval and passes them like UpdateRequest. It never returns
func (b *Bot) Run(callback Callback) {
	b.run(b.handler(callback))
}

func (b *Bot) run(handle Callback) {
	fmt.Println("Started")
	fmt.Println("Timeout: ", b.timeout)
	getUpdatesCount := 0
	for {
		b.requestUpdates(handle)
		getUpdatesCount++
		fmt.Println("Update", getUpdatesCount)
		time.Sleep(b.timeout)
	}
}

// UpdateRequest requests new updates once and passes each of them to CallbackOnUpdate from config,
// to the channel (if MakeChan was called) and to callback
func (b *Bot) UpdateRequest(callback Callback) {
	b.requestUpdates(b.handler(callback))
}

func (b *Bot) handler(callback Callback) Callback {
	return func(update Update) {
		if b.onUpdate != nil {
			b.onUpdate(update)
		}
		if b.updates != nil {
			b.updates <- update
		}
		if callback != nil {
			callback(update)
		}
	}
}

func (b *Bot) requestUpdates(handle Callback) {
	params := url.Values{}
	params.Set("allowed_updates", b.allowedUpdates)
	if !b.isUpdatesInitialized {
		// on the first request only the last update is requested, older ones are dropped by Telegram
		params.Set("offset", "-1")
	} else if b.lastUpdateID != 0 {
		// confirm processed updates, so Telegram doesn't send them again
		params.Set("offset", strconv.Itoa(b.lastUpdateID+1))
	}

	body, err := b.apiGet("updateRequest", "getUpdates", params)
	if err != nil {
		fmt.Println("Can't get updates:", err)
		return
	}

	var resultUpdate UpdateJSON
	if err := json.Unmarshal([]byte(body), &resultUpdate); err != nil {
		b.log("updateRequest", "RESP: BAD_RESPONSE "+err.Error())
		fmt.Println("Can't get updates:", err)
		return
	}

	length := len(resultUpdate.Result)
	if !b.isUpdatesInitialized {
		// updates received before start of the bot are skipped
		b.isUpdatesInitialized = true
		if length != 0 {
			b.lastUpdateID = resultUpdate.Result[length-1].UpdateID
		}
		return
	}

	if length != 0 && b.logger.mode != LogFull {
		ids := make([]string, 0, length)
		for _, update := range resultUpdate.Result {
			ids = append(ids, fmt.Sprintf("%d %s", update.UpdateID, update.Type()))
		}
		b.log("updateRequest", "UPDATES: "+strings.Join(ids, ", "))
	}

	for _, update := range resultUpdate.Result {
		b.lastUpdateID = update.UpdateID
		handle(update)
	}
}

func (b *Bot) log(logName, line string) {
	b.logger.write(logName, line)
}

// hideKeyInError hides bot token in URL of the error returned by HTTP client
func (b *Bot) hideKeyInError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		urlErr.URL = hideKey(urlErr.URL, b.apiKey)
	}
	return err
}

// apiGet calls Bot API method with params and returns response body.
// Request and response are written to log logName
func (b *Bot) apiGet(logName, method string, params url.Values) (string, error) {
	reqLink := b.apiLink + "/" + method
	if len(params) > 0 {
		reqLink += "?" + params.Encode()
	}
	if b.logger.mode == LogFull {
		b.log(logName, "REQV: GET "+reqLink)
	} else {
		b.log(logName, "REQV: GET "+method+" "+paramsWithoutContent(params))
	}

	resp, err := b.httpGet(reqLink)
	if err != nil {
		err = b.hideKeyInError(err)
		b.log(logName, "RESP: CONNECTION_ERROR "+err.Error())
		return "", err
	}
	return b.readAPIResponse(logName, method, resp)
}

// readAPIResponse reads response of Bot API method and returns *APIError if Telegram refused the request
func (b *Bot) readAPIResponse(logName, method string, resp *http.Response) (string, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.log(logName, "RESP: READ_ERROR "+err.Error())
		return "", err
	}

	var result apiResponse
	parseErr := json.Unmarshal(body, &result)
	switch {
	case b.logger.mode == LogFull:
		b.log(logName, fmt.Sprintf("RESP: %d %s", resp.StatusCode, body))
	case parseErr != nil:
		b.log(logName, fmt.Sprintf("RESP: %d WRONG_RESPONSE", resp.StatusCode))
	case !result.Ok:
		b.log(logName, fmt.Sprintf("RESP: %d %s", resp.StatusCode, result.Description))
	default:
		b.log(logName, fmt.Sprintf("RESP: %d OK", resp.StatusCode))
	}

	if parseErr != nil {
		return "", &APIError{Method: method, ErrorCode: resp.StatusCode, Description: "wrong response: " + parseErr.Error()}
	}
	if !result.Ok {
		return "", &APIError{
			Method:          method,
			ErrorCode:       result.ErrorCode,
			Description:     result.Description,
			RetryAfter:      result.Parameters.RetryAfter,
			MigrateToChatID: result.Parameters.MigrateToChatID,
		}
	}
	return string(body), nil
}

// Call calls any Bot API method (https://core.telegram.org/bots/api#available-methods) with params
// (a struct or a map, sent as JSON) and returns the "result" field of the response.
// Use it for methods that have no own function in the library
func (b *Bot) Call(method string, params interface{}) (json.RawMessage, error) {
	return b.callJSON("call", method, params)
}

// callJSON sends params as JSON to Bot API method and returns the "result" field of the response.
// Request and response are written to log logName
func (b *Bot) callJSON(logName, method string, params interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	reqLink := b.apiLink + "/" + method
	if b.logger.mode == LogFull {
		b.log(logName, fmt.Sprintf("REQV: POST %s %s", reqLink, body))
	} else {
		b.log(logName, "REQV: POST "+method+" "+jsonParamsWithoutContent(body))
	}

	resp, err := b.httpPost(reqLink, "application/json", bytes.NewReader(body))
	if err != nil {
		err = b.hideKeyInError(err)
		b.log(logName, "RESP: CONNECTION_ERROR "+err.Error())
		return nil, err
	}
	respBody, err := b.readAPIResponse(logName, method, resp)
	if err != nil {
		return nil, err
	}
	var result struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

// callInto calls Bot API method with params and decodes the result of the response to result (if not nil)
func (b *Bot) callInto(logName, method string, params interface{}, result interface{}) error {
	raw, err := b.callJSON(logName, method, params)
	if err != nil || result == nil {
		return err
	}
	if err := json.Unmarshal(raw, result); err != nil {
		return &APIError{Method: method, Description: "wrong result: " + err.Error()}
	}
	return nil
}

// Sends message to chat
func (b *Bot) SendMessage(chatID int, text string) error {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("text", text)
	_, err := b.apiGet("sendMessage", "sendMessage", params)
	return err
}

//TODO: function to edit message text
// func editMessageText(messageID,text string){

// }

// Edit inline keyboard by message id
func (b *Bot) EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) error {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("message_id", strconv.Itoa(messageID))
	params.Set("reply_markup", string(keyboardBytes))
	_, err = b.apiGet("editMessage", "editMessageReplyMarkup", params)
	return err
}

// Sending prepared inline keyboard to chat. With text(optional)
func (b *Bot) SendKeyboard(chatID int, text string, keyboard InlineKeyboard) error {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("text", text)
	params.Set("reply_markup", string(keyboardBytes))
	_, err = b.apiGet("sendKeyboard", "sendMessage", params)
	return err
}

// AnswerCallbackQuery must be called when the user pressed an inline button
// (update.CallbackQuery.ID != ""), otherwise Telegram shows progress on the button.
// text (optional) is shown to the user as a notification
func (b *Bot) AnswerCallbackQuery(callbackQueryID, text string) error {
	params := url.Values{}
	params.Set("callback_query_id", callbackQueryID)
	if text != "" {
		params.Set("text", text)
	}
	_, err := b.apiGet("answerCallback", "answerCallbackQuery", params)
	return err
}

// sendFile uploads local file fileName to chat with Bot API method (sendPhoto, sendVideo...).
// field is name of the form field for the file (photo, video...)
func (b *Bot) sendFile(logName, method, field string, chatID int, caption, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		b.log(logName, "REQV: CANT_OPEN_FILE "+err.Error())
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("chat_id", strconv.Itoa(chatID))
	if caption != "" {
		writer.WriteField("caption", caption)
	}
	part, err := writer.CreateFormFile(field, filepath.Base(fileName))
	if err == nil {
		_, err = io.Copy(part, file)
	}
	if err == nil {
		err = writer.Close()
	}
	if err != nil {
		b.log(logName, "REQV: CANT_READ_FILE "+err.Error())
		return err
	}

	reqLink := b.apiLink + "/" + method
	if b.logger.mode == LogFull {
		b.log(logName, fmt.Sprintf("REQV: POST %s chat_id=%d %s=%s caption=%q", reqLink, chatID, field, fileName, caption))
	} else {
		b.log(logName, fmt.Sprintf("REQV: POST %s chat_id=%d %s=%s", method, chatID, field, fileName))
	}

	resp, err := b.fileClient.Post(reqLink, writer.FormDataContentType(), body)
	if err != nil {
		err = b.hideKeyInError(err)
		b.log(logName, "RESP: CONNECTION_ERROR "+err.Error())
		return err
	}
	_, err = b.readAPIResponse(logName, method, resp)
	return err
}

// Sending photo to chat
func (b *Bot) SendPhoto(chatID int, text, photo string) error {
	return b.sendFile("sendPhoto", "sendPhoto", "photo", chatID, text, photo)
}

// Sending video to chat
func (b *Bot) SendVideo(chatID int, text, video string) error {
	return b.sendFile("sendVideo", "sendVideo", "video", chatID, text, video)
}

// Sending document to chat
func (b *Bot) SendDocument(chatID int, text, document string) error {
	return b.sendFile("sendDocument", "sendDocument", "document", chatID, text, document)
}

// Loading a file from user message to DownloadDir. Returns path to downloaded file
func (b *Bot) LoadFile(fileID string) (string, error) {
	params := url.Values{}
	params.Set("file_id", fileID)
	body, err := b.apiGet("loadFile", "getFile", params)
	if err != nil {
		return "", err
	}

	var resultFile File
	if err := json.Unmarshal([]byte(body), &resultFile); err != nil {
		return "", err
	}
	filePath := resultFile.Result.FilePath
	if filePath == "" {
		return "", &APIError{Method: "getFile", Description: "no file_path in response"}
	}
	if _, _, _, err := b.downloadFile(b.fileLink+"/"+filePath, filePath); err != nil {
		return "", err
	}
	return filepath.Join(b.downloadDir, filePath), nil
}

// downloadFile downloads file by link reqString and saves it to DownloadDir/filePath
func (b *Bot) downloadFile(reqString, filePath string) (*http.Response, []byte, string, error) {
	resp, err := b.fileClient.Get(reqString)
	if err != nil {
		err = b.hideKeyInError(err)
		b.log("fileLoad", "RESP: CONNECTION_ERROR "+err.Error())
		return nil, nil, "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	b.log("fileLoad", fmt.Sprintf("Response status: %s, Content-Type: %12s, Loading URL: %s", resp.Status, contentType, reqString))
	if resp.StatusCode != http.StatusOK {
		return nil, nil, "", &APIError{Method: "downloadFile", ErrorCode: resp.StatusCode, Description: http.StatusText(resp.StatusCode)}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, "", fmt.Errorf("download file %s: %w", filePath, err)
	}

	if err := saveFile(filepath.Join(b.downloadDir, filePath), data); err != nil {
		b.log("fileLoad", "CANT_SAVE_FILE "+err.Error())
		return nil, nil, "", fmt.Errorf("save file %s: %w", filePath, err)
	}
	return resp, data, contentType, nil
}
