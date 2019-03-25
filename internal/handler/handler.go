package handler

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/payfazz/go-middleware"
	"github.com/payfazz/go-router/defhandler"
	"github.com/payfazz/go-router/method"
	"github.com/payfazz/go-router/path"
	"github.com/payfazz/go-router/segment"

	"github.com/payfazz/cgi-proxy/internal/config"
)

// Handler .
type Handler struct {
	infoLog    *log.Logger
	errLog     *log.Logger
	configPath string
	handler    struct {
		sync.RWMutex
		http.HandlerFunc
	}
}

// New .
func New(infoLog *log.Logger, errLog *log.Logger, configPath string) *Handler {
	h := &Handler{
		infoLog:    infoLog,
		errLog:     errLog,
		configPath: configPath,
	}
	h.handler.HandlerFunc = defhandler.StatusNotFound

	h.Reload()

	return h
}

// Compile .
func (h *Handler) Compile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		func() http.HandlerFunc {
			h.handler.RLock()
			defer h.handler.RUnlock()
			return h.handler.HandlerFunc
		}()(w, r)
	}
}

// Reload .
func (h *Handler) Reload() {
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

func (h *Handler) createAuthMiddleware(keys []string) func(http.HandlerFunc) http.HandlerFunc {
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

			w.Header().Set("WWW-Authenticate", `Basic realm="cgi-proxy auth"`)
			defhandler.StatusUnauthorized(w, r)
		}
	}
}

func (h *Handler) createRootRoutingTable(entries []config.Entry) (http.HandlerFunc, error) {
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

		ret[path] = h.createCGIHandler(item.Cmd, item.AllowParallel, item.AllowSubPath, item.HijackTCP)
	}
	return ret.C(), nil
}

func (h *Handler) createCGIHandler(cmds []string, allowParallel bool, allowSubPath bool, hijack bool) http.HandlerFunc {
	var mu sync.Mutex
	var handler http.HandlerFunc

	if !hijack {
		handler = (&cgi.Handler{
			Path:   cmds[0],
			Args:   cmds[1:],
			Logger: h.errLog,
			Stderr: ioutil.Discard,
		}).ServeHTTP

	} else {
		handler = h.createHijackHandler(cmds)
	}

	subPathMiddlware := segment.MustEnd
	if allowSubPath {
		subPathMiddlware = middleware.Nop
	}

	parallelMiddleware := middleware.Nop
	if !allowParallel {
		parallelMiddleware = func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
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

func (h *Handler) createHijackHandler(cmds []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !(r.ProtoMajor == 1 && r.ProtoMinor == 1) {
			defhandler.ResponseCodeWithMessage(400, "400 Bad Request: only HTTP/1.1 are supported")(w, r)
			return
		}
		if strings.ToLower(r.Header.Get("Upgrade")) != "tcp" {
			defhandler.ResponseCodeWithMessage(426, "426 Upgrade Required: must upgrade to 'tcp'")(w, r)
			return
		}
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, buff, err := hijacker.Hijack(); err == nil {
				var cmd *exec.Cmd

				defer func() {
					conn.Close()
					if cmd != nil {
						cmd.Wait()
					}
				}()

				fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\n")
				fmt.Fprintf(conn, "Upgrade: tcp\r\n")
				fmt.Fprintf(conn, "Connection: Upgrade\r\n")
				fmt.Fprintf(conn, "\r\n")

				cwd, path := filepath.Split(cmds[0])
				if cwd == "" {
					cwd = "."
				}

				env := []string{
					"HTTP_HOST=" + r.Host,
					"REQUEST_METHOD=" + r.Method,
					"REQUEST_URI=" + r.URL.RequestURI(),
				}

				for k, v := range r.Header {
					k = strings.Map(
						func(r rune) rune {
							switch {
							case 'A' <= r && r <= 'Z':
								return r
							case 'a' <= r && r <= 'z':
								return r - ('a' - 'A')
							default:
								return '_'
							}
						}, k,
					)
					joinStr := ", "
					if k == "COOKIE" {
						joinStr = "; "
					}
					env = append(env, "HTTP_"+k+"="+strings.Join(v, joinStr))
				}

				envPath := os.Getenv("PATH")
				if envPath == "" {
					envPath = "/usr/local/sbin:/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin"
				}
				env = append(env, "PATH="+envPath)

				cmd = &exec.Cmd{
					Path:   path,
					Args:   append([]string{path}, cmds[1:]...),
					Dir:    cwd,
					Env:    env,
					Stdout: conn,
					Stderr: ioutil.Discard,
				}

				buffered := buff.Reader.Buffered()
				if buffered == 0 {
					cmd.Stdin = conn
				} else {
					buffBytes := make([]byte, buffered)
					buff.Reader.Read(buffBytes)
					cmd.Stdin = io.MultiReader(
						bytes.NewReader(buffBytes),
						conn,
					)
				}

				if err := cmd.Start(); err != nil {
					h.errLog.Printf("exec error: %v", err)
					return
				}
				cmd.Process.Wait()

				return
			}
		}

		defhandler.ResponseCodeWithMessage(426, "500 Internal Server Error: cannot hijack connection")(w, r)
	}
}
