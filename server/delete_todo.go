package server

import (
	"log/slog"
	"net/http"

	"github.com/romshark/todostar/events"
	"github.com/romshark/todostar/server/request"
	"github.com/starfederation/datastar-go/datastar"
)

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
