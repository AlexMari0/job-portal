package handler

import (
	"net/http"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/store"
)

// Middleware wraps a handler, e.g. to require authentication.
type Middleware func(http.Handler) http.Handler

// Deps is everything the routes need.
type Deps struct {
	Store              *store.Store
	ClerkWebhookSecret string // empty disables POST /webhooks/clerk (503)

	// CandidateAuth requires a signed-in Candidate and puts their id in the
	// request context (auth.ClerkCandidate in production, a fake in tests).
	// Nil disables the Candidate routes (503).
	CandidateAuth Middleware
	// FetchClerkUser creates a missing Candidate document from Clerk on
	// demand. Nil: rely on the webhook only.
	FetchClerkUser FetchClerkUser
	// Uploader stores uploaded files (Cloudinary). Nil disables uploads (503).
	Uploader Uploader
	// Files serves locally stored uploads at /uploads/ (development and
	// end-to-end tests only, when uploads go to disk instead of Cloudinary).
	Files http.Handler

	// CompanyTokens issues and verifies Company session tokens.
	// Nil disables Company login and the /company routes (503).
	CompanyTokens *auth.CompanyTokens
}

// NewRouter wires every route in the API contract (api/openapi.yaml).
func NewRouter(d Deps) http.Handler {
	// Go 1.22+ routing: method and path in one pattern, e.g. "GET /jobs/{id}";
	// r.PathValue("id") reads the {id} segment.
	mux := http.NewServeMux()

	candidate := d.CandidateAuth
	if candidate == nil {
		candidate = unavailable("auth_not_configured", "Candidate sign-in is not configured")
	}

	mux.Handle("GET /health", Health(d.Store))
	mux.Handle("GET /jobs", ListJobs(d.Store))
	mux.Handle("GET /jobs/{id}", GetJob(d.Store))

	// Candidate routes: Clerk session token required.
	mux.Handle("GET /me", candidate(GetMe(d.Store, d.FetchClerkUser)))
	mux.Handle("GET /me/applications", candidate(ListMyApplications(d.Store)))
	mux.Handle("POST /jobs/{id}/applications", candidate(ApplyToJob(d.Store, d.FetchClerkUser)))
	mux.Handle("PUT /me/resume", candidate(withUploads(d.Uploader, func(up Uploader) http.Handler {
		return UploadResume(d.Store, d.FetchClerkUser, up)
	})))

	// Company routes: Company session token (JWT from POST /companies/login).
	if d.CompanyTokens != nil {
		company := d.CompanyTokens.Middleware(Unauthenticated)
		mux.Handle("POST /companies/login", LoginCompany(d.Store, d.CompanyTokens))
		mux.Handle("POST /companies", withUploads(d.Uploader, func(up Uploader) http.Handler {
			return RegisterCompany(d.Store, d.CompanyTokens, up)
		}))
		mux.Handle("GET /company", company(GetCurrentCompany(d.Store)))
		mux.Handle("GET /company/jobs", company(ListCompanyJobs(d.Store)))
		mux.Handle("POST /company/jobs", company(CreateJob(d.Store)))
		mux.Handle("PATCH /company/jobs/{id}", company(UpdateJobVisibility(d.Store)))
		mux.Handle("GET /company/applications", company(ListCompanyApplications(d.Store)))
		mux.Handle("PATCH /company/applications/{id}", company(UpdateApplicationStatus(d.Store)))
	} else {
		off := unavailable("company_auth_not_configured", "Company sign-in is not configured")(nil)
		mux.Handle("POST /companies/login", off)
		mux.Handle("POST /companies", off)
		mux.Handle("GET /company", off)
		mux.Handle("/company/", off)
	}

	if d.Files != nil {
		mux.Handle("GET /uploads/", d.Files)
	}

	mux.Handle("POST /webhooks/clerk", ClerkWebhook(d.ClerkWebhookSecret, d.Store))

	return mux
}

// Unauthenticated answers 401 in the contract's error format; used when a
// Clerk session token is missing or invalid.
var Unauthenticated http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusUnauthorized, "unauthenticated", "Sign in required")
})

// unavailable is middleware that rejects every request with 503.
func unavailable(code, message string) Middleware {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusServiceUnavailable, code, message)
		})
	}
}

// withUploads builds an upload handler, or a 503 one when uploads are not configured.
func withUploads(up Uploader, build func(Uploader) http.Handler) http.Handler {
	if up == nil {
		return unavailable("uploads_not_configured", "File uploads are not configured")(nil)
	}
	return build(up)
}
