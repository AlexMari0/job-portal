package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// CandidateStore is the slice of the store the Candidate handlers need.
type CandidateStore interface {
	GetCandidate(ctx context.Context, id string) (store.Candidate, error)
	UpsertCandidate(ctx context.Context, p store.CandidateProfile) error
	GetVisibleJob(ctx context.Context, id string) (store.JobWithCompany, error)
	HasApplied(ctx context.Context, candidateID string, jobID bson.ObjectID) (bool, error)
	CreateApplication(ctx context.Context, candidateID string, job store.Job, now time.Time) (store.Application, error)
	ListCandidateApplications(ctx context.Context, candidateID string) ([]store.ApplicationWithJob, error)
}

// CandidateResponse is GET /me. Resume is null until the first upload.
type CandidateResponse struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Image  string  `json:"image"`
	Resume *string `json:"resume"`
}

// JobSummary is the part of a Job shown next to an Application.
type JobSummary struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Location string `json:"location"`
	Category string `json:"category"`
	Level    string `json:"level"`
	Salary   int64  `json:"salary"`
}

// ApplicationResponse is an Application as the Candidate sees it.
type ApplicationResponse struct {
	ID        string         `json:"id"`
	Status    string         `json:"status"`
	AppliedAt time.Time      `json:"appliedAt"`
	Job       JobSummary     `json:"job"`
	Company   CompanySummary `json:"company"`
}

func toJobSummary(j store.Job) JobSummary {
	return JobSummary{ID: j.ID.Hex(), Title: j.Title, Location: j.Location, Category: j.Category, Level: j.Level, Salary: j.Salary}
}

func toCompanySummary(c store.Company) CompanySummary {
	return CompanySummary{ID: c.ID.Hex(), Name: c.Name, Image: c.Image}
}

// candidateID reads the id the auth middleware put in the context. Routes
// using it are always wrapped in that middleware, so a miss is a wiring bug.
func candidateID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := auth.CandidateID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Sign in required")
	}
	return id, ok
}

// FetchClerkUser loads a user from the Clerk Backend API; it returns
// store.ErrNotFound when Clerk does not know the id.
type FetchClerkUser func(ctx context.Context, id string) (ClerkUser, error)

// loadCandidate returns the Candidate, creating the document from Clerk
// when the webhook has not done so yet (it can lag behind sign-up, and it
// cannot reach a local machine at all). fetch may be nil: no fallback.
func loadCandidate(ctx context.Context, s CandidateStore, fetch FetchClerkUser, id string) (store.Candidate, error) {
	c, err := s.GetCandidate(ctx, id)
	if !errors.Is(err, store.ErrNotFound) || fetch == nil {
		return c, err
	}
	u, err := fetch(ctx, id)
	if err != nil {
		return store.Candidate{}, err // store.ErrNotFound when Clerk does not know them either
	}
	if err := syncCandidate(ctx, s, u); err != nil {
		if errors.Is(err, errUnusableClerkUser) {
			slog.Warn("clerk user not synced", "id", id, "err", err)
			return store.Candidate{}, store.ErrNotFound // answered as candidate_not_found
		}
		return store.Candidate{}, err
	}
	return s.GetCandidate(ctx, id)
}

// GetMe handles GET /me.
func GetMe(s CandidateStore, fetch FetchClerkUser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := candidateID(w, r)
		if !ok {
			return
		}
		c, err := loadCandidate(r.Context(), s, fetch, id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "candidate_not_found", "Candidate profile not found")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		resp := CandidateResponse{ID: c.ID, Name: c.Name, Email: c.Email, Image: c.Image}
		if c.Resume != "" {
			resp.Resume = &c.Resume
		}
		writeJSON(w, http.StatusOK, resp)
	})
}

// ApplyToJob handles POST /jobs/{id}/applications.
func ApplyToJob(s CandidateStore, fetch FetchClerkUser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := candidateID(w, r)
		if !ok {
			return
		}
		ctx := r.Context()

		candidate, err := loadCandidate(ctx, s, fetch, id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "candidate_not_found", "Candidate profile not found")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}

		job, err := s.GetVisibleJob(ctx, r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job_not_found", "Job not found")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}

		// Enforced here, not only in the UI.
		if candidate.Resume == "" {
			writeError(w, http.StatusBadRequest, "resume_required", "Upload your resume before applying")
			return
		}

		applied, err := s.HasApplied(ctx, id, job.ID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		if applied {
			writeError(w, http.StatusConflict, "already_applied", "You already applied to this Job")
			return
		}

		app, err := s.CreateApplication(ctx, id, job.Job, time.Now())
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, ApplicationResponse{
			ID:        app.ID.Hex(),
			Status:    app.Status,
			AppliedAt: time.UnixMilli(app.Date).UTC(),
			Job:       toJobSummary(job.Job),
			Company:   toCompanySummary(job.Company),
		})
	})
}

// ListMyApplications handles GET /me/applications.
func ListMyApplications(s CandidateStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := candidateID(w, r)
		if !ok {
			return
		}
		apps, err := s.ListCandidateApplications(r.Context(), id)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		out := make([]ApplicationResponse, len(apps))
		for i, a := range apps {
			out[i] = ApplicationResponse{
				ID:        a.ID.Hex(),
				Status:    a.Status,
				AppliedAt: time.UnixMilli(a.Date).UTC(),
				Job:       toJobSummary(a.Job),
				Company:   toCompanySummary(a.Company),
			}
		}
		writeJSON(w, http.StatusOK, out)
	})
}
