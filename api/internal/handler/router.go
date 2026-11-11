package handler

import (
	"net/http"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// Deps is everything the routes need.
type Deps struct {
	Store              *store.Store
	ClerkWebhookSecret string // empty disables POST /webhooks/clerk (503)
}

// NewRouter wires every route in the API contract (api/openapi.yaml).
func NewRouter(d Deps) http.Handler {
	// Go 1.22+ routing: method and path in one pattern, e.g. "GET /jobs/{id}";
	// r.PathValue("id") reads the {id} segment.
	mux := http.NewServeMux()

	mux.Handle("GET /health", Health(d.Store))
	mux.Handle("GET /jobs", ListJobs(d.Store))
	mux.Handle("GET /jobs/{id}", GetJob(d.Store))
	mux.Handle("POST /webhooks/clerk", ClerkWebhook(d.ClerkWebhookSecret, d.Store))

	return mux
}
