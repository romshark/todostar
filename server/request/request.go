package request

import (
	"log/slog"
	"net/http"
	"strconv"
)

// PathValue parses the path value for name and returns (data,true),
// otherwise returns (data,false) indicating that processing the request
// should be aborted immediately.
func PathValue[T ~string | ~int64](
	w http.ResponseWriter, r *http.Request, name string,
) (T, bool) {
	var zero T
	v := r.PathValue(name)
	if v == "" {
		http.Error(w, "missing path value: "+name, http.StatusBadRequest)
		return zero, false
	}
	switch any(zero).(type) {
	case string:
		return any(v).(T), true
	case int64:
		x, err := strconv.ParseInt(v, 10, 64)
		if IfErrBadRequest(w, err, "invalid path value") {
			return zero, false
		}
		return any(x).(T), true
	default:
		http.Error(w, "unsupported type", http.StatusBadRequest)
		return zero, false
	}
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
