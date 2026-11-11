package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlexMari0/job-portal/api/internal/auth"
)

// These cases are rejected before Clerk is contacted, so they run offline.
func TestClerkCandidateRejectsMissingOrMalformedToken(t *testing.T) {
	onFail := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
	h := auth.ClerkCandidate("http://localhost:3000", onFail)(next)

	for name, header := range map[string]string{
		"no header":    "",
		"not a JWT":    "Bearer not-a-jwt",
		"wrong scheme": "Basic dXNlcjpwYXNz",
	} {
		t.Run(name, func(t *testing.T) {
			reached = false
			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401 from onFail", rec.Code)
			}
			if reached {
				t.Error("next handler was called without a valid session")
			}
		})
	}
}
