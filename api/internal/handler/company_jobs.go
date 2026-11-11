package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/htmlsafe"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// CompanyJobStore is the slice of the store the Manage/Add Job routes need.
type CompanyJobStore interface {
	CreateJob(ctx context.Context, j store.NewJob, now time.Time) (store.Job, error)
	ListCompanyJobs(ctx context.Context, companyID bson.ObjectID) ([]store.JobWithApplicationCount, error)
	SetJobVisibility(ctx context.Context, jobID string, companyID bson.ObjectID, visible bool) (store.JobWithApplicationCount, error)
}

// CompanyJobResponse is a Job as its Company sees it.
type CompanyJobResponse struct {
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

func toCompanyJob(j store.Job, count int) CompanyJobResponse {
	return CompanyJobResponse{
		ID: j.ID.Hex(), Title: j.Title, Location: j.Location, Category: j.Category, Level: j.Level,
		Salary: j.Salary, PostedAt: time.UnixMilli(j.Date).UTC(), Visible: j.Visible, ApplicationCount: count,
	}
}

// companyObjectID reads the signed-in Company's id as an ObjectID.
func companyObjectID(w http.ResponseWriter, r *http.Request) (bson.ObjectID, bool) {
	id, ok := companyID(w, r)
	if !ok {
		return bson.ObjectID{}, false
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "Sign in required")
		return bson.ObjectID{}, false
	}
	return oid, true
}

// decodeJSON reads a small JSON body into v, answering 400 on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Request body must be JSON")
		return false
	}
	return true
}

func oneOf(list []string) string { return "must be one of: " + strings.Join(list, ", ") }

// CreateJob handles POST /company/jobs.
func CreateJob(s CompanyJobStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := companyObjectID(w, r)
		if !ok {
			return
		}
		var body struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Location    string `json:"location"`
			Category    string `json:"category"`
			Level       string `json:"level"`
			Salary      *int64 `json:"salary"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}

		job := store.NewJob{
			Title:       strings.TrimSpace(body.Title),
			Description: htmlsafe.Sanitize(body.Description), // sanitize on write too
			Location:    body.Location, Category: body.Category, Level: body.Level,
			CompanyID: companyID,
		}
		fields := map[string]string{}
		if job.Title == "" {
			fields["title"] = "required"
		}
		// Quill's empty editor is "<p><br></p>": judge by the visible text.
		if htmlsafe.Excerpt(job.Description, 1) == "" {
			fields["description"] = "required"
		}
		if !slices.Contains(store.Locations, job.Location) {
			fields["location"] = oneOf(store.Locations)
		}
		if !slices.Contains(store.Categories, job.Category) {
			fields["category"] = oneOf(store.Categories)
		}
		if !slices.Contains(store.Levels, job.Level) {
			fields["level"] = oneOf(store.Levels)
		}
		if body.Salary == nil || *body.Salary < 0 {
			fields["salary"] = "must be a number of at least 0"
		} else {
			job.Salary = *body.Salary
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		created, err := s.CreateJob(r.Context(), job, time.Now())
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toCompanyJob(created, 0))
	})
}

// ListCompanyJobs handles GET /company/jobs.
func ListCompanyJobs(s CompanyJobStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := companyObjectID(w, r)
		if !ok {
			return
		}
		jobs, err := s.ListCompanyJobs(r.Context(), companyID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		out := make([]CompanyJobResponse, len(jobs))
		for i, j := range jobs {
			out[i] = toCompanyJob(j.Job, j.ApplicationCount)
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// UpdateJobVisibility handles PATCH /company/jobs/{id} with {"visible": bool}.
func UpdateJobVisibility(s CompanyJobStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := companyObjectID(w, r)
		if !ok {
			return
		}
		var body struct {
			Visible *bool `json:"visible"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Visible == nil {
			writeValidationError(w, map[string]string{"visible": "required (true or false)"})
			return
		}

		job, err := s.SetJobVisibility(r.Context(), r.PathValue("id"), companyID, *body.Visible)
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "job_not_found", "Job not found")
		case errors.Is(err, store.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden", "This Job belongs to another Company")
		case err != nil:
			writeInternalError(w, r, err)
		default:
			writeJSON(w, http.StatusOK, toCompanyJob(job.Job, job.ApplicationCount))
		}
	})
}
