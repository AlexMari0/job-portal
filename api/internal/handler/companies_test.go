package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

// Hash of "secret123" made by the Node bcrypt package the Express app uses
// ($2b$ prefix, cost 10): proves existing Company passwords keep working.
const nodeBcryptHash = "$2b$10$vPEteKSAIhxBg0EqOK4Hg.qwRMswXrKcNVvHGevKA3zJ8Ugs73d7."

var testCompanyTokens = auth.NewCompanyTokens("test-secret-at-least-32-bytes-long!!", 30*24*time.Hour)

func newCompanyRouter(s *store.Store) http.Handler {
	return handler.NewRouter(handler.Deps{Store: s, CompanyTokens: testCompanyTokens})
}

func seedCompany(t *testing.T, s *store.Store) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	testdb.Insert(t, s, store.CompaniesCollection, bson.D{
		{Key: "_id", Value: id}, {Key: "name", Value: "Acme"}, {Key: "email", Value: "hr@acme.test"},
		{Key: "image", Value: "https://img/acme.png"}, {Key: "password", Value: nodeBcryptHash}, {Key: "__v", Value: int32(0)},
	})
	return id
}

func postJSON(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func asCompany(t *testing.T, h http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type companyProfileJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Image string `json:"image"`
}

type sessionJSON struct {
	Token     string             `json:"token"`
	ExpiresAt time.Time          `json:"expiresAt"`
	Company   companyProfileJSON `json:"company"`
}

type validationJSON struct {
	Error struct {
		Code   string            `json:"code"`
		Fields map[string]string `json:"fields"`
	} `json:"error"`
}

func TestCompanyLogin(t *testing.T) {
	s := testdb.New(t)
	id := seedCompany(t, s)
	router := newCompanyRouter(s)

	// Email matching is case-insensitive (stored lowercased by Mongoose).
	rec := postJSON(t, router, "/companies/login", `{"email":" HR@Acme.test ","password":"secret123"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	sess := decode[sessionJSON](t, rec)
	want := companyProfileJSON{ID: id.Hex(), Name: "Acme", Email: "hr@acme.test", Image: "https://img/acme.png"}
	if sess.Company != want || sess.Token == "" {
		t.Errorf("session = %+v", sess)
	}
	if d := time.Until(sess.ExpiresAt); d < 29*24*time.Hour {
		t.Errorf("expiresAt in %v, want ~30 days", d)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("response leaks password: %s", rec.Body)
	}

	// The token opens GET /company.
	me := asCompany(t, router, http.MethodGet, "/company", sess.Token)
	if me.Code != http.StatusOK || decode[companyProfileJSON](t, me) != want {
		t.Errorf("GET /company = %d %s", me.Code, me.Body)
	}
}

func TestCompanyLoginRejects(t *testing.T) {
	s := testdb.New(t)
	seedCompany(t, s)
	router := newCompanyRouter(s)

	// Wrong password and unknown email look identical (no account probing).
	for name, body := range map[string]string{
		"wrong password": `{"email":"hr@acme.test","password":"nope"}`,
		"unknown email":  `{"email":"who@acme.test","password":"secret123"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := postJSON(t, router, "/companies/login", body)
			e := decode[errorJSON](t, rec)
			if rec.Code != http.StatusUnauthorized || e.Error.Code != "invalid_credentials" || e.Error.Message != "Invalid email or password" {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}

	t.Run("missing fields", func(t *testing.T) {
		rec := postJSON(t, router, "/companies/login", `{"email":""}`)
		e := decode[validationJSON](t, rec)
		if rec.Code != http.StatusBadRequest || e.Error.Code != "validation_failed" ||
			e.Error.Fields["email"] == "" || e.Error.Fields["password"] == "" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("not JSON", func(t *testing.T) {
		rec := postJSON(t, router, "/companies/login", `email=x`)
		if rec.Code != http.StatusBadRequest || decode[errorJSON](t, rec).Error.Code != "invalid_body" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})
}

func TestGetCompanyRequiresValidSession(t *testing.T) {
	s := testdb.New(t)
	id := seedCompany(t, s)
	router := newCompanyRouter(s)

	if rec := asCompany(t, router, http.MethodGet, "/company", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d", rec.Code)
	}

	// A valid token for a Company that was deleted afterwards: 401, so the
	// web app clears its cookie.
	token, _, _ := testCompanyTokens.Issue(id.Hex())
	if _, err := s.DB.Collection(store.CompaniesCollection).DeleteOne(context.Background(), bson.D{{Key: "_id", Value: id}}); err != nil {
		t.Fatal(err)
	}
	rec := asCompany(t, router, http.MethodGet, "/company", token)
	if rec.Code != http.StatusUnauthorized || decode[errorJSON](t, rec).Error.Code != "unauthenticated" {
		t.Errorf("deleted company: got %d %s", rec.Code, rec.Body)
	}

	// A Candidate's Clerk token is not a Company token.
	if rec := asCompany(t, router, http.MethodGet, "/company", "eyJhbGciOiJSUzI1NiJ9.e30.sig"); rec.Code != http.StatusUnauthorized {
		t.Errorf("foreign token: status = %d", rec.Code)
	}
}

func TestCompanyRoutesWithoutTokensConfigured(t *testing.T) {
	s := testdb.New(t)
	router := handler.NewRouter(handler.Deps{Store: s})
	for _, rec := range []*httptest.ResponseRecorder{
		postJSON(t, router, "/companies/login", `{"email":"hr@acme.test","password":"secret123"}`),
		asCompany(t, router, http.MethodGet, "/company", "x"),
	} {
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want 503", rec.Code)
		}
	}
}

// Node's bcrypt (used by Express) silently cut passwords at 72 bytes, so a
// Company may have signed up with a longer one. Go's bcrypt refuses passwords
// over 72 bytes, so login must cut them the same way to keep working.
func TestCompanyLoginWithPasswordOver72Bytes(t *testing.T) {
	s := testdb.New(t)
	const hashOf80xs = "$2b$10$reN40CqxhgX1whgD9MmJwez3j8Sa.QQn1DEN9gIwK54LU7MP1hmCS" // Node bcrypt
	testdb.Insert(t, s, store.CompaniesCollection, bson.D{
		{Key: "_id", Value: bson.NewObjectID()}, {Key: "name", Value: "Long"}, {Key: "email", Value: "long@pw.test"},
		{Key: "image", Value: "https://img/x.png"}, {Key: "password", Value: hashOf80xs}, {Key: "__v", Value: int32(0)},
	})
	router := newCompanyRouter(s)

	body := `{"email":"long@pw.test","password":"` + strings.Repeat("x", 80) + `"}`
	if rec := postJSON(t, router, "/companies/login", body); rec.Code != http.StatusOK {
		t.Errorf("login = %d %s", rec.Code, rec.Body)
	}
}
