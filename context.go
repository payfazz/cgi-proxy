package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"strings"
	"sync"

	"github.com/payfazz/go-middleware"
	"github.com/payfazz/go-router/defhandler"
	"github.com/payfazz/go-router/method"
	"github.com/payfazz/go-router/path"
	"github.com/payfazz/go-router/segment"

	"github.com/payfazz/cgi-proxy/internal/config"
)

type ctx struct {
	infoLog    *log.Logger
	errLog     *log.Logger
	configPath string
	handler    struct {
		sync.RWMutex
		http.HandlerFunc
	}
}

func newContext(infoLog *log.Logger, errLog *log.Logger, configPath string) *ctx {
	h := &ctx{
		infoLog:    infoLog,
		errLog:     errLog,
		configPath: configPath,
	}
	h.handler.HandlerFunc = defhandler.StatusNotFound

	h.reload()

	return h
}

func (h *ctx) compileRootHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		func() http.HandlerFunc {
			h.handler.RLock()
			defer h.handler.RUnlock()
			return h.handler.HandlerFunc
		}()(w, r)
	}
}

func (h *ctx) reload() {
	if err := func() error {

		conf, err := config.Parse(h.configPath)
		if err != nil {
			return err
		}

		authMiddleware := h.createAuthMiddleware(conf.AuthKeys)

		rootHandler, err := h.createRootRoutingTable(conf.Entry)
		if err != nil {
			return err
		}

		rootHandler = middleware.Compile(
			authMiddleware,
			method.Must("GET", "POST"),
			rootHandler,
		)

		func() {
			h.handler.Lock()
			defer h.handler.Unlock()
			h.handler.HandlerFunc = rootHandler
		}()

		return nil

	}(); err != nil {
		h.errLog.Println("cannot reload config:", err.Error())
	} else {
		h.infoLog.Println("config reloaded !!")
	}
}

func (h *ctx) createAuthMiddleware(keys []string) func(http.HandlerFunc) http.HandlerFunc {
	if len(keys) == 0 {
		h.infoLog.Println("warning: static key is empty, anyone can access the service now")
		return middleware.Nop
	}

	keyMap := make(map[string]struct{})
	for _, item := range keys {
		keyMap[item] = struct{}{}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, _, _ := r.BasicAuth()
			if _, ok := keyMap[user]; ok {
				next(w, r)
				return
			}

			h.err401(w, r)
		}
	}
}

func (h *ctx) createRootRoutingTable(entries []config.Entry) (http.HandlerFunc, error) {
	ret := path.H{}
	for _, item := range entries {
		path := item.Path
		if path != "/" {
			path = strings.TrimSuffix(path, "/")
		}
		if path == "" {
			return nil, fmt.Errorf("path cannot be empty")
		}
		if len(item.Cmd) == 0 {
			return nil, fmt.Errorf("cmd cannot be empty")
		}

		ret[path] = h.compileCGIHandler(item.Cmd, item.AllowParallel, item.AllowSubPath)
	}
	return ret.C(), nil
}

func (h *ctx) compileCGIHandler(args []string, allowParallel bool, allowSubPath bool) http.HandlerFunc {
	var mu sync.Mutex

	handler := &cgi.Handler{
		Path:   args[0],
		Args:   args[1:],
		Logger: h.errLog,
		Stderr: ioutil.Discard,
	}

	subPathMiddlware := segment.MustEnd
	if allowSubPath {
		subPathMiddlware = middleware.Nop
	}

	parallelMiddleware := middleware.Nop
	if !allowParallel {
		parallelMiddleware = func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if !allowParallel {
					mu.Lock()
					defer mu.Unlock()
				}
				next(w, r)
			}
		}
	}

	return middleware.Compile(
		subPathMiddlware,
		parallelMiddleware,
		handler,
	)
}

func (h *ctx) err401(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="cgi-proxy credential"`)
	defhandler.StatusUnauthorized(w, r)
}
