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

func (s *Server) getArchive(w http.ResponseWriter, r *http.Request) {
	startDark := request.ThemeIsDark(r)

	if !request.IsDS(r) {
		if err := template.PageArchive(startDark).Render(r.Context(), w); err != nil {
			slog.Error("rendering page archive", slog.Any("err", err))
		}
		return
	}

	sse := request.SSE(w, r)
	sse.Patch(template.ViewArchive(nil), "view archive")

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

	// First patch the shim.
	sse.Patch(template.PartArchivedTodos(nil), "part archived todos list") // Shim
	// And then patch each element with a short delay for animation.
	for i, todo := range todos {
		sse.AppendInto("#archived-todos-list",
			template.PartTodosListItem(todo), "part archived todos list item")
		if i+1 < len(todos) {
			time.Sleep(50 * time.Millisecond)
		}
	}

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
		sse.Patch(template.ViewArchive(todos), "view archive")
	})
	defer sub.Close()

	sse.Wait() // Wait until connection is closed.
}
