import "server-only";

import { auth } from "@clerk/nextjs/server";
import type { components } from "./api-types";

// Types generated from api/openapi.yaml (`npm run gen:api`).
export type Health = components["schemas"]["Health"];
export type Job = components["schemas"]["Job"];
export type ApiErrorBody = components["schemas"]["Error"];
export type Candidate = components["schemas"]["Candidate"];
export type Application = components["schemas"]["Application"];

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
  if (res.status === 204) return undefined as T;
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

// ---- Candidate (Clerk session) ----

/**
 * Calls a Candidate route with the signed-in Candidate's Clerk session token.
 * The token lives ~60s, so it is fetched per request and never stored.
 * Returns null when nobody is signed in.
 */
async function asCandidate<T>(path: string, init: RequestInit = {}): Promise<T | null> {
  const { userId, getToken } = await auth();
  if (!userId) return null;
  const token = await getToken();
  if (!token) return null;
  return apiFetch<T>(path, {
    ...init,
    headers: { ...init.headers, Authorization: `Bearer ${token}` },
  });
}

/** The signed-in Candidate, or null (signed out, or not synced from Clerk yet). */
export async function getMe(): Promise<Candidate | null> {
  try {
    return await asCandidate<Candidate>("/me");
  } catch (err) {
    if (err instanceof ApiError && err.code === "candidate_not_found") return null;
    throw err;
  }
}

/** The signed-in Candidate's Applications, newest first ([] when signed out). */
export async function getMyApplications(): Promise<Application[]> {
  return (await asCandidate<Application[]>("/me/applications")) ?? [];
}

/** Applies to a Job. Throws ApiError (e.g. already_applied, resume_required). */
export async function applyToJob(jobId: string): Promise<Application> {
  const app = await asCandidate<Application>(`/jobs/${encodeURIComponent(jobId)}/applications`, {
    method: "POST",
  });
  if (!app) throw new ApiError(401, "unauthenticated", "Sign in required");
  return app;
}

/** Uploads or replaces the Candidate's resume (a PDF up to 2 MB). */
export async function uploadResume(file: File): Promise<{ resume: string }> {
  const form = new FormData();
  form.append("resume", file, file.name);
  const res = await asCandidate<{ resume: string }>("/me/resume", { method: "PUT", body: form });
  if (!res) throw new ApiError(401, "unauthenticated", "Sign in required");
  return res;
}
