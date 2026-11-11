package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	svix "github.com/svix/svix-webhooks/go"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

var testWebhookSecret = "whsec_" + base64.StdEncoding.EncodeToString([]byte("test-signing-secret-32-bytes!!!!"))

// postWebhook sends payload to /webhooks/clerk signed the way Svix signs
// Clerk webhooks. Pass sign=false to send it unsigned.
func postWebhook(t *testing.T, h http.Handler, payload string, sign bool) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/webhooks/clerk", bytes.NewBufferString(payload))
	if sign {
		wh, err := svix.NewWebhook(testWebhookSecret)
		if err != nil {
			t.Fatal(err)
		}
		id, now := "msg_"+bson.NewObjectID().Hex(), time.Now()
		sig, err := wh.Sign(id, now, []byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("svix-id", id)
		req.Header.Set("svix-timestamp", strconv.FormatInt(now.Unix(), 10))
		req.Header.Set("svix-signature", sig)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func newWebhookRouter(s *store.Store) http.Handler {
	return handler.NewRouter(handler.Deps{Store: s, ClerkWebhookSecret: testWebhookSecret})
}

// candidateDoc reads a raw users document, so tests check the stored BSON
// shape, not a Go struct.
func candidateDoc(t *testing.T, s *store.Store, id string) bson.M {
	t.Helper()
	var doc bson.M
	err := s.DB.Collection(store.UsersCollection).FindOne(context.Background(), bson.D{{Key: "_id", Value: id}}).Decode(&doc)
	if err != nil {
		t.Fatalf("find user %s: %v", id, err)
	}
	return doc
}

func userCount(t *testing.T, s *store.Store) int64 {
	t.Helper()
	n, err := s.DB.Collection(store.UsersCollection).CountDocuments(context.Background(), bson.D{})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

const userCreated = `{"type":"user.created","object":"event","data":{
  "id":"user_abc","object":"user","first_name":"Alice","last_name":"Santoso",
  "image_url":"https://img.clerk.com/alice.png",
  "primary_email_address_id":"idn_primary",
  "email_addresses":[
    {"id":"idn_other","email_address":"old@example.test"},
    {"id":"idn_primary","email_address":"Alice@Example.test"}
  ]}}`

func TestClerkWebhookUserCreated(t *testing.T) {
	s := testdb.New(t)
	router := newWebhookRouter(s)

	rec := postWebhook(t, router, userCreated, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
	}

	doc := candidateDoc(t, s, "user_abc")
	want := bson.M{
		"_id": "user_abc", "name": "Alice Santoso",
		"email":  "alice@example.test", // primary address, lowercased like the Mongoose schema
		"image":  "https://img.clerk.com/alice.png",
		"resume": "", "__v": int32(0),
	}
	for k, v := range want {
		if doc[k] != v {
			t.Errorf("%s = %#v, want %#v", k, doc[k], v)
		}
	}
}

func TestClerkWebhookIsIdempotentAndKeepsResume(t *testing.T) {
	s := testdb.New(t)
	router := newWebhookRouter(s)
	postWebhook(t, router, userCreated, true)

	// The Candidate uploads a resume, then Clerk retries user.created.
	_, err := s.DB.Collection(store.UsersCollection).UpdateByID(context.Background(), "user_abc",
		bson.D{{Key: "$set", Value: bson.D{{Key: "resume", Value: "https://res.cloudinary.com/cv.pdf"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if rec := postWebhook(t, router, userCreated, true); rec.Code != http.StatusOK {
		t.Fatalf("retry status = %d", rec.Code)
	}

	if n := userCount(t, s); n != 1 {
		t.Errorf("users = %d, want 1", n)
	}
	if got := candidateDoc(t, s, "user_abc")["resume"]; got != "https://res.cloudinary.com/cv.pdf" {
		t.Errorf("resume = %v, want it kept", got)
	}
}

func TestClerkWebhookUserUpdated(t *testing.T) {
	s := testdb.New(t)
	router := newWebhookRouter(s)
	postWebhook(t, router, userCreated, true)

	// last_name removed, so the name must not become "Alice null".
	updated := `{"type":"user.updated","data":{"id":"user_abc","first_name":"Alicia","last_name":null,
	  "image_url":"https://img.clerk.com/new.png","primary_email_address_id":"idn_1",
	  "email_addresses":[{"id":"idn_1","email_address":"alicia@example.test"}]}}`
	if rec := postWebhook(t, router, updated, true); rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
	}

	doc := candidateDoc(t, s, "user_abc")
	if doc["name"] != "Alicia" || doc["email"] != "alicia@example.test" || doc["image"] != "https://img.clerk.com/new.png" {
		t.Errorf("doc = %v", doc)
	}
}

func TestClerkWebhookNameFallsBackToEmail(t *testing.T) {
	s := testdb.New(t)
	payload := `{"type":"user.created","data":{"id":"user_noname","first_name":null,"last_name":null,
	  "image_url":"https://img.clerk.com/x.png","primary_email_address_id":"idn_1",
	  "email_addresses":[{"id":"idn_1","email_address":"budi.pratama@example.test"}]}}`
	if rec := postWebhook(t, newWebhookRouter(s), payload, true); rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
	}
	if got := candidateDoc(t, s, "user_noname")["name"]; got != "budi.pratama" {
		t.Errorf("name = %v, want budi.pratama", got)
	}
}

func TestClerkWebhookUserDeleted(t *testing.T) {
	s := testdb.New(t)
	router := newWebhookRouter(s)
	postWebhook(t, router, userCreated, true)

	deleted := `{"type":"user.deleted","data":{"id":"user_abc","object":"user","deleted":true}}`
	for i := range 2 { // second delete: already gone, still 200
		if rec := postWebhook(t, router, deleted, true); rec.Code != http.StatusOK {
			t.Fatalf("delete #%d status = %d", i+1, rec.Code)
		}
	}
	if n := userCount(t, s); n != 0 {
		t.Errorf("users = %d, want 0", n)
	}
}

func TestClerkWebhookIgnoresOtherEvents(t *testing.T) {
	s := testdb.New(t)
	rec := postWebhook(t, newWebhookRouter(s), `{"type":"session.created","data":{"id":"sess_1"}}`, true)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestClerkWebhookRejectsBadSignature(t *testing.T) {
	s := testdb.New(t)
	rec := postWebhook(t, newWebhookRouter(s), userCreated, false)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if e := decode[errorJSON](t, rec); e.Error.Code != "invalid_signature" {
		t.Errorf("code = %q", e.Error.Code)
	}
	if n := userCount(t, s); n != 0 {
		t.Errorf("users = %d, want 0 (nothing written)", n)
	}
}

func TestClerkWebhookNotConfigured(t *testing.T) {
	s := testdb.New(t)
	router := handler.NewRouter(handler.Deps{Store: s}) // no secret
	rec := postWebhook(t, router, userCreated, true)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}
