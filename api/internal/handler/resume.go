package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// ResumeStore is the slice of the store PUT /me/resume needs.
type ResumeStore interface {
	CandidateStore
	SetCandidateResume(ctx context.Context, id, url string) error
}

var resumeRule = fileRule{
	field:    "resume",
	maxBytes: 2 << 20, // 2 MB
	types:    []string{"application/pdf"},
	message:  "must be a PDF up to 2 MB",
}

// UploadResume handles PUT /me/resume (multipart field "resume").
// The old file stays in Cloudinary, as before the migration.
func UploadResume(s ResumeStore, fetch FetchClerkUser, up Uploader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := candidateID(w, r)
		if !ok {
			return
		}
		// Validate before touching Cloudinary or the database.
		data, err := readFile(w, r, resumeRule)
		if err != nil {
			writeInvalidFile(w, resumeRule)
			return
		}

		if _, err := loadCandidate(r.Context(), s, fetch, id); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "candidate_not_found", "Candidate profile not found")
				return
			}
			writeInternalError(w, r, err)
			return
		}

		url, err := up.Upload(r.Context(), bytes.NewReader(data), "job-portal/resumes")
		if errors.Is(err, ErrFileRejected) {
			writeInvalidFile(w, resumeRule)
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if err := s.SetCandidateResume(r.Context(), id, url); err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"resume": url})
	})
}
