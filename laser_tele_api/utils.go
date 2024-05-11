package laser_tele_api

import (
	"fmt"
	"os"
	"regexp"
)

// get start variables from cmd file
func getEnvD(env_var_name string, default_value string) (res string) {
	res = os.Getenv(env_var_name)
	if res == "" {
		res = default_value
	}
	return
}

func loadApiKeyFromFile(fname string) (APIKEY string) {
	//try to read apikey from file named .APIKEY if file exists
	if _, err := os.Stat(fname); err == nil {
		//load data
		file, err := os.Open(fname)
		if err != nil {
			fmt.Println("Can't open file", fname, " ", err)
			return
		}
		defer file.Close()
		//read data
		data := make([]byte, 64)
		count, err := file.Read(data)
		if err != nil {
			fmt.Println("Can't read file", fname, " ", err)
			return
		}
		APIKEY = string(data[:count])
		//remove new line if it exists by regexp
		APIKEY = regexp.MustCompile(`\r?\n`).ReplaceAllString(APIKEY, "")

		return
	} else {
		fmt.Println("Can't find file", fname)
	}
	return
}
