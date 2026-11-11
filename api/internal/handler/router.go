package handler

import (
	"net/http"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// NewRouter wires every route in the API contract (api/openapi.yaml).
func NewRouter(s *store.Store) http.Handler {
	// Go 1.22+ routing: method and path in one pattern, e.g. "GET /jobs/{id}";
	// r.PathValue("id") reads the {id} segment.
	mux := http.NewServeMux()

	mux.Handle("GET /health", Health(s))
	mux.Handle("GET /jobs", ListJobs(s))
	mux.Handle("GET /jobs/{id}", GetJob(s))

	return mux
}
