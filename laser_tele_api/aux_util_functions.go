package laser_tele_api

import (
	"log"
	"os"
)

//auxilary utilty functions

// TG_API_KEY = os.Getenv("TG_API_KEY")
// if TG_API_KEY == "" {
// 	log.Panic("API KEY is not set")
// }

func getEnv(env_var_name string, default_value string) (res string) {
	res = os.Getenv(env_var_name)
	if res == "" && default_value != "" {
		res = default_value
	}
	return
}
func getEnvWithPanic(env_var_name string, is_mandatory bool) (res string) {
	res = getEnv(env_var_name, "")
	if res == "" {
		log.Panic("Mandatory config variable is not defined: " + env_var_name)
	}
	return
}
