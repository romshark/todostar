package server

import (
	"log/slog"
	"net/http"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/events"
	"github.com/romshark/todostar/server/request"
	"github.com/romshark/todostar/server/template"
	"github.com/starfederation/datastar-go/datastar"
)

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	startDark := request.ThemeIsDark(r)

	if !request.IsDS(r) {
		if err := template.PageIndex(startDark).Render(r.Context(), w); err != nil {
			slog.Error("rendering page index", slog.Any("err", err))
		}
		return
	}

	sse := request.SSE(w, r)
	sse.Patch(template.ViewIndex(nil), "view index")

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

	sse.Patch(template.PartTodos(todos), "part list todos")

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
		sse.Patch(template.ViewIndex(todos), "view index")
	})
	defer sub.Close()

	sse.Wait() // Wait until connection is closed.
}
