package main

import "os"

var defEnv = map[string]string{
	"APP_LISTEN": ":8080",
	"APP_CONFIG": "./config.yaml",
}

func getEnv(key string) string {
	ret := os.Getenv(key)
	if ret == "" {
		ret = defEnv[key]
	}
	return ret
}
