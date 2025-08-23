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
	"github.com/romshark/todostar/server/template"
	"github.com/starfederation/datastar-go/datastar"
)

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
