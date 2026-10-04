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

var APIKEY string
var timeout time.Duration

// Clients with timeouts, so a hung connection can't stall the bot forever
var httpClient = &http.Client{Timeout: 60 * time.Second}
var fileClient = &http.Client{Timeout: 10 * time.Minute}
var httpGet = httpClient.Get
var isConfigDone bool = false

var tgApiLinkKEY string
var newUpdate Update
var isUpdatesInitialized bool
var OnUpdateCallbackFunc func(Update)
var TgChan chan Update
var isChan bool

type Callback func(Update)

// Function,that creates channel for sending updates.
// Updates must be read from TgChan, otherwise processing of updates is blocked
func MakeChan() {
	if TgChan == nil {
		TgChan = make(chan Update)
	}
	isChan = true
}

func doLaserTeleInit() {

	//if APIKEY set via DoLaserTeleInit it will be not empty Else try to get from ENV or from file
	if APIKEY == "" {
		APIKEY = getEnvD("TG_API_KEY", "")
	}
	if APIKEY == "" {
		fmt.Println("ENV APIKEY is empty, try to read from .APIKEY file")
		APIKEY = loadApiKeyFromFile(".APIKEY")
		if APIKEY == "" {
			fmt.Println("Please set APIKEY in ENV or in .APIKEY file!")
			fmt.Println("Can't read APIKEY from file, exiting")
			os.Exit(1)
		}
	}
	//if timeout is set via DoLaserTeleInit it will be not 0 Else try to get from ENV (defalut 10)
	if timeout <= 0 {
		timeoutInt, err := strconv.Atoi(getEnvD("TIMEOUT", "10"))
		if err != nil || timeoutInt <= 0 {
			fmt.Println("Wrong TIMEOUT value, using default 10 seconds")
			timeoutInt = 10
		}
		timeout = time.Duration(timeoutInt) * time.Second
	}
	tgApiLinkKEY = tgApiLink + APIKEY
	isConfigDone = true
	fmt.Println("APIKEY is set")
	fmt.Println("TIMEOUT: ", timeout)
}

// Initializing of bot, setting api key and timeout via config
func DoLaserTeleInit(config LaserTeleConfigT) {
	timeout = config.Timeout
	APIKEY = config.APIKEY
	if config.CallbackOnUpdate != nil {
		OnUpdateCallbackFunc = config.CallbackOnUpdate
	}
	doLaserTeleInit()
}

// Running bot, returning updates with callback
func LaserTeleRun(callback Callback) {
	if !isConfigDone {
		doLaserTeleInit()
	}
	fmt.Println("Started")
	fmt.Println("Timeout: ", timeout)
	getUpdatesCount := 0
	for {
		UpdateRequest(callback)
		getUpdatesCount++
		fmt.Println("Update", getUpdatesCount)
		time.Sleep(timeout)
	}
}

// hideKey replaces bot token in s, so it doesn't get to logs
func hideKey(s string) string {
	if APIKEY == "" {
		return s
	}
	return strings.ReplaceAll(s, APIKEY, "<APIKEY>")
}

// hideKeyInError hides bot token in URL of the error returned by HTTP client
func hideKeyInError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		urlErr.URL = hideKey(urlErr.URL)
	}
	return err
}

// apiGet calls Bot API method with params and returns response body.
// Request and response are written to log logName
func apiGet(logName, method string, params url.Values) (string, error) {
	reqLink := tgApiLinkKEY + "/" + method
	if len(params) > 0 {
		reqLink += "?" + params.Encode()
	}
	line2logfile(logName, "REQV: GET "+reqLink)

	resp, err := httpGet(reqLink)
	if err != nil {
		err = hideKeyInError(err)
		line2logfile(logName, "RESP: CONNECTION_ERROR "+err.Error())
		return "", err
	}
	return readAPIResponse(logName, method, resp)
}

// readAPIResponse reads response of Bot API method and returns *APIError if Telegram refused the request
func readAPIResponse(logName, method string, resp *http.Response) (string, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		line2logfile(logName, "RESP: READ_ERROR "+err.Error())
		return "", err
	}
	line2logfile(logName, fmt.Sprintf("RESP: %d %s", resp.StatusCode, body))

	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", &APIError{Method: method, ErrorCode: resp.StatusCode, Description: "wrong response: " + err.Error()}
	}
	if !result.Ok {
		return "", &APIError{Method: method, ErrorCode: result.ErrorCode, Description: result.Description}
	}
	return string(body), nil
}

func UpdateRequest(callback Callback) {
	params := url.Values{}
	if !isUpdatesInitialized {
		// on the first request only the last update is requested, older ones are dropped by Telegram
		params.Set("offset", "-1")
	} else if newUpdate.UpdateID != 0 {
		// confirm processed updates, so Telegram doesn't send them again
		params.Set("offset", strconv.Itoa(newUpdate.UpdateID+1))
	}

	body, err := apiGet("updateRequest", "getUpdates", params)
	if err != nil {
		fmt.Println("Can't get updates")
		return
	}

	var resultUpdate UpdateJSON
	if err := json.Unmarshal([]byte(body), &resultUpdate); err != nil {
		line2logfile("updateRequest", "RESP: BAD_RESPONSE "+err.Error())
		fmt.Println("Can't get updates")
		return
	}

	length := len(resultUpdate.Result)
	if !isUpdatesInitialized {
		// updates received before start of the bot are skipped
		isUpdatesInitialized = true
		if length != 0 {
			newUpdate.UpdateID = resultUpdate.Result[length-1].UpdateID
		}
		return
	}

	for num := range resultUpdate.Result {
		newUpdate.UpdateID = resultUpdate.Result[num].UpdateID
		newUpdate.UpdateMessage = UpdateMessageT(resultUpdate.Result[num].Message)
		newUpdate.CallbackQuery = UpdateCallBackQueryT(resultUpdate.Result[num].CallbackQuery)
		if OnUpdateCallbackFunc != nil {
			OnUpdateCallbackFunc(newUpdate)
		}
		if isChan {
			TgChan <- newUpdate
		}
		if callback != nil {
			callback(newUpdate)
		}
	}
}

