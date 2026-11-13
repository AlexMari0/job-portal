package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ClerkEmailAddress is one entry of a Clerk user's email_addresses.
type ClerkEmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
}

// ClerkUser holds the fields we read from a Clerk user object (webhook
// payloads and the Clerk Backend API use the same JSON shape).
// first_name/last_name are pointers because Clerk sends null for missing names.
type ClerkUser struct {
	ID                    string              `json:"id"`
	FirstName             *string             `json:"first_name"`
	LastName              *string             `json:"last_name"`
	ImageURL              string              `json:"image_url"`
	PrimaryEmailAddressID string              `json:"primary_email_address_id"`
	EmailAddresses        []ClerkEmailAddress `json:"email_addresses"`
}

// Profile converts the Clerk user into what we store for a Candidate.
func (u ClerkUser) Profile() store.CandidateProfile {
	return store.CandidateProfile{ID: u.ID, Name: u.displayName(), Email: u.primaryEmail(), Image: u.ImageURL}
}

// primaryEmail returns the address Clerk marks as primary (not just the
// first one), lowercased like the Mongoose schema did.
func (u ClerkUser) primaryEmail() string {
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
func (u ClerkUser) displayName() string {
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
		var user ClerkUser
		if err := json.Unmarshal(ev.Data, &user); err != nil || user.ID == "" {
			// Not a user payload: nothing for us to do.
			w.WriteHeader(http.StatusOK)
			return
		}

		switch ev.Type {
		case "user.created", "user.updated":
			err = syncCandidate(r.Context(), candidates, user)
			if errors.Is(err, errUnusableClerkUser) {
				// Retrying cannot fix this: acknowledge, and report it.
				slog.Warn("clerk user not synced", "id", user.ID, "err", err)
				reportError(r, err)
				err = nil
			}
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

// errUnusableClerkUser marks Clerk users we cannot store as Candidates: no
// email (Candidates need one), or an email another Candidate document uses.
var errUnusableClerkUser = errors.New("clerk user cannot be stored as a Candidate")

type candidateUpserter interface {
	UpsertCandidate(ctx context.Context, p store.CandidateProfile) error
}

// syncCandidate stores a Clerk user as a Candidate.
func syncCandidate(ctx context.Context, candidates candidateUpserter, u ClerkUser) error {
	p := u.Profile()
	if p.Email == "" {
		return fmt.Errorf("%w: %s has no email address", errUnusableClerkUser, u.ID)
	}
	err := candidates.UpsertCandidate(ctx, p)
	if errors.Is(err, store.ErrEmailTaken) {
		return fmt.Errorf("%w: %s uses an email another Candidate has", errUnusableClerkUser, u.ID)
	}
	return err
}
