package env

import "os"

var defEnv = map[string]string{
	"APP_LISTEN": "tcp::8080",
	"APP_CONFIG": "./config.yaml",
}

var derivedEnv = map[string]string{}

func init() {
	for key, value := range defEnv {
		realValue := os.Getenv(key)
		if realValue == "" {
			realValue = value
		}
		derivedEnv[key] = realValue
		os.Unsetenv(key)
	}
}

// Get .
func Get(key string) string {
	return derivedEnv[key]
}
