package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Local stores uploads on disk and serves them under /uploads/. It stands in
// for Cloudinary in development and end-to-end tests, so nothing is sent to
// a real account. Like Cloudinary's CDN it is a plain file host: browsers
// load the files directly, but never call the API itself through it.
type Local struct {
	dir     string // where files are written
	baseURL string // public origin of this API, e.g. http://localhost:8090
}

func NewLocal(dir, baseURL string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Local{dir: dir, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

// Upload writes the file under folder with a random name (keeping the
// extension of its detected type) and returns its public URL.
func (l *Local) Upload(_ context.Context, file io.Reader, folder string) (string, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	random := make([]byte, 12)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	name := hex.EncodeToString(random) + extension(data)

	if err := os.MkdirAll(filepath.Join(l.dir, filepath.FromSlash(folder)), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(l.dir, filepath.FromSlash(folder), name), data, 0o644); err != nil {
		return "", fmt.Errorf("write upload: %w", err)
	}
	return l.baseURL + "/uploads/" + path.Join(folder, name), nil
}

func extension(data []byte) string {
	exts, _ := mime.ExtensionsByType(http.DetectContentType(data))
	for _, preferred := range []string{".pdf", ".png", ".jpg", ".webp"} {
		for _, e := range exts {
			if e == preferred {
				return e
			}
		}
	}
	if len(exts) > 0 {
		return exts[0]
	}
	return ""
}

// Handler serves GET /uploads/<folder>/<file>. Directory listings are
// refused; http.Dir already blocks paths that escape the directory.
func (l *Local) Handler() http.Handler {
	files := http.StripPrefix("/uploads/", http.FileServer(http.Dir(l.dir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") || strings.Contains(r.URL.Path, "..") {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
}
