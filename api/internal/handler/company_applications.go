package handler

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// CompanyApplicationStore is the slice of the store View Applications needs.
type CompanyApplicationStore interface {
	ListCompanyApplications(ctx context.Context, companyID bson.ObjectID) ([]store.CompanyApplication, error)
	SetApplicationStatus(ctx context.Context, id string, companyID bson.ObjectID, status string) error
}

// ApplicantSummary is the Candidate as shown to a Company.
type ApplicantSummary struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	Resume *string `json:"resume"`
}

// CompanyApplicationResponse is an Application as the Company sees it.
type CompanyApplicationResponse struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	AppliedAt time.Time        `json:"appliedAt"`
	Candidate ApplicantSummary `json:"candidate"`
	Job       JobSummary       `json:"job"`
}

// ListCompanyApplications handles GET /company/applications.
func ListCompanyApplications(s CompanyApplicationStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := companyObjectID(w, r)
		if !ok {
			return
		}
		apps, err := s.ListCompanyApplications(r.Context(), companyID)
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		out := make([]CompanyApplicationResponse, len(apps))
		for i, a := range apps {
			applicant := ApplicantSummary{ID: a.Candidate.ID, Name: a.Candidate.Name, Image: a.Candidate.Image}
			if a.Candidate.Resume != "" {
				applicant.Resume = &a.Candidate.Resume
			}
			out[i] = CompanyApplicationResponse{
				ID: a.ID.Hex(), Status: a.Status, AppliedAt: time.UnixMilli(a.Date).UTC(),
				Candidate: applicant, Job: toJobSummary(a.Job),
			}
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// UpdateApplicationStatus handles PATCH /company/applications/{id} with
// {"status": "Pending" | "Accepted" | "Rejected"}; answers 204.
func UpdateApplicationStatus(s CompanyApplicationStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		companyID, ok := companyObjectID(w, r)
		if !ok {
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if !slices.Contains(store.Statuses, body.Status) {
			writeValidationError(w, map[string]string{"status": oneOf(store.Statuses)})
			return
		}

		err := s.SetApplicationStatus(r.Context(), r.PathValue("id"), companyID, body.Status)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "application_not_found", "Application not found")
			return
		}
		if err != nil {
			writeInternalError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
