package laser_tele_api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
