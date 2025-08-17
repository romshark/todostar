package server

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/events"
	"github.com/romshark/todostar/pkg/timefmt"
	"github.com/romshark/todostar/server/middleware"
	"github.com/romshark/todostar/server/request"
	"github.com/romshark/todostar/server/template"

	datastar "github.com/starfederation/datastar-go/datastar"
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
		slog.Info("serving static from disk (dev mode)")
	} else {
		// Serve embedded in prod.
		staticFiles, err := fs.Sub(staticFS, "static")
		if err != nil {
			panic(err)
		}
		getStatic = http.FileServer(http.FS(staticFiles))
	}

	newHandler := func(pattern string, h http.HandlerFunc) {
		handler := http.Handler(h)
		if accessLog {
			handler = middleware.AccessLog(h)
		}
		m.Handle(pattern, handler)
	}

	// Assets
	m.Handle("GET /static/", http.StripPrefix("/static/", getStatic))

	// Healthcheck
	m.HandleFunc("GET /livez", s.getLivez)
	m.HandleFunc("GET /readyz", s.getReadyz)

	// Pages
	newHandler("GET /", s.getIndex)

	// Streams
	newHandler("GET /stream", s.getIndexStream)

	// Fragments
	newHandler("POST /form/new", s.postFormNew)
	newHandler("POST /form/edit", s.postFormEdit)

	// Commands
	newHandler("DELETE /todo/{id}", s.deleteTodo)
	newHandler("PUT /todo", s.putTodo)
	newHandler("POST /todo/{id}", s.postTodo)

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

func (s *Server) getIndexStream(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)

	var signals SignalsIndex
	if err := datastar.ReadSignals(r, &signals); err != nil {
		slog.Error("reading signals", slog.Any("err", err))
	}

	// Patch the async list first.
	itr, err := s.store.Search(r.Context(), domain.SearchFilters{
		TextMatch: signals.Search.Term,
	})
	if err != nil {
		slog.Error("searching todos", slog.Any("err", err))
		return
	}
	if err = sse.PatchElementTempl(
		template.PartListTodos(itr),
		datastar.WithSelectorID("todos"),
	); err != nil {
		slog.Error("merging fragment", slog.Any("err", err))
		return
	}

	sub := events.OnTodosChanged(func(etc events.EventTodosChanged) {
		itr, err := s.store.Search(r.Context(), domain.SearchFilters{
			TextMatch: signals.Search.Term,
		})
		if err != nil {
			slog.Error("searching todos", slog.Any("err", err))
			return
		}
		if err = sse.PatchElementTempl(template.ViewIndex(itr)); err != nil {
			slog.Error("merging fragment", slog.Any("err", err))
			return
		}
	})
	defer sub.Close()

	<-sse.Context().Done() // Wait until connection is closed.
}

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	if err := template.PageIndex().Render(r.Context(), w); err != nil {
		slog.Error("rendering page index", slog.Any("err", err))
	}
}

func (s *Server) postFormNew(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Title       string `json:"newTitle"`
		Description string `json:"newDescription"`
	}
	err := datastar.ReadSignals(r, &signals)
	if request.IfErrBadRequest(w, err, "bad signals") {
		return
	}

	var msgTitle, msgDescription string
	errVal := domain.Validate(signals.Title, signals.Description)
	if errVal.IsErr() {
		if errVal.TitleEmpty {
			msgTitle = "Title must not be empty"
		}
		if errVal.TitleTooLong {
			msgTitle = "Title is too long"
		}
		if errVal.DescriptionTooLong {
			msgDescription = "Description is too long"
		}
	} else if request.IfErrInternal(w, err, "") {
		// Unexpected error.
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("datastar-selector", "#el_dialogNew") // target element

	if err := template.PartDialogNew(true, msgTitle, msgDescription).
		Render(r.Context(), w); err != nil {
		slog.Error("rendering part dialog new", slog.Any("err", err))
	}
}

func (s *Server) postFormEdit(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Title       string `json:"editTitle"`
		Description string `json:"editDescription"`
	}
	err := datastar.ReadSignals(r, &signals)
	if request.IfErrBadRequest(w, err, "bad signals") {
		return
	}

	var msgTitle, msgDescription string
	errVal := domain.Validate(signals.Title, signals.Description)
	if errVal.IsErr() {
		if errVal.TitleEmpty {
			msgTitle = "Title must not be empty"
		}
		if errVal.TitleTooLong {
			msgTitle = "Title is too long"
		}
		if errVal.DescriptionTooLong {
			msgDescription = "Description is too long"
		}
	} else if request.IfErrInternal(w, err, "") {
		// Unexpected error.
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("datastar-selector", "#el_dialogEdit") // target element

	if err := template.PartDialogEdit(true, msgTitle, msgDescription).
		Render(r.Context(), w); err != nil {
		slog.Error("rendering part dialog new", slog.Any("err", err))
	}
}

