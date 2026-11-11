package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

// fakeCandidateAuth stands in for Clerk: the test names the signed-in
// Candidate in a header; no header means "not signed in" (401).
func fakeCandidateAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Test-Candidate")
		if id == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated","message":"Sign in required"}}`))
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithCandidateID(r.Context(), id)))
	})
}

func newCandidateRouter(s *store.Store) http.Handler {
	return handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth})
}

func asCandidate(t *testing.T, h http.Handler, method, path, candidate string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(nil))
	if candidate != "" {
		req.Header.Set("X-Test-Candidate", candidate)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type candidateJSON struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Image  string  `json:"image"`
	Resume *string `json:"resume"`
}

type applicationJSON struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"appliedAt"`
	Job       struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Location string `json:"location"`
		Category string `json:"category"`
		Level    string `json:"level"`
		Salary   int64  `json:"salary"`
	} `json:"job"`
	Company companyJSON `json:"company"`
}

// seedCandidates adds Candidates shaped like the webhook/Mongoose documents.
func seedCandidates(t *testing.T, s *store.Store) {
	t.Helper()
	testdb.Insert(t, s, store.UsersCollection,
		bson.D{{Key: "_id", Value: "user_alice"}, {Key: "name", Value: "Alice"}, {Key: "email", Value: "alice@example.test"},
			{Key: "image", Value: "https://img/alice.png"}, {Key: "resume", Value: "https://res.cloudinary.com/alice.pdf"}, {Key: "__v", Value: int32(0)}},
		bson.D{{Key: "_id", Value: "user_budi"}, {Key: "name", Value: "Budi"}, {Key: "email", Value: "budi@example.test"},
			{Key: "image", Value: "https://img/budi.png"}, {Key: "resume", Value: ""}, {Key: "__v", Value: int32(0)}},
	)
}

func TestGetMe(t *testing.T) {
	s := testdb.New(t)
	seedCandidates(t, s)
	router := newCandidateRouter(s)

	t.Run("with resume", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodGet, "/me", "user_alice")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; %s", rec.Code, rec.Body)
		}
		me := decode[candidateJSON](t, rec)
		if me.ID != "user_alice" || me.Name != "Alice" || me.Resume == nil || *me.Resume != "https://res.cloudinary.com/alice.pdf" {
			t.Errorf("me = %+v", me)
		}
	})

	t.Run("empty resume is null", func(t *testing.T) {
		me := decode[candidateJSON](t, asCandidate(t, router, http.MethodGet, "/me", "user_budi"))
		if me.Resume != nil {
			t.Errorf("resume = %q, want null", *me.Resume)
		}
	})

	t.Run("not synced from Clerk yet", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodGet, "/me", "user_unknown")
		if rec.Code != http.StatusNotFound || decode[errorJSON](t, rec).Error.Code != "candidate_not_found" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("signed out", func(t *testing.T) {
		if rec := asCandidate(t, router, http.MethodGet, "/me", ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}

func TestApplyToJob(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	seedCandidates(t, s)
	router := newCandidateRouter(s)
	path := "/jobs/" + f.newest.Hex() + "/applications"

	rec := asCandidate(t, router, http.MethodPost, path, "user_alice")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	app := decode[applicationJSON](t, rec)
	if app.Status != "Pending" || app.Job.ID != f.newest.Hex() || app.Company.ID != f.acme.Hex() || app.ID == "" {
		t.Errorf("application = %+v", app)
	}
	if time.Since(app.AppliedAt) > time.Minute {
		t.Errorf("appliedAt = %v, want now", app.AppliedAt)
	}

	// Stored exactly like Mongoose stored it.
	var doc bson.M
	if err := s.DB.Collection(store.JobApplicationsCollection).FindOne(context.Background(), bson.D{}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["userId"] != "user_alice" || doc["jobId"] != f.newest || doc["companyId"] != f.acme ||
		doc["status"] != "Pending" || doc["__v"] != int32(0) {
		t.Errorf("stored = %v", doc)
	}
	if _, isDouble := doc["date"].(float64); !isDouble {
		t.Errorf("date stored as %T, want float64 (BSON double, like Mongoose)", doc["date"])
	}

	t.Run("already applied", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodPost, path, "user_alice")
		if rec.Code != http.StatusConflict || decode[errorJSON](t, rec).Error.Code != "already_applied" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("no resume", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodPost, path, "user_budi")
		if rec.Code != http.StatusBadRequest || decode[errorJSON](t, rec).Error.Code != "resume_required" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})

	for name, id := range map[string]string{"hidden job": f.hidden.Hex(), "unknown job": bson.NewObjectID().Hex()} {
		t.Run(name, func(t *testing.T) {
			rec := asCandidate(t, router, http.MethodPost, "/jobs/"+id+"/applications", "user_alice")
			if rec.Code != http.StatusNotFound || decode[errorJSON](t, rec).Error.Code != "job_not_found" {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}

	t.Run("signed out", func(t *testing.T) {
		if rec := asCandidate(t, router, http.MethodPost, path, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("candidate not synced yet", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodPost, path, "user_unknown")
		if rec.Code != http.StatusNotFound || decode[errorJSON](t, rec).Error.Code != "candidate_not_found" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})
}

func TestListMyApplications(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	seedCandidates(t, s)
	router := newCandidateRouter(s)

	app := func(user string, job, company bson.ObjectID, status string, dateMs float64) bson.D {
		return bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "userId", Value: user}, {Key: "companyId", Value: company},
			{Key: "jobId", Value: job}, {Key: "status", Value: status}, {Key: "date", Value: dateMs}, {Key: "__v", Value: int32(0)}}
	}
	base := float64(f.newestPosted.UnixMilli())
	testdb.Insert(t, s, store.JobApplicationsCollection,
		app("user_alice", f.older, f.globex, "Accepted", base-1000),
		app("user_alice", f.hidden, f.acme, "Rejected", base),               // Job hidden after applying: still listed
		app("user_alice", bson.NewObjectID(), f.acme, "Pending", base+1000), // Job deleted: left out
		app("user_budi", f.newest, f.acme, "Pending", base),                 // someone else's
	)

	rec := asCandidate(t, router, http.MethodGet, "/me/applications", "user_alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	apps := decode[[]applicationJSON](t, rec)
	if len(apps) != 2 {
		t.Fatalf("got %d applications, want 2: %+v", len(apps), apps)
	}
	// Newest first.
	if apps[0].Job.ID != f.hidden.Hex() || apps[0].Status != "Rejected" || apps[0].Company.Name != "Acme" {
		t.Errorf("first = %+v", apps[0])
	}
	if apps[1].Job.Title != "Older Job" || apps[1].Status != "Accepted" || apps[1].Company.Name != "Globex" {
		t.Errorf("second = %+v", apps[1])
	}

	t.Run("none yet", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodGet, "/me/applications", "user_nobody")
		if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
			t.Errorf("got %d %q", rec.Code, rec.Body)
		}
	})
}

func TestCandidateRoutesWithoutAuthConfigured(t *testing.T) {
	s := testdb.New(t)
	router := handler.NewRouter(handler.Deps{Store: s}) // no CandidateAuth
	if rec := asCandidate(t, router, http.MethodGet, "/me", "user_alice"); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestCandidateSyncedOnDemand(t *testing.T) {
	s := testdb.New(t)
	first, last := "Citra", "Dewi"
	calls := 0
	fetch := func(_ context.Context, id string) (handler.ClerkUser, error) {
		calls++
		if id != "user_new" {
			return handler.ClerkUser{}, store.ErrNotFound
		}
		u := handler.ClerkUser{ID: id, FirstName: &first, LastName: &last, ImageURL: "https://img.clerk.com/c.png", PrimaryEmailAddressID: "idn_1"}
		u.EmailAddresses = []handler.ClerkEmailAddress{{ID: "idn_1", EmailAddress: "Citra@Example.test"}}
		return u, nil
	}
	router := handler.NewRouter(handler.Deps{Store: s, CandidateAuth: fakeCandidateAuth, FetchClerkUser: fetch})

	// Signed in with Clerk, but the webhook has not created the document yet.
	rec := asCandidate(t, router, http.MethodGet, "/me", "user_new")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	me := decode[candidateJSON](t, rec)
	if me.Name != "Citra Dewi" || me.Email != "citra@example.test" || me.Resume != nil {
		t.Errorf("me = %+v", me)
	}
	doc := candidateDoc(t, s, "user_new")
	if doc["resume"] != "" || doc["__v"] != int32(0) {
		t.Errorf("stored = %v", doc)
	}

	// Now it exists: no second call to Clerk.
	asCandidate(t, router, http.MethodGet, "/me", "user_new")
	if calls != 1 {
		t.Errorf("clerk fetched %d times, want 1", calls)
	}

	t.Run("unknown to Clerk too", func(t *testing.T) {
		rec := asCandidate(t, router, http.MethodGet, "/me", "user_ghost")
		if rec.Code != http.StatusNotFound || decode[errorJSON](t, rec).Error.Code != "candidate_not_found" {
			t.Errorf("got %d %s", rec.Code, rec.Body)
		}
	})
}
