package server

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/server/middleware"
)

// TODO: instead of doing `if request.IfErrInternal(w, err, "") { return }``
// maybe show an error to the user to indicate something went wrong
// like a toast popping up asking to try again later.

const SSEHeartBeatDur = 25 * time.Second

//go:embed static/*
var staticFS embed.FS

func New(store *domain.Store, accessLog bool) *Server {
	s := &Server{
		store: store,
	}
	m := http.NewServeMux()

	// The files in staticFS are under the "static" directory.
	// strip it so the file server can serve "/static/dist.css"
	var getStatic http.Handler
	if isDevMode() {
		// Serve from disk for instant reloads during development.
		getStatic = middleware.NoCache(
			http.FileServer(http.Dir("./server/static")),
		)
		getStatic = middleware.Brotli(getStatic, 9)
		slog.Info("serving static from disk (dev mode)")
	} else {
		// Serve embedded in prod.
		staticFiles, err := fs.Sub(staticFS, "static")
		if err != nil {
			panic(err)
		}
		getStatic = http.FileServer(http.FS(staticFiles))
		getStatic = middleware.Brotli(getStatic, 9)
	}

	newHandler := func(pattern string, h http.HandlerFunc) {
		handler := http.Handler(h)
		handler = middleware.Brotli(handler, 9)
		if accessLog {
			handler = middleware.AccessLog(h)
		}
		m.Handle(pattern, handler)
	}

	// Assets
	m.Handle("GET /static/", http.StripPrefix("/static/", getStatic))

	// Healthcheck
	m.HandleFunc("GET /livez/{$}", s.getLivez)
	m.HandleFunc("GET /readyz/{$}", s.getReadyz)

	// Pages
	newHandler("GET /", s.getIndex)
	newHandler("GET /archive/{$}", s.getArchive)

	// Fragments
	newHandler("POST /form/new/{$}", s.postFormNew)
	newHandler("POST /form/edit/{$}", s.postFormEdit)

	// Actions
	newHandler("DELETE /todo/{$}", s.deleteTodo)
	newHandler("PUT /todo/{$}", s.putTodo)
	newHandler("POST /todo/{$}", s.postTodo)

	s.mux = m
	return s
}

func isDevMode() bool { return os.Getenv("TEMPL_DEV_MODE") != "" }

type Server struct {
	mux   *http.ServeMux
	store *domain.Store
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) getReadyz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("ready"))
}

func (s *Server) getLivez(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

type Signals struct {
	Search struct {
		Term string `json:"term,omitempty"`
	} `json:"search,omitempty"`
}
