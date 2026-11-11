// Command api is the job portal HTTP API (the Go counterpart of server.js).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlexMari0/job-portal/api/internal/config"
	"github.com/AlexMari0/job-portal/api/internal/handler"
	"github.com/AlexMari0/job-portal/api/internal/store"
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

	srv := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: handler.NewRouter(handler.Deps{
			Store:              db,
			ClerkWebhookSecret: cfg.ClerkWebhookSigningSecret,
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
