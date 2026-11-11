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
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
