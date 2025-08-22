package server

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/a-h/templ"
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

func isDatastarReq(r *http.Request) bool {
	return r.Header.Get("Datastar-Request") == "true"
}

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	startDark := request.ThemeIsDark(r)

	if !isDatastarReq(r) {
		if err := template.PageIndex(startDark).Render(r.Context(), w); err != nil {
			slog.Error("rendering page index", slog.Any("err", err))
		}
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.ReplaceURL(url.URL{Path: "/"}); err != nil {
		slog.Error("replacing url", slog.Any("err", err))
	}
	patch(sse, template.ViewIndex(nil), "view index")

	var signals Signals
	if err := datastar.ReadSignals(r, &signals); err != nil {
		slog.Error("reading signals", slog.Any("err", err))
	}

	todos, err := s.store.Search(r.Context(), domain.SearchFilters{
		TextMatch: signals.Search.Term,
	})
	if err != nil {
		slog.Error("searching todos", slog.Any("err", err))
		return
	}

	patch(sse, template.PartListTodos(todos), "part list todos")

	// Subscribe and keep updating the view until the connection is closed.
	sub := events.OnTodosChanged(func(etc events.EventTodosChanged) {
		todos, err := s.store.Search(r.Context(), domain.SearchFilters{
			TextMatch: signals.Search.Term,
		})
		if err != nil {
			slog.Error("searching todos", slog.Any("err", err))
			return
		}
		if todos == nil {
			todos = []*domain.Todo{}
		}
		patch(sse, template.ViewIndex(todos), "view index")
	})
	defer sub.Close()

	<-sse.Context().Done() // Wait until connection is closed.
}

func (s *Server) getArchive(w http.ResponseWriter, r *http.Request) {
	startDark := request.ThemeIsDark(r)

	if !isDatastarReq(r) {
		if err := template.PageArchive(startDark).Render(r.Context(), w); err != nil {
			slog.Error("rendering page archive", slog.Any("err", err))
		}
		return
	}

	sse := datastar.NewSSE(w, r)
	if err := sse.ReplaceURL(url.URL{Path: "/archive"}); err != nil {
		slog.Error("replacing url", slog.Any("err", err))
	}
	patch(sse, template.ViewArchive(nil), "view archive")

	var signals Signals
	if err := datastar.ReadSignals(r, &signals); err != nil {
		slog.Error("reading signals", slog.Any("err", err))
	}

	todos, err := s.store.Search(r.Context(), domain.SearchFilters{
		TextMatch: signals.Search.Term,
		Archived:  true,
	})
	if err != nil {
		slog.Error("searching archived todos", slog.Any("err", err))
		return
	}

	patch(sse, template.PartListArchivedTodos(todos), "part list archived todos")

	// Subscribe and keep updating the view until the connection is closed.
	sub := events.OnTodosChanged(func(etc events.EventTodosChanged) {
		todos, err := s.store.Search(r.Context(), domain.SearchFilters{
			TextMatch: signals.Search.Term,
			Archived:  true,
		})
		if err != nil {
			slog.Error("searching archived todos", slog.Any("err", err))
			return
		}
		if todos == nil {
			todos = []*domain.Todo{}
		}
		patch(sse, template.ViewArchive(todos), "view archive")
	})
	defer sub.Close()

	<-sse.Context().Done() // Wait until connection is closed.
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
	var signals struct {
		SelectedTodoID int64 `json:"selectedTodoID,omitempty"`
	}
	err := datastar.ReadSignals(r, &signals)
	if request.IfErrBadRequest(w, err, "bad signals") {
		return
	}

	err = s.store.Delete(r.Context(), signals.SelectedTodoID)
	if request.IfErrInternal(w, err, "") {
		return
	}

	n := events.NotifyTodosChanged()
	slog.Debug("notified todos changed", slog.Int("clients", n))
}

func (s *Server) putTodo(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Signals
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

	n := events.NotifyTodosChanged()
	slog.Debug("notified todos changed", slog.Int("clients", n))
}

func (s *Server) postTodo(w http.ResponseWriter, r *http.Request) {
	var signals struct {
		Signals
		SelectedTodoID int64   `json:"selectedTodoID"`
		Archived       *bool   `json:"editArchived"`
		Checked        *bool   `json:"editChecked"`
		Title          *string `json:"editTitle"`
		Description    *string `json:"editDescription"`
		Due            *string `json:"editDue"`
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

	err = s.store.Edit(r.Context(), signals.SelectedTodoID, func(t *domain.Todo) error {
		if signals.Checked != nil {
			if *signals.Checked {
				t.Status = domain.StatusDone
			} else {
				t.Status = domain.StatusOpen
			}
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

	n := events.NotifyTodosChanged()
	slog.Debug("notified todos changed", slog.Int("clients", n))
}

type Signals struct {
	Search struct {
		Term string `json:"term,omitempty"`
	} `json:"search,omitempty"`
}

func patch(
	sse *datastar.ServerSentEventGenerator, comp templ.Component, compName string,
) {
	if err := sse.PatchElementTempl(comp); err != nil {
		slog.Error("patching", slog.String("component", compName), slog.Any("err", err))
		return
	}
}
