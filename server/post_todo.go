package server

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/events"
	"github.com/romshark/todostar/pkg/timefmt"
	"github.com/romshark/todostar/server/request"
	"github.com/starfederation/datastar-go/datastar"
)

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
