package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
)

// Uploader stores a file and returns its public URL (Cloudinary in
// production, a fake in tests).
type Uploader interface {
	Upload(ctx context.Context, file io.Reader, folder string) (url string, err error)
}

// fileRule says which files a multipart field accepts.
type fileRule struct {
	field    string   // form field name, e.g. "resume"
	maxBytes int64    // largest accepted file
	types    []string // accepted types as sniffed from the content
	message  string   // shown to the user on rejection, e.g. "must be a PDF up to 2 MB"
}

var errInvalidFile = errors.New("invalid file")

// ErrFileRejected is returned (wrapped) by an Uploader when the storage
// service cannot process the file itself, e.g. a corrupt PDF. The API then
// answers 400 invalid_file instead of 500.
var ErrFileRejected = errors.New("file rejected by storage")

// multipartOverhead leaves room for boundaries and headers around the file.
const multipartOverhead = 64 << 10

// readFile reads and validates the rule's file from a multipart request,
// entirely in memory (no temp files). The type is detected from the bytes,
// so renaming a .png to .pdf does not get it accepted.
func readFile(w http.ResponseWriter, r *http.Request, rule fileRule) ([]byte, error) {
	limit := rule.maxBytes + multipartOverhead
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := r.ParseMultipartForm(limit); err != nil {
		return nil, errInvalidFile // malformed, or larger than the limit
	}
	defer r.MultipartForm.RemoveAll()

	f, _, err := r.FormFile(rule.field)
	if err != nil {
		return nil, errInvalidFile // field missing
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, rule.maxBytes+1))
	if err != nil || len(data) == 0 || int64(len(data)) > rule.maxBytes {
		return nil, errInvalidFile
	}
	if !slices.Contains(rule.types, http.DetectContentType(data)) {
		return nil, errInvalidFile
	}
	return data, nil
}

func writeInvalidFile(w http.ResponseWriter, rule fileRule) {
	writeJSON(w, http.StatusBadRequest, map[string]apiError{"error": {
		Code:    "invalid_file",
		Message: "The file was rejected",
		Fields:  map[string]string{rule.field: rule.message},
	}})
}
