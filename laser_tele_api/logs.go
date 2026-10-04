package laser_tele_api

import (
	"fmt"
	"log"
	"os"
)

// function makes new log record
func line2logfile(logName, line string) {
	f, err := os.OpenFile(logName+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
	if err != nil {
		fmt.Println("Can't open log file: ", err)
		return
	}
	defer f.Close()
	// own logger, so output of the standard logger used by application is not changed
	log.New(f, "", log.LstdFlags).Println(logName, " ", hideKey(line))
}
