import "server-only";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ApiError, apiFetch } from "./api";
import { COMPANY_COOKIE } from "./company-cookie";
import type { components } from "./api-types";

export type CompanyProfile = components["schemas"]["CompanyProfile"];
export type CompanySession = components["schemas"]["CompanySession"];
export type CompanyJob = components["schemas"]["CompanyJob"];
export type NewJob = components["schemas"]["NewJob"];
export type CompanyApplication = components["schemas"]["CompanyApplication"];
export type ApplicationStatus = components["schemas"]["ApplicationStatus"];

// httpOnly cookie on the Next.js domain holding the Company's JWT. Browser
// JavaScript cannot read it; only this server forwards it to the Go API.
export { COMPANY_COOKIE };

// Where to send a Company whose token the API rejected: a route handler that
// clears the cookie (Server Components cannot modify cookies) and goes home.
export const SESSION_EXPIRED_PATH = "/company/session-expired";

/** Exchanges email + password for a session token (does not set the cookie). */
export function loginCompany(email: string, password: string): Promise<CompanySession> {
  return apiFetch<CompanySession>("/companies/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
}

/** Registers a Company (multipart with the logo); returns a session token. */
export function registerCompany(form: FormData): Promise<CompanySession> {
  return apiFetch<CompanySession>("/companies", { method: "POST", body: form });
}

/** Whether a Company session cookie exists (it may still be expired). */
export async function hasCompanySession(): Promise<boolean> {
  return (await cookies()).has(COMPANY_COOKIE);
}

/**
 * Calls a /company route with the Company's token. Throws ApiError with
 * status 401 when there is no session or the API rejects it.
 */
export async function asCompany<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = (await cookies()).get(COMPANY_COOKIE)?.value;
  if (!token) throw new ApiError(401, "unauthenticated", "Sign in required");
  return apiFetch<T>(path, { ...init, headers: { ...init.headers, Authorization: `Bearer ${token}` } });
}

/** The signed-in Company, or null when there is no valid session. */
export async function getCurrentCompany(): Promise<CompanyProfile | null> {
  try {
    return await asCompany<CompanyProfile>("/company");
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null;
    throw err;
  }
}

/**
 * Runs a Company API call; when the API rejects the session (401), clears it
 * via the session-expired route instead of showing an error.
 */
export async function withCompanySession<T>(call: () => Promise<T>): Promise<T> {
  try {
    return await call();
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) redirect(SESSION_EXPIRED_PATH);
    throw err;
  }
}

export const listCompanyJobs = () => asCompany<CompanyJob[]>("/company/jobs");

export const createJob = (job: NewJob) =>
  asCompany<CompanyJob>("/company/jobs", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(job),
  });

export const setJobVisibility = (id: string, visible: boolean) =>
  asCompany<CompanyJob>(`/company/jobs/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ visible }),
  });

export const listCompanyApplications = () => asCompany<CompanyApplication[]>("/company/applications");

export const setApplicationStatus = (id: string, status: ApplicationStatus) =>
  asCompany<void>(`/company/applications/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ status }),
  });
