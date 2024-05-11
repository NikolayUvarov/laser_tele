package laser_tele_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const tgApiLink = "https://api.telegram.org/bot"

var APIKEY string
var timeout time.Duration
var httpGet = http.Get
var isConfigDone bool = false

var tgApiLinkKEY string
var newUpdate Update
var OnUpdateCallbackFunc func(Update)
var TgChan chan Update
var isChan bool

type Callback func(Update)

// Function,that creates channel for sending updates
func MakeChan() {
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
	if timeout == 0 {
		timeoutInt, _ := strconv.Atoi(getEnvD("TIMEOUT", "10"))
		timeout = time.Duration(timeoutInt) * time.Second
	}
	tgApiLinkKEY = tgApiLink + "" + APIKEY
	fmt.Println("APIKEY: ", APIKEY)
	fmt.Println("TIMEOUT: ", timeout)
	fmt.Println("tgApiLinkKEY: ", tgApiLinkKEY)
}

// Initializing of bot, setting api key and timeout via config
func DoLaserTeleInit(config LaserTeleConfigT) {
	timeout = config.Timeout
	APIKEY = config.APIKEY
	if config.CallbackOnUpdate != nil && reflect.TypeOf(config.CallbackOnUpdate).Kind() == reflect.Func {
		//onUpdateCallbackFunc = reflect.ValueOf(onUpdateCallbackFunc).(func(Update))
		//set to onUpdateCallbackFunc value of config.CallbackOnUpdate
		OnUpdateCallbackFunc = config.CallbackOnUpdate.(func(Update))
	}
	doLaserTeleInit()
}

// Running bot, returning updates with callback
func LaserTeleRun(callback Callback) {
	fmt.Println("Started")
	fmt.Println("Timeout: ", timeout)
	getUpdatesCount := 0
	for {
		UpdateRequest(func(update Update) {
			callback(update)
		})
		getUpdatesCount++
		fmt.Println("Update", getUpdatesCount)
		time.Sleep(timeout)
	}
}

func UpdateRequest(callback Callback) {
	reqLink := tgApiLinkKEY + "/getUpdates"
	reqType := "GET"
	var reqBody io.Reader = nil

	line2logfile("updateRequest", fmt.Sprintf("REQV: %s %s %v", reqType, reqLink, reqBody))
	req, err := http.NewRequest(reqType, reqLink, reqBody)
	if err != nil {
		line2logfile("updateRequest", "RESP: REQUEST_ERROR")
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	resp, err := httpGet(req.URL.String())

	if err == nil {
		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)

		line2logfile("updateRequest", fmt.Sprintf("RESP: %d %s", resp.StatusCode, buf.String()))

		var resultUpdate UpdateJSON
		json.Unmarshal([]byte(buf.String()), &resultUpdate)
		length := len(resultUpdate.Result)
		fmt.Println(length)
		if newUpdate.UpdateID == 0 {
			if length != 0 {
				newUpdate.UpdateID = resultUpdate.Result[length-1].UpdateID
			} else {
				newUpdate.UpdateID = 0
			}
		} else {
			for newUpdate.UpdateID < resultUpdate.Result[length-1].UpdateID {
				newUpdate.UpdateID++

				for num := range resultUpdate.Result {
					if resultUpdate.Result[num].UpdateID == newUpdate.UpdateID {
						newUpdate.UpdateMessage = UpdateMessageT(resultUpdate.Result[num].Message)
						newUpdate.CallbackQuery = UpdateCallBackQueryT(resultUpdate.Result[num].CallbackQuery)
						fmt.Println(resultUpdate.Result[num].UpdateID)
						fmt.Println(newUpdate.UpdateMessage)
						if isChan {
							TgChan <- newUpdate
						}

						callback(newUpdate)
					}
				}
			}
		}
	} else {
		line2logfile("updateRequest", "RESP: CONNECION_ERROR")
		fmt.Println("Can't get updates")
	}

}

