package laser_tele_api

import (
	"log"
	"os"
)

// function makes new log record
func line2logfile(logName, line string) {
	f, err := os.OpenFile(logName+".log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0750)
	if err != nil {
		_, err := os.Create(logName + ".log")
		if err != nil {
			log.Fatal("Error creating log file: ", err)
		}
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println(logName, " ", line)
}
