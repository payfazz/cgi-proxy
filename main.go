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

	"github.com/payfazz/cgi-proxy/internal/env"
)

func main() {
	infoLog := log.New(os.Stdout, "INF: ", log.LstdFlags)
	errLog := log.New(os.Stderr, "ERR: ", log.LstdFlags)

	ctx := newContext(infoLog, errLog, env.Get("APP_CONFIG"))
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGHUP)
		for range signalChan {
			ctx.reload()
		}
	}()

	listen := env.Get("APP_LISTEN")
	listenParts := strings.SplitN(listen, ":", 2)
	if len(listenParts) != 2 {
		panic(fmt.Errorf("cannot parse APP_LISTEN: %s", listen))
	}
	infoLog.Printf("listen on %s, pid=%d\n", listen, os.Getpid())

	handler := ctx.compileRootHandler()

	if listenParts[0] == "tcp" {
		panic(http.ListenAndServe(listenParts[1], handler))
	} else {
		listener, err := net.Listen(listenParts[0], listenParts[1])
		if err != nil {
			panic(err)
		}

		ignoreError := false

		if listenParts[0] == "unix" {
			defer os.Remove(listenParts[1])
			go func() {
				signalChan := make(chan os.Signal, 1)
				signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT}
				signal.Notify(signalChan, signals...)
				<-signalChan
				ignoreError = true
				listener.Close()
				signal.Reset(signals...)
			}()
		}

		if err := (&http.Server{Handler: handler}).Serve(listener); err != nil && !ignoreError {
			panic(err)
		}
	}
}
