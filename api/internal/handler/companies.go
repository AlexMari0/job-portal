package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// CompanyAuthStore is the slice of the store Company login and profile need.
type CompanyAuthStore interface {
	GetCompanyCredentials(ctx context.Context, email string) (store.CompanyCredentials, error)
	GetCompany(ctx context.Context, id string) (store.Company, error)
}

// TokenIssuer signs Company session tokens (auth.CompanyTokens).
type TokenIssuer interface {
	Issue(companyID string) (token string, expiresAt time.Time, err error)
}

// CompanyProfile is the signed-in Company as it sees itself.
type CompanyProfile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Image string `json:"image"`
}

// CompanySession is returned by login and register.
type CompanySession struct {
	Token     string         `json:"token"`
	ExpiresAt time.Time      `json:"expiresAt"`
	Company   CompanyProfile `json:"company"`
}

func toCompanyProfile(c store.Company) CompanyProfile {
	return CompanyProfile{ID: c.ID.Hex(), Name: c.Name, Email: c.Email, Image: c.Image}
}

// dummyHash is compared against when the email is unknown, so a login for a
// missing account takes as long as one with a wrong password.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), 10)

// LoginCompany handles POST /companies/login.
func LoginCompany(s CompanyAuthStore, tokens TokenIssuer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "Request body must be JSON")
			return
		}
		fields := map[string]string{}
		if strings.TrimSpace(body.Email) == "" {
			fields["email"] = "required"
		}
		if body.Password == "" {
			fields["password"] = "required"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		creds, err := s.GetCompanyCredentials(r.Context(), body.Email)
		hash := []byte(creds.PasswordHash)
		if errors.Is(err, store.ErrNotFound) {
			hash = dummyHash
		} else if err != nil {
			writeInternalError(w, r, err)
			return
		}
		// Go's bcrypt reads the $2b$ hashes written by Node's bcrypt.
		if bcrypt.CompareHashAndPassword(hash, []byte(body.Password)) != nil || err != nil {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}

		writeCompanySession(w, r, tokens, creds.Company, http.StatusOK)
	})
}

func writeCompanySession(w http.ResponseWriter, r *http.Request, tokens TokenIssuer, c store.Company, status int) {
	token, expiresAt, err := tokens.Issue(c.ID.Hex())
	if err != nil {
		writeInternalError(w, r, err)
		return
	}
	writeJSON(w, status, CompanySession{Token: token, ExpiresAt: expiresAt.UTC(), Company: toCompanyProfile(c)})
}

// companyID reads the id the Company auth middleware put in the context.
func companyID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.CompanyID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Sign in required")
	}
	return id, ok
}

// GetCurrentCompany handles GET /company. A token for a Company that no
// longer exists is answered with 401, so the web app drops its session.
func GetCurrentCompany(s CompanyAuthStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := companyID(w, r)
		if !ok {
			return
		}
		c, err := s.GetCompany(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", "Sign in required")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toCompanyProfile(c))
	})
}
