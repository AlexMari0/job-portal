// Package auth identifies who is calling the API. Handlers only read the
// caller's id from the request context; the middleware that puts it there
// (Clerk in production, a fake in tests) is chosen when the router is built.
package auth

import (
	"context"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

type candidateKey struct{}

// WithCandidateID returns a context carrying the signed-in Candidate's id
// (their Clerk user id).
func WithCandidateID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, candidateKey{}, id)
}

// CandidateID returns the signed-in Candidate's id, if any.
func CandidateID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(candidateKey{}).(string)
	return id, ok && id != ""
}

// ClerkCandidate returns middleware that requires a valid Clerk session
// token in "Authorization: Bearer <token>". The token's authorized party
// (azp) must be webOrigin, the Next.js site that obtained it; this blocks
// tokens minted for other sites. onFail answers requests without a valid token.
//
// clerk.SetKey(CLERK_SECRET_KEY) must be called before serving requests.
func ClerkCandidate(webOrigin string, onFail http.Handler) func(http.Handler) http.Handler {
	// WithHeaderAuthorization (not RequireHeaderAuthorization, which answers a
	// missing token with a bare 403) so every rejection goes through onFail:
	// invalid tokens via AuthorizationFailureHandler, missing ones below.
	verifySession := clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizedPartyMatches(webOrigin),
		clerkhttp.AuthorizationFailureHandler(onFail),
	)

	return func(next http.Handler) http.Handler {
		withID := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := clerk.SessionClaimsFromContext(r.Context())
			if !ok || claims.Subject == "" {
				onFail.ServeHTTP(w, r)
				return
			}
			// claims.Subject is the Clerk user id == users._id in MongoDB.
			next.ServeHTTP(w, r.WithContext(WithCandidateID(r.Context(), claims.Subject)))
		})
		return verifySession(withID)
	}
}
