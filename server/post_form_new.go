package server

import (
	"log/slog"
	"net/http"

	"github.com/romshark/todostar/domain"
	"github.com/romshark/todostar/server/request"
	"github.com/romshark/todostar/server/template"
	datastar "github.com/starfederation/datastar-go/datastar"
)

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
