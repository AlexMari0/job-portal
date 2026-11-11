// Command api is the job portal HTTP API (the Go counterpart of server.js).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/user"

	"github.com/AlexMari0/job-portal/api/internal/auth"
	"github.com/AlexMari0/job-portal/api/internal/config"
	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
	"github.com/AlexMari0/job-portal/api/internal/upload"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	// ctx is cancelled on Ctrl+C or SIGTERM (what Fly.io/Docker send on deploy).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	db, err := store.Connect(connectCtx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer db.Close(context.Background())
	slog.Info("database connected", "db", cfg.MongoDB)
	if cfg.ClerkWebhookSigningSecret == "" {
		slog.Warn("CLERK_WEBHOOK_SIGNING_SECRET not set: POST /webhooks/clerk returns 503")
	}

	var candidateAuth handler.Middleware
	var fetchClerkUser handler.FetchClerkUser
	if cfg.ClerkSecretKey != "" {
		clerk.SetKey(cfg.ClerkSecretKey)
		candidateAuth = auth.ClerkCandidate(cfg.WebOrigin, handler.Unauthenticated)
		fetchClerkUser = fetchFromClerk
	} else {
		slog.Warn("CLERK_SECRET_KEY not set: Candidate routes return 503")
	}

	var uploader handler.Uploader
	if cfg.CloudinaryCloudName != "" {
		cld, err := upload.NewCloudinary(cfg.CloudinaryCloudName, cfg.CloudinaryAPIKey, cfg.CloudinaryAPISecret)
		if err != nil {
			return err
		}
		uploader = cld
	} else {
		slog.Warn("CLOUDINARY_CLOUD_NAME not set: upload routes return 503")
	}

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: handler.NewRouter(handler.Deps{
			Store:              db,
			ClerkWebhookSecret: cfg.ClerkWebhookSigningSecret,
			CandidateAuth:      candidateAuth,
			FetchClerkUser:     fetchClerkUser,
			Uploader:           uploader,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Run the server in a goroutine so main can wait for a shutdown signal.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	// Give in-flight requests up to 10s to finish.
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}

// fetchFromClerk loads a user from the Clerk Backend API. clerk.User uses
// the same JSON field names as webhook payloads, so a JSON round trip maps
// it onto handler.ClerkUser and reuses the webhook's name/email rules.
func fetchFromClerk(ctx context.Context, id string) (handler.ClerkUser, error) {
	u, err := user.Get(ctx, id)
	var apiErr *clerk.APIErrorResponse
	if errors.As(err, &apiErr) && apiErr.HTTPStatusCode == http.StatusNotFound {
		return handler.ClerkUser{}, store.ErrNotFound
	}
	if err != nil {
		return handler.ClerkUser{}, fmt.Errorf("fetch clerk user %s: %w", id, err)
	}
	raw, err := json.Marshal(u)
	if err != nil {
		return handler.ClerkUser{}, err
	}
	var cu handler.ClerkUser
	err = json.Unmarshal(raw, &cu)
	return cu, err
}
