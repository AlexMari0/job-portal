import "server-only";

// Base URL of the Go API. Read on the server only, so it never ships to the browser.
const API_URL = process.env.API_URL ?? "http://localhost:8080";

export type Health = {
  status: "ok" | "degraded";
  database: "ok" | "unreachable";
};

/**
 * Fetches the Go API health check. Returns null when the API itself is down,
 * so the page can show that instead of crashing.
 */
export async function getHealth(): Promise<Health | null> {
  try {
    const res = await fetch(`${API_URL}/health`);
    return (await res.json()) as Health;
  } catch {
    return null;
  }
}
