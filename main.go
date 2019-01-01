package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	infoLog := log.New(os.Stdout, "INF: ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lshortfile)
	errLog := log.New(os.Stderr, "ERR: ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Llongfile)
	listen := getEnv("APP_LISTEN")
	infoLog.Println("listen on " + listen)
	panic(http.ListenAndServe(listen, compileHandler(infoLog, errLog)))
}
