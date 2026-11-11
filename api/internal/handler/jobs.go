package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/AlexMari0/job-portal/api/internal/htmlsafe"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// JobReader is the slice of the store the public Job handlers need.
type JobReader interface {
	ListVisibleJobs(ctx context.Context) ([]store.JobWithCompany, error)
	GetVisibleJob(ctx context.Context, id string) (store.JobWithCompany, error)
}

// excerptLength is the plain-text preview length shown on Job cards.
const excerptLength = 150

// CompanySummary is the public view of a Company embedded in a Job.
type CompanySummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
}

// JobResponse is a Job as the API contract describes it: camelCase,
// `id` instead of `_id`, ISO 8601 dates, sanitized HTML, nested Company.
type JobResponse struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Excerpt     string         `json:"excerpt"`
	Location    string         `json:"location"`
	Category    string         `json:"category"`
	Level       string         `json:"level"`
	Salary      int64          `json:"salary"`
	PostedAt    time.Time      `json:"postedAt"`
	Company     CompanySummary `json:"company"`
}

func toJobResponse(j store.JobWithCompany) JobResponse {
	return JobResponse{
		ID:          j.ID.Hex(),
		Title:       j.Title,
		Description: htmlsafe.Sanitize(j.Description), // sanitize on read: covers old data too
		Excerpt:     htmlsafe.Excerpt(j.Description, excerptLength),
		Location:    j.Location,
		Category:    j.Category,
		Level:       j.Level,
		Salary:      j.Salary,
		PostedAt:    time.UnixMilli(j.Date).UTC(),
		Company: CompanySummary{
			ID:    j.Company.ID.Hex(),
			Name:  j.Company.Name,
			Image: j.Company.Image,
		},
	}
}

// ListJobs handles GET /jobs.
func ListJobs(jobs JobReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		list, err := jobs.ListVisibleJobs(r.Context())
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		out := make([]JobResponse, len(list))
		for i, j := range list {
			out[i] = toJobResponse(j)
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// GetJob handles GET /jobs/{id}.
func GetJob(jobs JobReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		job, err := jobs.GetVisibleJob(r.Context(), r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "job_not_found", "Job not found")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toJobResponse(job))
	})
}