// Sends message to chat
func SendMessage(chatID int, text string) {
	reqLink := tgApiLinkKEY + "/sendMessage?chat_id=" + fmt.Sprint(chatID) + "&text=" + text
	reqType := "GET"
	var reqBody io.Reader = nil
	line2logfile("sendMessage", fmt.Sprintf("REQV: %s %s %v", reqType, reqLink, reqBody))
	req, err := http.NewRequest("GET", reqLink, nil)
	if err != nil {
		fmt.Println("Error creating request")
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	update, err := httpGet(req.URL.String())
	if err != nil {
		line2logfile("sendMessage", "RESP: ERROR_SENDING_MESSAGE")
	} else {
		buf := new(strings.Builder)
		io.Copy(buf, update.Body)

		line2logfile("sendMessage", fmt.Sprintf("RESP: %d %s", update.StatusCode, buf.String()))
	}
}

//TODO: function to edit message text
// func editMessageText(messageID,text string){

// }

// Edit inline keyboard by message id
func EditMessageReplyMarkup(chatID, messageID int, keyboard InlineKeyboard) {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		fmt.Println(err)
	}
	reqLink := tgApiLinkKEY + "/editMessageReplyMarkup?chat_id=" + fmt.Sprint(chatID) + "&message_id=" + fmt.Sprint(messageID) + "&reply_markup=" + string(keyboardBytes)
	reqType := "GET"
	var reqBody io.Reader = nil
	line2logfile("editMessage", fmt.Sprintf("REQV: %s %s %v", reqType, reqLink, reqBody))
	req, err := http.NewRequest("GET", reqLink, nil)
	if err != nil {
		fmt.Println("Error creating request")
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	update, err := httpGet(req.URL.String())
	if err != nil {
		line2logfile("editMessage", "RESP: ERROR_SENDING_MESSAGE")
	} else {
		buf := new(strings.Builder)
		io.Copy(buf, update.Body)

		line2logfile("editMessage", fmt.Sprintf("RESP: %d %s", update.StatusCode, buf.String()))
	}
}

// Sending prepared inline keyboard to chat. With text(optional)
func SendKeyboard(chatID int, text string, keyboard InlineKeyboard) {
	keyboardBytes, err := json.Marshal(&keyboard)
	if err != nil {
		fmt.Println(err)
	}
	reqLink := tgApiLinkKEY + "/sendMessage?chat_id=" + fmt.Sprint(chatID) + "&text=" + text + "&reply_markup=" + string(keyboardBytes)
	reqType := "GET"
	var reqBody io.Reader = nil
	line2logfile("sendKeyboard", fmt.Sprintf("REQV: %s %s %v", reqType, reqLink, reqBody))
	req, err := http.NewRequest("GET", reqLink, nil)
	if err != nil {
		fmt.Println("Error creating request")
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()
	update, err := httpGet(req.URL.String())
	if err != nil {
		line2logfile("sendMessage", "RESP: ERROR_SENDING_MESSAGE")
	} else {
		buf := new(strings.Builder)
		io.Copy(buf, update.Body)

		line2logfile("sendMessage", fmt.Sprintf("RESP: %d %s", update.StatusCode, buf.String()))
	}
}

// Sending photo to chat
func SendPhoto(chatID int, text, photo string) {
	reqLink := tgApiLinkKEY + "/sendPhoto?chat_id=" + fmt.Sprintf("%d", chatID) + "&caption=" + text
	reqType := "POST"
	fileDir, _ := os.Getwd()
	fileName := photo
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("photo", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	r, _ := http.NewRequest(reqType, reqLink, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		line2logfile("sendPhoto", "RESP: ERROR_SENDING_PHOTO")

	} else {
		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)
		fmt.Println(resp)
		line2logfile("sendPhoto", fmt.Sprintf("RESP: %d %s", resp.StatusCode, buf.String()))
	}

}

// Sending video to chat
func SendVideo(chatID int, text, video string) {

	reqLink := tgApiLinkKEY + "/sendVideo?chat_id=" + fmt.Sprint(chatID) + "&caption=" + text
	reqType := "POST"
	fileDir, _ := os.Getwd()
	fileName := video
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("video", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	r, _ := http.NewRequest(reqType, reqLink, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		line2logfile("sendPhoto", "RESP: ERROR_SENDING_PHOTO")

	} else {
		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)
		fmt.Println(resp)
		line2logfile("sendPhoto", fmt.Sprintf("RESP: %d %s", resp.StatusCode, buf.String()))
	}

}

// Sending document to chat
func SendDocument(chatID int, text, document string) {
	reqLink := tgApiLinkKEY + "/sendDocument?chat_id=" + fmt.Sprintf("%d", chatID) + "&caption=" + text
	reqType := "POST"
	fileDir, _ := os.Getwd()
	fileName := document
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("document", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.Close()

	r, _ := http.NewRequest(reqType, reqLink, body)
	r.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(r)

	if err != nil {
		line2logfile("sendPhoto", "RESP: ERROR_SENDING_PHOTO")

	} else {
		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)
		fmt.Println(resp)
		line2logfile("sendPhoto", fmt.Sprintf("RESP: %d %s", resp.StatusCode, buf.String()))
	}

}

// Loading a file from user message
func LoadFile(chatID int, fileID string) string {

	fmt.Println(chatID, fileID)
	tgApiLinkFile := "https://api.telegram.org/file/bot" + APIKEY
	reqLink := tgApiLinkKEY + "/getFile?file_id=" + fileID
	reqType := "GET"
	var reqBody io.Reader = nil

	line2logfile("updateRequest", fmt.Sprintf("REQV: %s %s %v", reqType, reqLink, reqBody))

	req, err := http.NewRequest(reqType, reqLink, reqBody)
	if err != nil {
		line2logfile("updateRequest", "RESP: REQUEST_ERROR")
	}
	q := req.URL.Query()
	req.URL.RawQuery = q.Encode()

	resp, err := httpGet(req.URL.String())

	if err == nil {

		buf := new(strings.Builder)
		io.Copy(buf, resp.Body)

		line2logfile("updateRequest", fmt.Sprintf("RESP: %d %s", resp.StatusCode, buf.String()))

		var resultFile File
		json.Unmarshal([]byte(buf.String()), &resultFile)
		if resultFile.Ok {
			filePath := resultFile.Result.FilePath
			fileLink := tgApiLinkFile + "/" + filePath
			FileDownload(fileLink, filePath)
			//fmt.Println("Downloaded a file")
			return "downloadedFiles/" + filePath
		}

	}
	return ""
}
func FileDownload(reqString, filePath string) (resp *http.Response, data []byte, contentType string) {

	var err error
	rrReq, _ := http.NewRequest("GET", reqString, nil)
	client := &http.Client{}
	resp, err = client.Do(rrReq)

	if err != nil {
		//fmt.Println("Cannot open addr " + reqString)
		return nil, []byte("Cannot open addr " + reqString), ""
	}
	defer resp.Body.Close()

	contentType = resp.Header["Content-Type"][0]
	line2logfile("fileLoad", fmt.Sprintf("Response status: %s, Content-Type: %12s, Loading URL: %s\n", resp.Status, contentType, reqString))

	data, err = io.ReadAll(resp.Body)

	if err != nil {
		//fmt.Println("Error reading data from remote " + reqString)
		return nil, []byte("Error reading data from remote " + reqString), ""
	}

	StringToFile("downloadedFiles/"+filePath, string(data))
	return resp, data, contentType
}
func StringToFile(fileName string, str string) {
	checkPath(fileName)
	f, _ := os.Create(fileName)

	f.WriteString(str)
	f.Close()
}

// checkPath check if path to file exists and creates path if not. If meant to be path-to-file is directory - returns error
func checkPath(path string) error {
	dir := filepath.Dir(path)
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0750)
		return nil
	}
	if fileInfo.IsDir() {
		return errors.New("ISDIR")
	}
	return err
}
