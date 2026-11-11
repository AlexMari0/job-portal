package handler

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

// Observe wraps the whole router for error reporting (the Go counterpart of
// Sentry.setupExpressErrorHandler):
//   - each request gets its own Sentry hub carrying the request details, so
//     writeInternalError can report 5xx errors with that context;
//   - a panic is reported and answered with the contract's 500 instead of
//     dropping the connection.
//
// Without a Sentry DSN every capture is a no-op.
func Observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(r)
		ctx := sentry.SetHubOnContext(r.Context(), hub)
		r = r.WithContext(ctx)

		defer func() {
			if v := recover(); v != nil {
				hub.RecoverWithContext(ctx, v)
				writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// reportError sends err to Sentry using the request's hub when there is one.
func reportError(r *http.Request, err error) {
	if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
		hub.CaptureException(err)
		return
	}
	sentry.CaptureException(err)
}
