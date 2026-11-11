package upload_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlexMari0/job-portal/api/internal/upload"
)

func TestLocalUploadAndServe(t *testing.T) {
	local, err := upload.NewLocal(t.TempDir(), "http://localhost:8090")
	if err != nil {
		t.Fatal(err)
	}
	png := []byte("\x89PNG\r\n\x1a\n0000IHDR")

	url, err := local.Upload(context.Background(), bytes.NewReader(png), "job-portal/logos")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://localhost:8090/uploads/job-portal/logos/") || !strings.HasSuffix(url, ".png") {
		t.Fatalf("url = %q", url)
	}

	// The returned URL serves the same bytes.
	path := strings.TrimPrefix(url, "http://localhost:8090")
	rec := httptest.NewRecorder()
	local.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Body)
	if rec.Code != http.StatusOK || !bytes.Equal(body, png) {
		t.Errorf("GET %s = %d, %d bytes", path, rec.Code, len(body))
	}

	// Two uploads never share a name.
	other, _ := local.Upload(context.Background(), bytes.NewReader(png), "job-portal/logos")
	if other == url {
		t.Error("second upload overwrote the first")
	}
}

func TestLocalHandlerRefusesListingsAndTraversal(t *testing.T) {
	local, _ := upload.NewLocal(t.TempDir(), "http://localhost:8090")
	_, _ = local.Upload(context.Background(), bytes.NewReader([]byte("%PDF-1.4\n")), "job-portal/resumes")

	for _, path := range []string{"/uploads/", "/uploads/job-portal/resumes/", "/uploads/../go.mod"} {
		rec := httptest.NewRecorder()
		local.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code == http.StatusOK {
			t.Errorf("GET %s = 200, want refusal; body %q", path, rec.Body)
		}
	}
}
