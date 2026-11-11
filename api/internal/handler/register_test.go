package handler_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/crypto/bcrypt"

	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

func registerRequest(t *testing.T, h http.Handler, fields map[string]string, image []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if image != nil {
		fw, _ := mw.CreateFormFile("image", "logo.png")
		_, _ = fw.Write(image)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/companies", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newRegisterRouter(s *store.Store, up handler.Uploader) http.Handler {
	return handler.NewRouter(handler.Deps{Store: s, CompanyTokens: testCompanyTokens, Uploader: up})
}

var validCompany = map[string]string{"name": "  Initech ", "email": "Jobs@Initech.test", "password": "secret123"}

func TestRegisterCompany(t *testing.T) {
	s := testdb.New(t)
	up := &fakeUploader{}
	router := newRegisterRouter(s, up)

	rec := registerRequest(t, router, validCompany, pngBytes)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	sess := decode[sessionJSON](t, rec)
	if sess.Token == "" || sess.Company.Name != "Initech" || sess.Company.Email != "jobs@initech.test" ||
		sess.Company.Image != "https://res.cloudinary.com/demo/job-portal/logos/new.pdf" {
		t.Errorf("session = %+v", sess)
	}
	if up.folder != "job-portal/logos" || !bytes.Equal(up.body, pngBytes) {
		t.Errorf("uploaded folder=%q", up.folder)
	}

	// Stored like Mongoose did: lowercased email, bcrypt cost 10, __v.
	var doc bson.M
	if err := s.DB.Collection(store.CompaniesCollection).FindOne(context.Background(), bson.D{{Key: "email", Value: "jobs@initech.test"}}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	hash, _ := doc["password"].(string)
	if cost, err := bcrypt.Cost([]byte(hash)); err != nil || cost != 10 || bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret123")) != nil {
		t.Errorf("password hash %q (cost %d, %v)", hash, cost, err)
	}
	if doc["name"] != "Initech" || doc["__v"] != int32(0) {
		t.Errorf("stored = %v", doc)
	}

	// The token works, and so does logging in with the new password.
	if rec := asCompany(t, router, http.MethodGet, "/company", sess.Token); rec.Code != http.StatusOK {
		t.Errorf("GET /company = %d", rec.Code)
	}
	if rec := postJSON(t, router, "/companies/login", `{"email":"jobs@initech.test","password":"secret123"}`); rec.Code != http.StatusOK {
		t.Errorf("login = %d %s", rec.Code, rec.Body)
	}
}

func TestRegisterCompanyRejects(t *testing.T) {
	s := testdb.New(t)
	seedCompany(t, s) // hr@acme.test

	t.Run("email taken (any case), nothing uploaded", func(t *testing.T) {
		up := &fakeUploader{}
		fields := map[string]string{"name": "Acme 2", "email": "HR@ACME.test", "password": "secret123"}
		rec := registerRequest(t, newRegisterRouter(s, up), fields, pngBytes)
		if rec.Code != http.StatusConflict || decode[errorJSON](t, rec).Error.Code != "email_taken" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
		if up.body != nil {
			t.Error("logo uploaded for a failed registration")
		}
	})

	t.Run("invalid fields", func(t *testing.T) {
		up := &fakeUploader{}
		rec := registerRequest(t, newRegisterRouter(s, up), map[string]string{"name": " ", "email": "not-an-email", "password": "12345"}, pngBytes)
		e := decode[validationJSON](t, rec)
		if rec.Code != http.StatusBadRequest || e.Error.Code != "validation_failed" ||
			e.Error.Fields["name"] == "" || e.Error.Fields["email"] == "" || e.Error.Fields["password"] == "" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
		if up.body != nil {
			t.Error("logo uploaded despite invalid fields")
		}
	})

	for name, image := range map[string][]byte{
		"logo is a PDF":  pdfBytes,
		"logo over 1 MB": append(append([]byte{}, pngBytes...), bytes.Repeat([]byte("x"), 1<<20)...),
		"logo missing":   nil,
	} {
		t.Run(name, func(t *testing.T) {
			up := &fakeUploader{}
			rec := registerRequest(t, newRegisterRouter(s, up), validCompany, image)
			e := decode[invalidFileJSON](t, rec)
			if rec.Code != http.StatusBadRequest || e.Error.Code != "invalid_file" || !strings.Contains(e.Error.Fields["image"], "PNG") {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
			if up.body != nil {
				t.Error("invalid logo uploaded")
			}
		})
	}

	n, _ := s.DB.Collection(store.CompaniesCollection).CountDocuments(context.Background(), bson.D{})
	if n != 1 {
		t.Errorf("companies = %d, want 1 (nothing created)", n)
	}
}

func TestRegisterCompanyNeedsUploads(t *testing.T) {
	s := testdb.New(t)
	router := handler.NewRouter(handler.Deps{Store: s, CompanyTokens: testCompanyTokens}) // no uploader
	if rec := registerRequest(t, router, validCompany, pngBytes); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}
