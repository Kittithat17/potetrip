// Package http wires HTTP routes, middleware and JSON helpers.
package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/raisin/travel-split/backend/internal/apperr"
)

const maxJSONBody = 1 << 20

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// writeErr maps domain errors to HTTP responses; anything unexpected is a 500.
func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperr.As(err); ok {
		status := map[apperr.Kind]int{
			apperr.KindInvalid:      http.StatusBadRequest,
			apperr.KindNotFound:     http.StatusNotFound,
			apperr.KindUnauthorized: http.StatusUnauthorized,
			apperr.KindForbidden:    http.StatusForbidden,
			apperr.KindConflict:     http.StatusConflict,
		}[e.Kind]
		writeError(w, status, e.Code, e.Message)
		return
	}
	slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "เกิดข้อผิดพลาดในระบบ")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return apperr.Invalid("invalid_json", "ต้องส่ง JSON body")
		}
		return apperr.Invalid("invalid_json", "รูปแบบ JSON ไม่ถูกต้อง: "+err.Error())
	}
	return nil
}
