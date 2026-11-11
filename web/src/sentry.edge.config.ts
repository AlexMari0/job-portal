import * as Sentry from "@sentry/nextjs";

// Same as the server config, for code running in the edge runtime.
Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ?? "development",
  tracesSampleRate: 0,
  debug: process.env.SENTRY_DEBUG === "1", // SDK logs, for troubleshooting
});
