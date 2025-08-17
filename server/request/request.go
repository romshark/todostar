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

// // withJSON parses the request body as json according to model T and calls fn.
// // if fn returns an error or something goes wrong returns true indicating that
// // processing the request should be aborted immediately.
// func withJSON[T any](
// 	w http.ResponseWriter, r *http.Request, fn func(w http.ResponseWriter, data T) error,
// ) (stop bool) {
// 	var data T
// 	err := json.NewDecoder(r.Body).Decode(&data)
// 	if errBadRequest(w, err) {
// 		return true
// 	}
// 	err = fn(w, data)
// 	if errInternal(w, err) {
// 		return true
// 	}
// 	// OK
// 	return false
// }
