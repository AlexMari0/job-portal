import * as Sentry from "@sentry/nextjs";

// Next.js calls register() once when the server starts.
export async function register() {
  if (process.env.NEXT_RUNTIME === "nodejs") await import("./sentry.server.config");
  if (process.env.NEXT_RUNTIME === "edge") await import("./sentry.edge.config");
}

// Next.js calls this for every error thrown while handling a request
// (rendering, Server Actions, route handlers, proxy).
export const onRequestError = Sentry.captureRequestError;
