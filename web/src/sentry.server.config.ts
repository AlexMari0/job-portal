import * as Sentry from "@sentry/nextjs";

// Server-side errors: Server Components, Server Actions, route handlers.
// No DSN (the local default) leaves Sentry disabled.
Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ?? "development",
  tracesSampleRate: 0,
  debug: process.env.SENTRY_DEBUG === "1", // SDK logs, for troubleshooting // errors only, as the Express app had tracing off
});
