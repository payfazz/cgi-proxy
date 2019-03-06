package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	infoLog := log.New(os.Stdout, "INF: ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Lshortfile)
	errLog := log.New(os.Stderr, "ERR: ", log.LstdFlags|log.Lmicroseconds|log.LUTC|log.Llongfile)
	ctx := newContext(infoLog, errLog)

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGHUP)
		for range signalChan {
			ctx.reload()
		}
	}()

	listen := getEnv("APP_LISTEN")
	infoLog.Printf("listen on %s, pid=%d\n", listen, os.Getpid())
	panic(http.ListenAndServe(listen, ctx.compileHandler()))
}