// Sends message to chat
func SendMessage(chatID int, text string) error {
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("text", text)
	_, err := apiGet("sendMessage", "sendMessage", params)
	return err
}

//TODO: function to edit message text
// func editMessageText(messageID,text string){

// }

// Edit inline keyboard by message id
func EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) error {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("message_id", strconv.Itoa(messageID))
	params.Set("reply_markup", string(keyboardBytes))
	_, err = apiGet("editMessage", "editMessageReplyMarkup", params)
	return err
}

// Sending prepared inline keyboard to chat. With text(optional)
func SendKeyboard(chatID int, text string, keyboard InlineKeyboard) error {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("chat_id", strconv.Itoa(chatID))
	params.Set("text", text)
	params.Set("reply_markup", string(keyboardBytes))
	_, err = apiGet("sendKeyboard", "sendMessage", params)
	return err
}

// sendFile uploads local file fileName to chat with Bot API method (sendPhoto, sendVideo...).
// field is name of the form field for the file (photo, video...)
func sendFile(logName, method, field string, chatID int, caption, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		line2logfile(logName, "REQV: CANT_OPEN_FILE "+err.Error())
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
		line2logfile(logName, "REQV: CANT_READ_FILE "+err.Error())
		return err
	}

	reqLink := tgApiLinkKEY + "/" + method
	line2logfile(logName, fmt.Sprintf("REQV: POST %s chat_id=%d %s=%s caption=%q", reqLink, chatID, field, fileName, caption))

	resp, err := fileClient.Post(reqLink, writer.FormDataContentType(), body)
	if err != nil {
		err = hideKeyInError(err)
		line2logfile(logName, "RESP: CONNECTION_ERROR "+err.Error())
		return err
	}
	_, err = readAPIResponse(logName, method, resp)
	return err
}

// Sending photo to chat
func SendPhoto(chatID int, text, photo string) error {
	return sendFile("sendPhoto", "sendPhoto", "photo", chatID, text, photo)
}

// Sending video to chat
func SendVideo(chatID int, text, video string) error {
	return sendFile("sendVideo", "sendVideo", "video", chatID, text, video)
}

// Sending document to chat
func SendDocument(chatID int, text, document string) error {
	return sendFile("sendDocument", "sendDocument", "document", chatID, text, document)
}

// Loading a file from user message. Returns path to downloaded file or "" on error
func LoadFile(chatID int, fileID string) string {
	params := url.Values{}
	params.Set("file_id", fileID)
	body, err := apiGet("loadFile", "getFile", params)
	if err != nil {
		return ""
	}

	var resultFile File
	if err := json.Unmarshal([]byte(body), &resultFile); err != nil || !resultFile.Ok {
		return ""
	}
	filePath := resultFile.Result.FilePath
	fileLink := tgApiFileLink + APIKEY + "/" + filePath
	if resp, _, _ := FileDownload(fileLink, filePath); resp == nil {
		return ""
	}
	return "downloadedFiles/" + filePath
}

func FileDownload(reqString, filePath string) (resp *http.Response, data []byte, contentType string) {

	var err error
	resp, err = fileClient.Get(reqString)
	if err != nil {
		line2logfile("fileLoad", "RESP: CONNECTION_ERROR "+err.Error())
		return nil, []byte("Cannot open addr " + hideKey(reqString)), ""
	}
	defer resp.Body.Close()

	contentType = resp.Header.Get("Content-Type")
	line2logfile("fileLoad", fmt.Sprintf("Response status: %s, Content-Type: %12s, Loading URL: %s", resp.Status, contentType, reqString))
	if resp.StatusCode != http.StatusOK {
		return nil, []byte("Bad response status " + resp.Status + " from " + hideKey(reqString)), ""
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, []byte("Error reading data from remote " + hideKey(reqString)), ""
	}

	if err := saveFile("downloadedFiles/"+filePath, data); err != nil {
		line2logfile("fileLoad", "CANT_SAVE_FILE "+err.Error())
		return nil, []byte("Error saving file " + filePath + ": " + err.Error()), ""
	}
	return resp, data, contentType
}

func StringToFile(fileName string, str string) {
	if err := saveFile(fileName, []byte(str)); err != nil {
		fmt.Println("Can't save file", fileName, err)
	}
}

func saveFile(fileName string, data []byte) error {
	if err := checkPath(fileName); err != nil {
		return err
	}
	return os.WriteFile(fileName, data, 0644)
}

// checkPath check if path to file exists and creates path if not. If meant to be path-to-file is directory - returns error
func checkPath(path string) error {
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(filepath.Dir(path), 0750)
	}
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return errors.New("ISDIR")
	}
	return nil
}
