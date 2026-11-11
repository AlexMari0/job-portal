import "server-only";

import type { components } from "./api-types";

// Types generated from api/openapi.yaml (`npm run gen:api`).
export type Health = components["schemas"]["Health"];
export type Job = components["schemas"]["Job"];
export type ApiErrorBody = components["schemas"]["Error"];

// Base URL of the Go API. Read on the server only (see ADR 0001: the
// browser never calls the Go API), so it never ships to the browser.
const API_URL = process.env.API_URL ?? "http://localhost:8080";

/** A non-2xx response from the Go API, carrying the contract's error code. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, init);
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as ApiErrorBody | null;
    throw new ApiError(
      res.status,
      body?.error.code ?? "unknown_error",
      body?.error.message ?? res.statusText,
    );
  }
  return (await res.json()) as T;
}

/** Go API health. Returns null when the API itself is unreachable. */
export async function getHealth(): Promise<Health | null> {
  try {
    const res = await fetch(`${API_URL}/health`);
    return (await res.json()) as Health;
  } catch {
    return null;
  }
}

/** All visible Jobs, newest first. */
export function getJobs(): Promise<Job[]> {
  return apiFetch<Job[]>("/jobs");
}

/** One visible Job, or null when it does not exist or is a Hidden Job. */
export async function getJob(id: string): Promise<Job | null> {
  try {
    return await apiFetch<Job>(`/jobs/${encodeURIComponent(id)}`);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}
