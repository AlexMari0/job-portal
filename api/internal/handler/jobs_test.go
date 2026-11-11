package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

// Response shapes from the API contract.
type companyJSON struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

type jobJSON struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Excerpt     string      `json:"excerpt"`
	Location    string      `json:"location"`
	Category    string      `json:"category"`
	Level       string      `json:"level"`
	Salary      int64       `json:"salary"`
	PostedAt    time.Time   `json:"postedAt"`
	Company     companyJSON `json:"company"`
}

type errorJSON struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// fixture inserts documents shaped exactly like the ones Mongoose writes
// in production: date as a BSON double, salary as int32, a __v field.
type fixture struct {
	acme, globex                  bson.ObjectID
	newest, older, hidden, orphan bson.ObjectID
	newestPosted                  time.Time
}

func seedJobs(t *testing.T, s *store.Store) fixture {
	t.Helper()
	f := fixture{
		acme: bson.NewObjectID(), globex: bson.NewObjectID(),
		newest: bson.NewObjectID(), older: bson.NewObjectID(),
		hidden: bson.NewObjectID(), orphan: bson.NewObjectID(),
		newestPosted: time.UnixMilli(1_762_000_000_000).UTC(),
	}

	testdb.Insert(t, s, store.CompaniesCollection,
		bson.D{{Key: "_id", Value: f.acme}, {Key: "name", Value: "Acme"}, {Key: "email", Value: "hr@acme.test"},
			{Key: "image", Value: "https://img/acme.png"}, {Key: "password", Value: "$2b$10$secret"}, {Key: "__v", Value: int32(0)}},
		bson.D{{Key: "_id", Value: f.globex}, {Key: "name", Value: "Globex"}, {Key: "email", Value: "jobs@globex.test"},
			{Key: "image", Value: "https://img/globex.png"}, {Key: "password", Value: "$2b$10$secret"}, {Key: "__v", Value: int32(0)}},
	)

	job := func(id, company bson.ObjectID, title string, dateMs float64, visible bool, desc string) bson.D {
		return bson.D{
			{Key: "_id", Value: id}, {Key: "title", Value: title}, {Key: "description", Value: desc},
			{Key: "location", Value: "Mumbai"}, {Key: "category", Value: "Programming"},
			{Key: "level", Value: "Senior level"}, {Key: "salary", Value: int32(90000)},
			{Key: "date", Value: dateMs}, {Key: "visible", Value: visible},
			{Key: "companyId", Value: company}, {Key: "__v", Value: int32(0)},
		}
	}
	testdb.Insert(t, s, store.JobsCollection,
		job(f.older, f.globex, "Older Job", float64(f.newestPosted.Add(-48*time.Hour).UnixMilli()), true, "<p>Older</p>"),
		job(f.newest, f.acme, "Newest Job", float64(f.newestPosted.UnixMilli()), true,
			`<p>We are <strong>hiring</strong>.</p><script>alert(1)</script>`),
		job(f.hidden, f.acme, "Hidden Job", float64(f.newestPosted.UnixMilli()), false, "<p>Hidden</p>"),
		// A Job whose Company no longer exists.
		job(f.orphan, bson.NewObjectID(), "Orphan Job", float64(f.newestPosted.UnixMilli()), true, "<p>Orphan</p>"),
	)
	return f
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestListJobs(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	router := handler.NewRouter(handler.Deps{Store: s})

	rec := get(t, router, "/jobs")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	jobs := decode[[]jobJSON](t, rec)

	// Visible Jobs only, newest first; Hidden Jobs and Jobs without a Company are left out.
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want 2: %+v", len(jobs), jobs)
	}
	if jobs[0].ID != f.newest.Hex() || jobs[1].ID != f.older.Hex() {
		t.Errorf("order = [%s %s], want [newest older]", jobs[0].Title, jobs[1].Title)
	}

	got := jobs[0]
	if got.Company != (companyJSON{ID: f.acme.Hex(), Name: "Acme", Image: "https://img/acme.png"}) {
		t.Errorf("company = %+v", got.Company)
	}
	if !got.PostedAt.Equal(f.newestPosted) {
		t.Errorf("postedAt = %v, want %v", got.PostedAt, f.newestPosted)
	}
	if got.Salary != 90000 || got.Level != "Senior level" || got.Location != "Mumbai" || got.Category != "Programming" {
		t.Errorf("fields = %+v", got)
	}
	if got.Description != "<p>We are <strong>hiring</strong>.</p>" {
		t.Errorf("description not sanitized: %q", got.Description)
	}
	if got.Excerpt != "We are hiring." {
		t.Errorf("excerpt = %q", got.Excerpt)
	}
	if body := rec.Body.String(); strings.Contains(body, "password") || strings.Contains(body, "_id") {
		t.Errorf("response leaks internal fields: %s", body)
	}
}

func TestListJobsEmpty(t *testing.T) {
	s := testdb.New(t)
	rec := get(t, handler.NewRouter(handler.Deps{Store: s}), "/jobs")
	if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Errorf("got %d %q, want 200 []", rec.Code, rec.Body)
	}
}

func TestGetJob(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	router := handler.NewRouter(handler.Deps{Store: s})

	t.Run("visible job", func(t *testing.T) {
		rec := get(t, router, "/jobs/"+f.newest.Hex())
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		job := decode[jobJSON](t, rec)
		if job.ID != f.newest.Hex() || job.Company.Name != "Acme" {
			t.Errorf("job = %+v", job)
		}
	})

	notFound := map[string]string{
		"hidden job":          "/jobs/" + f.hidden.Hex(),
		"job without company": "/jobs/" + f.orphan.Hex(),
		"unknown id":          "/jobs/" + bson.NewObjectID().Hex(),
		"malformed id":        "/jobs/not-an-id",
	}
	for name, path := range notFound {
		t.Run(name, func(t *testing.T) {
			rec := get(t, router, path)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if e := decode[errorJSON](t, rec); e.Error.Code != "job_not_found" || e.Error.Message == "" {
				t.Errorf("error = %+v", e)
			}
		})
	}
}
