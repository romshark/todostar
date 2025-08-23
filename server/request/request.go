package request

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/starfederation/datastar-go/datastar"
)

// IsDS returns true if this request was made by Datastar.
func IsDS(r *http.Request) bool {
	return r.Header.Get("Datastar-Request") == "true"
}

func IfErrInternal(w http.ResponseWriter, err error, msg string) (stop bool) {
	if err == nil {
		return false
	}
	slog.Error("internal error", slog.Any("err", err))
	if msg == "" {
		msg = http.StatusText(http.StatusInternalServerError)
	}
	slog.Debug("internal error", slog.Any("err", err), slog.String("msg", msg))
	http.Error(w, msg, http.StatusInternalServerError)
	return true
}

func IfErrBadRequest(w http.ResponseWriter, err error, msg string) (stop bool) {
	if err == nil {
		return false
	}
	if msg == "" {
		msg = http.StatusText(http.StatusBadRequest)
	}
	slog.Debug("bad request", slog.Any("err", err), slog.String("msg", msg))
	http.Error(w, msg, http.StatusBadRequest)
	return true
}

// ThemeIsDark reads from the cookie "themeisdark" whether to prefer rendering in dark.
func ThemeIsDark(r *http.Request) bool {
	c, err := r.Cookie("themeisdark")
	if err != nil || c.Value != "1" {
		return false
	}
	return true
}

type SSEHandle struct {
	sse *datastar.ServerSentEventGenerator
}

func SSE(w http.ResponseWriter, r *http.Request, opts ...datastar.SSEOption) SSEHandle {
	return SSEHandle{sse: datastar.NewSSE(w, r, opts...)}
}

// Patch patches an element on the page.
func (h SSEHandle) Patch(
	comp templ.Component, compName string,
	options ...datastar.PatchElementOption,
) (ok bool) {
	if err := h.sse.PatchElementTempl(comp, options...); err != nil {
		slog.Error("patch", slog.String("component", compName), slog.Any("err", err))
		return false
	}
	return true
}

// Remove removes an element on the page by selector.
func (h SSEHandle) Remove(
	selector string, opts ...datastar.PatchElementOption,
) (ok bool) {
	if err := h.sse.RemoveElement(selector, opts...); err != nil {
		slog.Error("patch remove", slog.String("selector", selector), slog.Any("err", err))
		return false
	}
	return true
}

// Wait waits until the sse request is canceled.
func (h SSEHandle) Wait() { <-h.sse.Context().Done() }

func (h SSEHandle) AppendInto(
	containerSelector string, comp templ.Component, compName string,
) (ok bool) {
	if err := h.sse.PatchElementTempl(comp,
		datastar.WithModeInner(),
		datastar.WithModeAppend(),
		datastar.WithSelector(containerSelector)); err != nil {
		slog.Error("patch append into",
			slog.String("container selector", containerSelector),
			slog.Any("err", err))
		return false
	}
	return true
}
