// Package config reads settings from environment variables
// (the Go counterpart of `dotenv` + `process.env`).
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	MongoURI string
	MongoDB  string

	// ClerkWebhookSigningSecret verifies Clerk webhooks (Clerk dashboard →
	// Webhooks → endpoint → Signing Secret, "whsec_..."). Optional locally.
	ClerkWebhookSigningSecret string

	// ClerkSecretKey verifies Candidate session tokens ("sk_test_..." / "sk_live_...").
	ClerkSecretKey string
	// WebOrigin is the Next.js site's origin; Clerk tokens must be issued for it (azp).
	WebOrigin string

	// Cloudinary account for uploads; all three empty disables uploads.
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string

	// CompanyJWTSecret signs Company session tokens; empty disables Company sign-in.
	CompanyJWTSecret string

	// SentryDSN enables error reporting; empty (the local default) disables it.
	SentryDSN         string
	SentryEnvironment string
}

// Load reads api/.env if present (local development), then the process
// environment. Real environment variables win over the file.
func Load() Config {
	_ = godotenv.Load() // missing .env is fine: production sets real env vars

	return Config{
		Port:     getenv("PORT", "8080"),
		MongoURI: getenv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:  getenv("MONGODB_DB", "job-portal"),

		ClerkWebhookSigningSecret: os.Getenv("CLERK_WEBHOOK_SIGNING_SECRET"),
		ClerkSecretKey:            os.Getenv("CLERK_SECRET_KEY"),
		WebOrigin:                 getenv("WEB_ORIGIN", "http://localhost:3000"),

		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),

		CompanyJWTSecret: os.Getenv("COMPANY_JWT_SECRET"),

		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: getenv("SENTRY_ENVIRONMENT", "development"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