func (s *Server) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := request.PathValue[int64](w, r, "id")
	if !ok {
		return
	}

	err := s.store.Delete(r.Context(), id)
	if request.IfErrInternal(w, err, "") {
		return
	}

	// n := events.NotifyTodosChanged()
	// slog.Debug("notified todos changed", slog.Int("clients", n))
}

func (s *Server) putTodo(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		SignalsIndex
		Title       string `json:"newTitle"`
		Description string `json:"newDescription"`
		Due         string `json:"newDue"`
	}
	err := datastar.ReadSignals(r, &signals)
	if request.IfErrBadRequest(w, err, "bad signals") {
		return
	}

	var dueTime time.Time
	if signals.Due != "" {
		dueTime, err = time.Parse(timefmt.TimeFormat, signals.Due)
		if request.IfErrBadRequest(w, err, "invalid due time") {
			return
		}
	}

	_, err = s.store.Add(
		r.Context(), signals.Title, signals.Description, time.Now(), dueTime,
	)
	var errValid domain.ErrorValidation
	if errors.As(err, &errValid) {
		var msgTitle, msgDescription string
		if errValid.TitleEmpty {
			msgTitle = "Title must not be empty"
		}
		if errValid.TitleTooLong {
			msgTitle = "Title is too long"
		}
		if errValid.DescriptionTooLong {
			msgDescription = "Description is too long"
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("datastar-selector", "#el_dialogNew") // target element

		if err := template.PartDialogNew(true, msgTitle, msgDescription).
			Render(r.Context(), w); err != nil {
			slog.Error("rendering part dialog new", slog.Any("err", err))
		}
		return
	} else if request.IfErrInternal(w, err, "") {
		return
	}

	// TODO: Notifying currently bugs out for some reason, needs investigation.
	// n := events.NotifyTodosChanged()
	// slog.Debug("notified todos changed", slog.Int("clients", n))
	patchViewIndex(s, w, r, signals.SignalsIndex)
}

func (s *Server) postTodo(w http.ResponseWriter, r *http.Request) {
	id, ok := request.PathValue[int64](w, r, "id")
	if !ok {
		return
	}

	var signals struct {
		SignalsIndex
		Archived    *bool   `json:"archived"`
		Checked     bool    `json:"editChecked"`
		Title       *string `json:"editTitle"`
		Description *string `json:"editDescription"`
		Due         *string `json:"editDue"`
	}
	err := datastar.ReadSignals(r, &signals)
	if request.IfErrBadRequest(w, err, "bad signals") {
		return
	}

	var due *time.Time
	if signals.Due != nil {
		if *signals.Due != "" {
			tm, err := time.Parse(timefmt.TimeFormat, *signals.Due)
			if request.IfErrBadRequest(w, err, "invalid due time") {
				return
			}
			due = &tm
		} else {
			due = new(time.Time)
		}
	}

	err = s.store.Edit(r.Context(), id, func(t *domain.Todo) error {
		if signals.Checked {
			t.Status = domain.StatusDone
		} else {
			t.Status = domain.StatusOpen
		}

		if signals.Archived != nil {
			t.Archived = *signals.Archived
		}

		if signals.Title != nil {
			t.Title = *signals.Title
		}

		if signals.Description != nil {
			t.Description = *signals.Description
		}

		if due != nil {
			t.Due = *due
		}

		return nil
	})
	var errValid domain.ErrorValidation
	if errors.As(err, &errValid) {
		if request.IfErrBadRequest(w, err, "invalid input") {
			return
		}
	}
	if request.IfErrInternal(w, err, "") {
		return
	}

	// TODO: Notifying currently bugs out for some reason, needs investigation.
	patchViewIndex(s, w, r, signals.SignalsIndex)
	// n := events.NotifyTodosChanged()
	// slog.Debug("notified todos changed", slog.Int("clients", n))
}

type SignalsIndex struct {
	Search struct {
		Term string `json:"term"`
	} `json:"search"`
}

func patchViewIndex(
	s *Server, w http.ResponseWriter, r *http.Request, sig SignalsIndex,
) {
	sse := datastar.NewSSE(w, r)

	// Patch the async list first.
	itr, err := s.store.Search(r.Context(), domain.SearchFilters{
		TextMatch: sig.Search.Term,
	})
	if err != nil {
		slog.Error("searching todos", slog.Any("err", err))
		return
	}
	if err = sse.PatchElementTempl(
		template.PartListTodos(itr),
		datastar.WithSelectorID("todos"),
	); err != nil {
		slog.Error("merging fragment", slog.Any("err", err))
		return
	}

	sub := events.OnTodosChanged(func(etc events.EventTodosChanged) {
		itr, err := s.store.Search(r.Context(), domain.SearchFilters{
			TextMatch: sig.Search.Term,
		})
		if err != nil {
			slog.Error("searching todos", slog.Any("err", err))
			return
		}
		if err = sse.PatchElementTempl(template.ViewIndex(itr)); err != nil {
			slog.Error("merging fragment", slog.Any("err", err))
			return
		}
	})
	defer sub.Close()

	<-sse.Context().Done() // Wait until connection is closed.
}
