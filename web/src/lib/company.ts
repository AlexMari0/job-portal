import "server-only";

import { cookies } from "next/headers";
import { ApiError, apiFetch } from "./api";
import type { components } from "./api-types";

export type CompanyProfile = components["schemas"]["CompanyProfile"];
export type CompanySession = components["schemas"]["CompanySession"];

// httpOnly cookie on the Next.js domain holding the Company's JWT. Browser
// JavaScript cannot read it; only this server forwards it to the Go API.
export const COMPANY_COOKIE = "company_session";

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
