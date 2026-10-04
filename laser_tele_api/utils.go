package laser_tele_api

import (
	"fmt"
	"os"
	"strings"
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
	data, err := os.ReadFile(fname)
	if os.IsNotExist(err) {
		fmt.Println("Can't find file", fname)
		return
	}
	if err != nil {
		fmt.Println("Can't read file", fname, " ", err)
		return
	}
	//remove new line and spaces if they exist
	return strings.TrimSpace(string(data))
}
