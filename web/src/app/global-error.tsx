"use client";

import * as Sentry from "@sentry/nextjs";
import { useEffect } from "react";

// Last-resort error page for errors in the root layout; reports them to
// Sentry. It replaces the root layout, so it renders its own <html>.
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    Sentry.captureException(error);
  }, [error]);

  return (
    <html lang="en">
      <body style={{ fontFamily: "sans-serif", padding: "4rem", textAlign: "center" }}>
        <h1>Something went wrong</h1>
        <p>The error has been reported.</p>
        <button type="button" onClick={reset} style={{ marginTop: "1rem", textDecoration: "underline" }}>
          Try again
        </button>
      </body>
    </html>
  );
}
