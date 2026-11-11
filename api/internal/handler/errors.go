package handler

import (
	"log/slog"
	"net/http"
)

// apiError is the error body defined by the API contract:
// {"error": {"code": "...", "message": "...", "fields": {...}}}
type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: message}})
}

// writeInternalError logs the real cause and returns a generic 500,
// so internal details never reach the client.
func writeInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	reportError(r, err)
	writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
}

// writeValidationError answers 400 validation_failed with per-field messages.
func writeValidationError(w http.ResponseWriter, fields map[string]string) {
	writeJSON(w, http.StatusBadRequest, map[string]apiError{"error": {
		Code: "validation_failed", Message: "Some fields are invalid", Fields: fields,
	}})
}
