// Package handler contains the HTTP handlers (the Go counterpart of
// Express controllers).
package handler

import (
	"context"
	"net/http"
	"time"
)

// Pinger is anything that can check whether a dependency is alive.
// *mongo.Client satisfies it via a small adapter in main, and tests use a fake.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health reports whether the API and its database are reachable.
func Health(db Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "degraded",
				"database": "unreachable",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "ok",
		})
	})
}
