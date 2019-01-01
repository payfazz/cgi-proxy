package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/payfazz/go-router/method"
	"github.com/payfazz/go-router/path"
	"github.com/payfazz/go-router/segment"
)

func compileHandler(infoLog, errLog *log.Logger) http.HandlerFunc {
	h := &ctx{
		infoLog:     infoLog,
		errLog:      errLog,
		key:         make(map[string]struct{}),
		rootHandler: make(map[string]http.HandlerFunc),
	}

	if err := h.reload(); err != nil {
		h.errLog.Println("cannot load initial config config")
	}

	cgiHandlerTmp := segment.Stripper(h.cgiHandler)

	return path.H{
		"/reload": segment.E(method.H{
			http.MethodGet: h.reloadHandler,
		}.C()),
		"/cgi": method.H{
			http.MethodGet:  cgiHandlerTmp,
			http.MethodPost: cgiHandlerTmp,
		}.C(),
	}.C()
}

func (h *ctx) reloadHandler(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(r) {
		h.err401(w, r)
		return
	}

	if err := h.reload(); err != nil {
		h.err500(w, r)
		return
	}

	fmt.Fprintln(w, "DONE")
}

func (h *ctx) cgiHandler(w http.ResponseWriter, r *http.Request) {
	if !h.allowed(r) {
		h.err401(w, r)
		return
	}

	path := strings.TrimSuffix(r.URL.EscapedPath(), "/")

	h.mu.RLock()
	handler := h.rootHandler[path]
	h.mu.RUnlock()

	if handler == nil {
		h.err404(w, r)
		return
	}

	handler(w, r)
}
