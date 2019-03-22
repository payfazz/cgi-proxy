package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	infoLog := log.New(os.Stdout, "INF: ", log.LstdFlags)
	errLog := log.New(os.Stderr, "ERR: ", log.LstdFlags|log.Lshortfile)
	ctx := newContext(infoLog, errLog)

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGHUP)
		for range signalChan {
			ctx.reload()
		}
	}()

	listen := getEnv("APP_LISTEN")
	listenParts := strings.SplitN(listen, ":", 2)
	if len(listenParts) != 2 {
		panic(fmt.Errorf("cannot parse APP_LISTEN: %s", listen))
	}
	infoLog.Printf("listen on %s, pid=%d\n", listen, os.Getpid())

	handler := ctx.compileHandler()

	if listenParts[0] == "tcp" {
		panic(http.ListenAndServe(listenParts[1], handler))
	} else {
		listener, err := net.Listen(listenParts[0], listenParts[1])
		if err != nil {
			panic(err)
		}
		panic((&http.Server{Handler: handler}).Serve(listener))
	}
}
