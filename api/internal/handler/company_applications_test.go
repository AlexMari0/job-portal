package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/testdb"
)

type companyApplicationJSON struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	AppliedAt time.Time `json:"appliedAt"`
	Candidate struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		Image  string  `json:"image"`
		Resume *string `json:"resume"`
	} `json:"candidate"`
	Job struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Location string `json:"location"`
	} `json:"job"`
}

type applicationFixture struct {
	fixture
	aliceNewest, budiNewest, aliceGlobex, ghost, deletedJob bson.ObjectID
}

func seedApplications(t *testing.T, s *store.Store) applicationFixture {
	t.Helper()
	f := applicationFixture{fixture: seedJobs(t, s)}
	seedCandidates(t, s)
	f.aliceNewest, f.budiNewest, f.aliceGlobex = bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	f.ghost, f.deletedJob = bson.NewObjectID(), bson.NewObjectID()

	base := float64(f.newestPosted.UnixMilli())
	app := func(id bson.ObjectID, user string, job, company bson.ObjectID, status string, date float64) bson.D {
		return bson.D{{Key: "_id", Value: id}, {Key: "userId", Value: user}, {Key: "companyId", Value: company},
			{Key: "jobId", Value: job}, {Key: "status", Value: status}, {Key: "date", Value: date}, {Key: "__v", Value: int32(0)}}
	}
	testdb.Insert(t, s, store.JobApplicationsCollection,
		app(f.aliceNewest, "user_alice", f.newest, f.acme, "Pending", base),
		app(f.budiNewest, "user_budi", f.newest, f.acme, "Accepted", base+1000),
		app(f.aliceGlobex, "user_alice", f.older, f.globex, "Pending", base),
		app(f.ghost, "user_deleted", f.newest, f.acme, "Pending", base),                   // Candidate deleted
		app(f.deletedJob, "user_alice", bson.NewObjectID(), f.acme, "Pending", base+2000), // Job deleted
	)
	return f
}

func TestListCompanyApplications(t *testing.T) {
	s := testdb.New(t)
	f := seedApplications(t, s)
	router := newCompanyRouter(s)

	rec := companyRequest(t, router, http.MethodGet, "/company/applications", tokenFor(t, f.acme), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	apps := decode[[]companyApplicationJSON](t, rec)
	// Acme's only; rows whose Candidate or Job is gone are left out; newest first.
	if len(apps) != 2 || apps[0].ID != f.budiNewest.Hex() || apps[1].ID != f.aliceNewest.Hex() {
		t.Fatalf("apps = %+v", apps)
	}
	alice := apps[1]
	if alice.Candidate.Name != "Alice" || alice.Candidate.Resume == nil || *alice.Candidate.Resume != "https://res.cloudinary.com/alice.pdf" ||
		alice.Job.Title != "Newest Job" || alice.Status != "Pending" {
		t.Errorf("alice = %+v", alice)
	}
	if apps[0].Candidate.Resume != nil {
		t.Errorf("budi has no resume, got %q", *apps[0].Candidate.Resume)
	}
}

func TestSetApplicationStatus(t *testing.T) {
	s := testdb.New(t)
	f := seedApplications(t, s)
	router := newCompanyRouter(s)
	acme := tokenFor(t, f.acme)

	rec := companyRequest(t, router, http.MethodPatch, "/company/applications/"+f.aliceNewest.Hex(), acme, `{"status":"Accepted"}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d; %s", rec.Code, rec.Body)
	}
	var doc bson.M
	if err := s.DB.Collection(store.JobApplicationsCollection).FindOne(context.Background(), bson.D{{Key: "_id", Value: f.aliceNewest}}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["status"] != "Accepted" {
		t.Errorf("stored status = %v", doc["status"])
	}

	for name, tc := range map[string]struct {
		path, body, code string
		status           int
	}{
		// Not found and not yours look the same, as in the Express app.
		"another Company's application": {"/company/applications/" + f.aliceGlobex.Hex(), `{"status":"Rejected"}`, "application_not_found", http.StatusNotFound},
		"unknown application":           {"/company/applications/" + bson.NewObjectID().Hex(), `{"status":"Rejected"}`, "application_not_found", http.StatusNotFound},
		"invalid status":                {"/company/applications/" + f.aliceNewest.Hex(), `{"status":"Hired"}`, "validation_failed", http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			rec := companyRequest(t, router, http.MethodPatch, tc.path, acme, tc.body)
			if rec.Code != tc.status || decode[errorJSON](t, rec).Error.Code != tc.code {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}

	var other bson.M
	_ = s.DB.Collection(store.JobApplicationsCollection).FindOne(context.Background(), bson.D{{Key: "_id", Value: f.aliceGlobex}}).Decode(&other)
	if other["status"] != "Pending" {
		t.Errorf("another Company's application changed to %v", other["status"])
	}

	if rec := companyRequest(t, router, http.MethodGet, "/company/applications", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: %d", rec.Code)
	}
}
