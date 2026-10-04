package laser_tele_api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LogMode sets what is written to log files
type LogMode int

const (
	// LogWithoutContent (default): requests and responses are logged without texts of messages
	LogWithoutContent LogMode = iota
	// LogFull: full requests and responses with texts of messages are logged, for debugging
	LogFull
	// LogOff: nothing is logged
	LogOff
)

const defaultLogMaxSize = 10 << 20 // 10 MB

// all bots write logs under the mutex, so rotation of a log file is safe even if bots share it
var logMutex sync.Mutex

type logger struct {
	mode    LogMode
	dir     string
	maxSize int64
	apiKey  string
}

func newLogger(config LaserTeleConfigT, apiKey string) (*logger, error) {
	l := &logger{mode: config.LogMode, dir: config.LogDir, maxSize: config.LogMaxSize, apiKey: apiKey}
	if l.maxSize <= 0 {
		l.maxSize = defaultLogMaxSize
	}
	if l.dir != "" && l.mode != LogOff {
		if err := os.MkdirAll(l.dir, 0750); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// write makes new record in log file logName.log.
// If the file is larger than maxSize, it is renamed to logName.log.1 (the previous one is deleted)
func (l *logger) write(logName, line string) {
	if l.mode == LogOff {
		return
	}
	logMutex.Lock()
	defer logMutex.Unlock()

	fileName := filepath.Join(l.dir, logName+".log")
	if info, err := os.Stat(fileName); err == nil && info.Size() >= l.maxSize {
		os.Remove(fileName + ".1")
		if err := os.Rename(fileName, fileName+".1"); err != nil {
			fmt.Println("Can't rotate log file: ", err)
		}
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		fmt.Println("Can't open log file: ", err)
		return
	}
	defer f.Close()
	// own logger, so output of the standard logger used by application is not changed
	log.New(f, "", log.LstdFlags).Println(logName, " ", hideKey(line, l.apiKey))
}

// hideKey replaces bot token in s, so it doesn't get to logs and errors
func hideKey(s, apiKey string) string {
	if apiKey == "" {
		return s
	}
	return strings.ReplaceAll(s, apiKey, "<APIKEY>")
}

// params without texts of messages, they are logged in LogWithoutContent mode
var paramsToLog = []string{
	"chat_id", "message_id", "inline_message_id", "user_id", "offset", "file_id", "ok",
	"callback_query_id", "inline_query_id", "guest_query_id", "shipping_query_id", "pre_checkout_query_id",
	"business_connection_id", "game_short_name",
}

// paramsWithoutContent leaves only params without texts of messages, for LogWithoutContent mode
func paramsWithoutContent(params url.Values) string {
	safe := url.Values{}
	for _, name := range paramsToLog {
		if value, ok := params[name]; ok {
			safe[name] = value
		}
	}
	return safe.Encode()
}

// jsonParamsWithoutContent is paramsWithoutContent for params sent as JSON object
func jsonParamsWithoutContent(body []byte) string {
	var params map[string]json.RawMessage
	if err := json.Unmarshal(body, &params); err != nil {
		return ""
	}
	values := url.Values{}
	for name, value := range params {
		values.Set(name, strings.Trim(string(value), `"`))
	}
	return paramsWithoutContent(values)
}
