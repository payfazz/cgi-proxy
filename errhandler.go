package main

import (
	"net/http"

	"github.com/payfazz/go-router/defhandler"
)

func (h *ctx) err401(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="cgi-proxy credential"`)
	defhandler.StatusUnauthorized(w, r)
}

func (h *ctx) err404(w http.ResponseWriter, r *http.Request) {
	defhandler.StatusNotFound(w, r)
}

func (h *ctx) err500(w http.ResponseWriter, r *http.Request) {
	defhandler.StatusInternalServerError(w, r)
}
