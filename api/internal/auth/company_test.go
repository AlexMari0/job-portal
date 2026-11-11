package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/AlexMari0/job-portal/api/internal/auth"
)

const secret = "test-secret-at-least-32-bytes-long!!"

func TestCompanyTokenRoundTrip(t *testing.T) {
	tokens := auth.NewCompanyTokens(secret, 30*24*time.Hour)
	token, expiresAt, err := tokens.Issue("6af3484e75e1e94cf1d738c0")
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expiresAt); d < 29*24*time.Hour || d > 30*24*time.Hour {
		t.Errorf("expiresAt in %v, want ~30 days", d)
	}
	id, err := tokens.Verify(token)
	if err != nil || id != "6af3484e75e1e94cf1d738c0" {
		t.Errorf("Verify = %q, %v", id, err)
	}
}

func TestCompanyTokenRejects(t *testing.T) {
	good := auth.NewCompanyTokens(secret, time.Hour)

	expired, _, _ := auth.NewCompanyTokens(secret, -time.Minute).Issue("c1")
	otherSecret, _, _ := auth.NewCompanyTokens("another-secret-also-32-bytes-long!!", time.Hour).Issue("c1")
	// Same secret but HS384: only HS256 is accepted (no algorithm switching).
	hs384, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.RegisteredClaims{
		Subject: "c1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString([]byte(secret))
	// The Express app's token shape ({id}, no sub) is not accepted.
	legacy, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": "c1", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))

	for name, token := range map[string]string{
		"expired": expired, "other secret": otherSecret, "HS384": hs384,
		"legacy Express token": legacy, "garbage": "not.a.jwt", "empty": "",
	} {
		t.Run(name, func(t *testing.T) {
			if id, err := good.Verify(token); err == nil {
				t.Errorf("accepted, id %q", id)
			}
		})
	}
}

func TestCompanyMiddleware(t *testing.T) {
	tokens := auth.NewCompanyTokens(secret, time.Hour)
	token, _, _ := tokens.Issue("c42")

	var gotID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = auth.CompanyID(r.Context())
	})
	onFail := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	h := tokens.Middleware(onFail)(next)

	for name, tc := range map[string]struct {
		header string
		want   int
		id     string
	}{
		"valid":        {"Bearer " + token, http.StatusOK, "c42"},
		"missing":      {"", http.StatusUnauthorized, ""},
		"wrong scheme": {"Token " + token, http.StatusUnauthorized, ""},
		"invalid":      {"Bearer nope", http.StatusUnauthorized, ""},
	} {
		t.Run(name, func(t *testing.T) {
			gotID = ""
			req := httptest.NewRequest(http.MethodGet, "/company", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want || gotID != tc.id {
				t.Errorf("status %d id %q, want %d %q", rec.Code, gotID, tc.want, tc.id)
			}
		})
	}
}
