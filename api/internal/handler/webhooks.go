package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	svix "github.com/svix/svix-webhooks/go"

	"github.com/AlexMari0/job-portal/api/internal/store"
)

// CandidateSyncer is the slice of the store the Clerk webhook needs.
type CandidateSyncer interface {
	UpsertCandidate(ctx context.Context, p store.CandidateProfile) error
	DeleteCandidate(ctx context.Context, id string) error
}

// maxWebhookBody caps the payload size; Clerk user events are a few KB.
const maxWebhookBody = 1 << 20

// clerkEvent is the envelope of every Clerk webhook.
type clerkEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// clerkUser holds the fields we read from a Clerk user object.
// first_name/last_name are pointers because Clerk sends null for missing names.
type clerkUser struct {
	ID                    string  `json:"id"`
	FirstName             *string `json:"first_name"`
	LastName              *string `json:"last_name"`
	ImageURL              string  `json:"image_url"`
	PrimaryEmailAddressID string  `json:"primary_email_address_id"`
	EmailAddresses        []struct {
		ID           string `json:"id"`
		EmailAddress string `json:"email_address"`
	} `json:"email_addresses"`
}

// primaryEmail returns the address Clerk marks as primary (not just the
// first one), lowercased like the Mongoose schema did.
func (u clerkUser) primaryEmail() string {
	for _, e := range u.EmailAddresses {
		if e.ID == u.PrimaryEmailAddressID {
			return strings.ToLower(e.EmailAddress)
		}
	}
	if len(u.EmailAddresses) > 0 {
		return strings.ToLower(u.EmailAddresses[0].EmailAddress)
	}
	return ""
}

// displayName joins the name parts that exist; with no name at all it
// falls back to the part of the email before "@" (the schema requires a name).
func (u clerkUser) displayName() string {
	var parts []string
	for _, p := range []*string{u.FirstName, u.LastName} {
		if p != nil && strings.TrimSpace(*p) != "" {
			parts = append(parts, strings.TrimSpace(*p))
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	local, _, _ := strings.Cut(u.primaryEmail(), "@")
	return local
}

// ClerkWebhook handles POST /webhooks/clerk: it verifies the Svix signature
// on the raw body, then keeps the users collection in sync with Clerk.
// An empty secret disables it (503), so the API can run locally without one.
func ClerkWebhook(secret string, candidates CandidateSyncer) http.Handler {
	if secret == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusServiceUnavailable, "webhook_not_configured", "Clerk webhook secret is not set")
		})
	}
	wh, err := svix.NewWebhook(secret)
	if err != nil {
		panic("invalid CLERK_WEBHOOK_SIGNING_SECRET: " + err.Error()) // config error: fail at startup
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the exact bytes Clerk sent; re-encoding JSON would break the signature.
		payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "Could not read request body")
			return
		}
		if err := wh.Verify(payload, r.Header); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_signature", "Webhook signature is invalid")
			return
		}

		var ev clerkEvent
		if err := json.Unmarshal(payload, &ev); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", "Webhook body is not valid JSON")
			return
		}
		var user clerkUser
		if err := json.Unmarshal(ev.Data, &user); err != nil || user.ID == "" {
			// Not a user payload: nothing for us to do.
			w.WriteHeader(http.StatusOK)
			return
		}

		switch ev.Type {
		case "user.created", "user.updated":
			err = candidates.UpsertCandidate(r.Context(), store.CandidateProfile{
				ID: user.ID, Name: user.displayName(), Email: user.primaryEmail(), Image: user.ImageURL,
			})
		case "user.deleted":
			err = candidates.DeleteCandidate(r.Context(), user.ID)
		default:
			slog.Debug("ignoring clerk event", "type", ev.Type)
		}
		if err != nil {
			// 5xx makes Clerk retry later, which is what we want for DB errors.
			writeInternalError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
