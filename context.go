package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"strings"
	"sync"

	yaml "gopkg.in/yaml.v2"
)

type ctx struct {
	mu              sync.RWMutex
	infoLog, errLog *log.Logger
	key             map[string]struct{}
	rootHandler     map[string]http.HandlerFunc
}

func (h *ctx) allowed(r *http.Request) bool {
	h.mu.RLock()
	bypass := len(h.key) == 0
	h.mu.RUnlock()

	if bypass {
		return true
	}

	auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

	if len(auth) != 2 {
		return false
	}
	if strings.ToLower(auth[0]) != "basic" {
		return false
	}

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 {
		return false
	}

	h.mu.RLock()
	_, ok := h.key[pair[0]]
	h.mu.RUnlock()

	return ok
}

func (h *ctx) reload() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	confBytes, err := ioutil.ReadFile(getEnv("APP_CONFIG"))
	if err != nil {
		h.errLog.Println(err)
		return err
	}

	var conf struct {
		AuthKeys []string `yaml:"static_key"`
		Entry    []struct {
			Path string   `yaml:"path"`
			Cmd  []string `yaml:"cmd"`
		} `yaml:"entry"`
	}

	if err := yaml.Unmarshal(confBytes, &conf); err != nil {
		h.errLog.Println(err)
		return err
	}

	newKey := make(map[string]struct{})
	for _, item := range conf.AuthKeys {
		newKey[item] = struct{}{}
	}

	newRootHandler := make(map[string]http.HandlerFunc)
	for _, item := range conf.Entry {
		path := item.Path
		if path != "/" {
			path = strings.TrimSuffix(path, "/")
		}
		if path == "" {
			err := fmt.Errorf("parse error: path cannot be empty")
			h.errLog.Println(err)
			return err
		}
		if len(item.Cmd) == 0 {
			err := fmt.Errorf("parse error: cmd cannot be empty")
			h.errLog.Println(err)
			return err
		}
		newRootHandler[path] = h.compileCGIHandler(item.Cmd)
	}

	h.key = newKey
	h.rootHandler = newRootHandler

	h.infoLog.Println("config reloaded !!")
	if len(h.key) == 0 {
		h.infoLog.Println("warning: static key is empty, anyone can access the service now")
	}

	return nil
}

func (h *ctx) compileCGIHandler(args []string) http.HandlerFunc {
	var mu sync.Mutex
	handler := &cgi.Handler{
		Path:   args[0],
		Args:   args[1:],
		Logger: h.errLog,
		Stderr: ioutil.Discard,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// avoid parallel execution
		mu.Lock()
		defer mu.Unlock()

		h.infoLog.Printf("Executing CGI: %#v", args)
		handler.ServeHTTP(w, r)
	}
}
