import * as Sentry from "@sentry/nextjs";

// Browser-side errors. The DSN is public by design (it only allows sending events).
Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
  environment: process.env.NEXT_PUBLIC_SENTRY_ENVIRONMENT ?? "development",
  tracesSampleRate: 0,
});
