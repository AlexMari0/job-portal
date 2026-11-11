package handler_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

// fakeUploader records uploads instead of sending them to Cloudinary.
type fakeUploader struct {
	folder string
	body   []byte
	err    error
}

func (f *fakeUploader) Upload(_ context.Context, file io.Reader, folder string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.folder = folder
	f.body, _ = io.ReadAll(file)
	return "https://res.cloudinary.com/demo/" + folder + "/new.pdf", nil
}

// A minimal file that http.DetectContentType recognises as application/pdf.
var pdfBytes = []byte("%PDF-1.4\n%fake resume\n%%EOF\n")

// pngBytes starts with the PNG signature.
var pngBytes = []byte("\x89PNG\r\n\x1a\n0000IHDR")

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if field != "" {
		fw, err := mw.CreateFormFile(field, filename)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write(content)
	}
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func putResume(t *testing.T, h http.Handler, candidate, field string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, ct := multipartBody(t, field, "cv.pdf", content)
	req := httptest.NewRequest(http.MethodPut, "/me/resume", body)
	req.Header.Set("Content-Type", ct)
	if candidate != "" {
		req.Header.Set("X-Test-Candidate", candidate)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type invalidFileJSON struct {
	Error struct {
		Code   string            `json:"code"`
		Fields map[string]string `json:"fields"`
	} `json:"error"`
}

func TestUploadResume(t *testing.T) {
	s := testdb.New(t)
	seedCandidates(t, s)
	up := &fakeUploader{}
	router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, Uploader: up})

	rec := putResume(t, router, "user_budi", "resume", pdfBytes)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	want := "https://res.cloudinary.com/demo/job-portal/resumes/new.pdf"
	if got := decode[struct {
		Resume string `json:"resume"`
	}](t, rec); got.Resume != want {
		t.Errorf("resume = %q, want %q", got.Resume, want)
	}
	if up.folder != "job-portal/resumes" || !bytes.Equal(up.body, pdfBytes) {
		t.Errorf("uploaded folder=%q body=%q", up.folder, up.body)
	}
	if got := candidateDoc(t, s, "user_budi")["resume"]; got != want {
		t.Errorf("stored resume = %v", got)
	}
}

func TestUploadResumeRejectsInvalidFiles(t *testing.T) {
	s := testdb.New(t)
	seedCandidates(t, s)

	tooBig := append(append([]byte{}, pdfBytes...), bytes.Repeat([]byte("x"), 2<<20)...) // PDF header, > 2 MB
	cases := map[string]struct {
		field   string
		content []byte
	}{
		"not a PDF (checked by content, not name)": {"resume", pngBytes},
		"larger than 2 MB":                         {"resume", tooBig},
		"no file":                                  {"", nil},
		"empty file":                               {"resume", []byte{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			up := &fakeUploader{}
			router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, Uploader: up})
			rec := putResume(t, router, "user_alice", tc.field, tc.content)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; %s", rec.Code, rec.Body)
			}
			e := decode[invalidFileJSON](t, rec)
			if e.Error.Code != "invalid_file" || !strings.Contains(e.Error.Fields["resume"], "PDF") {
				t.Errorf("error = %+v", e.Error)
			}
			if up.body != nil {
				t.Error("file was uploaded despite being invalid")
			}
		})
	}
	if got := candidateDoc(t, s, "user_alice")["resume"]; got != "https://res.cloudinary.com/alice.pdf" {
		t.Errorf("resume changed to %v", got)
	}
}

func TestUploadResumeFailures(t *testing.T) {
	s := testdb.New(t)
	seedCandidates(t, s)

	t.Run("signed out", func(t *testing.T) {
		router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, Uploader: &fakeUploader{}})
		if rec := putResume(t, router, "", "resume", pdfBytes); rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("Cloudinary error keeps the old resume", func(t *testing.T) {
		up := &fakeUploader{err: errors.New("cloudinary down")}
		router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, Uploader: up})
		if rec := putResume(t, router, "user_alice", "resume", pdfBytes); rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
		if got := candidateDoc(t, s, "user_alice")["resume"]; got != "https://res.cloudinary.com/alice.pdf" {
			t.Errorf("resume changed to %v", got)
		}
	})

	t.Run("file Cloudinary cannot read is the user's error", func(t *testing.T) {
		up := &fakeUploader{err: fmt.Errorf("cloudinary upload: %w", handler.ErrFileRejected)}
		router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, Uploader: up})
		rec := putResume(t, router, "user_alice", "resume", pdfBytes)
		if rec.Code != http.StatusBadRequest || decode[invalidFileJSON](t, rec).Error.Code != "invalid_file" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("uploads not configured", func(t *testing.T) {
		router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth})
		if rec := putResume(t, router, "user_alice", "resume", pdfBytes); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 503", rec.Code)
		}
	})
}
