package main

import "os"

var defConfig = map[string]string{
	"APP_LISTEN": "tcp::8080",
	"APP_CONFIG": "./config.yaml",
}

var derivedConfig = map[string]string{}

func init() {
	for key, value := range defConfig {
		realValue := os.Getenv(key)
		if realValue == "" {
			realValue = value
		}
		derivedConfig[key] = realValue
		os.Unsetenv(key)
	}
}

func getEnv(key string) string {
	return derivedConfig[key]
}
