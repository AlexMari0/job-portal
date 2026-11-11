package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

type companyJobJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Location         string    `json:"location"`
	Category         string    `json:"category"`
	Level            string    `json:"level"`
	Salary           int64     `json:"salary"`
	PostedAt         time.Time `json:"postedAt"`
	Visible          bool      `json:"visible"`
	ApplicationCount int       `json:"applicationCount"`
}

func companyRequest(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func tokenFor(t *testing.T, id bson.ObjectID) string {
	t.Helper()
	token, _, err := testCompanyTokens.Issue(id.Hex())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

const validJob = `{"title":"  Go Engineer ","description":"<p>Build <strong>APIs</strong></p><script>x()</script>",
  "location":"Mumbai","category":"Programming","level":"Beginner level","salary":75000}`

func TestCreateJob(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	router := newCompanyRouter(s)

	rec := companyRequest(t, router, http.MethodPost, "/company/jobs", tokenFor(t, f.acme), validJob)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	job := decode[companyJobJSON](t, rec)
	if job.Title != "Go Engineer" || !job.Visible || job.ApplicationCount != 0 || job.Salary != 75000 || job.Level != "Beginner level" {
		t.Errorf("job = %+v", job)
	}
	if time.Since(job.PostedAt) > time.Minute {
		t.Errorf("postedAt = %v", job.PostedAt)
	}

	// Stored in the Mongoose shape, with the description sanitized on write.
	oid, _ := bson.ObjectIDFromHex(job.ID)
	var doc bson.M
	if err := s.DB.Collection(store.JobsCollection).FindOne(context.Background(), bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["companyId"] != f.acme || doc["visible"] != true || doc["salary"] != int32(75000) || doc["__v"] != int32(0) {
		t.Errorf("stored = %v", doc)
	}
	if _, ok := doc["date"].(float64); !ok {
		t.Errorf("date stored as %T, want float64", doc["date"])
	}
	if doc["description"] != "<p>Build <strong>APIs</strong></p>" {
		t.Errorf("description = %q", doc["description"])
	}

	// It is public right away.
	if rec := get(t, router, "/jobs/"+job.ID); rec.Code != http.StatusOK {
		t.Errorf("GET /jobs/{id} = %d", rec.Code)
	}
}

func TestCreateJobValidation(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	router := newCompanyRouter(s)

	body := `{"title":" ","description":"<p><br></p>","location":"Paris","category":"Cooking","level":"Beginner Level","salary":-1}`
	rec := companyRequest(t, router, http.MethodPost, "/company/jobs", tokenFor(t, f.acme), body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	e := decode[validationJSON](t, rec)
	for _, field := range []string{"title", "description", "location", "category", "level", "salary"} {
		if e.Error.Fields[field] == "" {
			t.Errorf("no error for %s: %v", field, e.Error.Fields)
		}
	}

	if rec := companyRequest(t, router, http.MethodPost, "/company/jobs", "", validJob); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: status = %d", rec.Code)
	}
}

func TestListCompanyJobs(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	app := func(job, company bson.ObjectID) bson.D {
		return bson.D{{Key: "_id", Value: bson.NewObjectID()}, {Key: "userId", Value: "user_x"}, {Key: "companyId", Value: company},
			{Key: "jobId", Value: job}, {Key: "status", Value: "Pending"}, {Key: "date", Value: float64(1)}, {Key: "__v", Value: int32(0)}}
	}
	testdb.Insert(t, s, store.JobApplicationsCollection, app(f.newest, f.acme), app(f.newest, f.acme), app(f.hidden, f.acme), app(f.older, f.globex))
	router := newCompanyRouter(s)

	rec := companyRequest(t, router, http.MethodGet, "/company/jobs", tokenFor(t, f.acme), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	jobs := decode[[]companyJobJSON](t, rec)
	// Acme's Jobs only, Hidden ones included.
	counts := map[string]int{}
	visible := map[string]bool{}
	for _, j := range jobs {
		counts[j.ID], visible[j.ID] = j.ApplicationCount, j.Visible
	}
	if len(jobs) != 2 || counts[f.newest.Hex()] != 2 || counts[f.hidden.Hex()] != 1 || visible[f.hidden.Hex()] {
		t.Errorf("jobs = %+v", jobs)
	}

	empty := companyRequest(t, router, http.MethodGet, "/company/jobs", tokenFor(t, bson.NewObjectID()), "")
	if empty.Code != http.StatusOK || empty.Body.String() != "[]\n" {
		t.Errorf("company without jobs: %d %q", empty.Code, empty.Body)
	}
}

func TestSetJobVisibility(t *testing.T) {
	s := testdb.New(t)
	f := seedJobs(t, s)
	router := newCompanyRouter(s)
	acme := tokenFor(t, f.acme)

	rec := companyRequest(t, router, http.MethodPatch, "/company/jobs/"+f.newest.Hex(), acme, `{"visible":false}`)
	if rec.Code != http.StatusOK || decode[companyJobJSON](t, rec).Visible {
		t.Fatalf("hide: %d %s", rec.Code, rec.Body)
	}
	if rec := get(t, router, "/jobs/"+f.newest.Hex()); rec.Code != http.StatusNotFound {
		t.Errorf("hidden job still public: %d", rec.Code)
	}
	// Explicit value, not a toggle: sending false twice keeps it hidden.
	companyRequest(t, router, http.MethodPatch, "/company/jobs/"+f.newest.Hex(), acme, `{"visible":false}`)
	if rec := get(t, router, "/jobs/"+f.newest.Hex()); rec.Code != http.StatusNotFound {
		t.Errorf("second false made it visible: %d", rec.Code)
	}

	for name, tc := range map[string]struct {
		path, token, body, code string
		status                  int
	}{
		"someone else's job": {"/company/jobs/" + f.older.Hex(), acme, `{"visible":false}`, "forbidden", http.StatusForbidden},
		"unknown job":        {"/company/jobs/" + bson.NewObjectID().Hex(), acme, `{"visible":true}`, "job_not_found", http.StatusNotFound},
		"missing visible":    {"/company/jobs/" + f.newest.Hex(), acme, `{}`, "validation_failed", http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			rec := companyRequest(t, router, http.MethodPatch, tc.path, tc.token, tc.body)
			if rec.Code != tc.status || decode[errorJSON](t, rec).Error.Code != tc.code {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}
}
