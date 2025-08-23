package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/events"
	"github.com/romshark/todostar/server/request"
	"github.com/romshark/todostar/server/template"
	datastar "github.com/starfederation/datastar-go/datastar"
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

	// First patch the shim.
	sse.Patch(template.PartTodos(nil), "part list todos") // Shim
	// And then patch each element with a short delay for animation.
	for i, todo := range todos {
		sse.AppendInto("#todos-list",
			template.PartTodosListItem(todo), "part todos list item")
		if i+1 < len(todos) {
			time.Sleep(80 * time.Millisecond)
		}
	}

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
