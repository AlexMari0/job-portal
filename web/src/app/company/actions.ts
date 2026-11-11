"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { ApiError } from "@/lib/api";
import { COMPANY_COOKIE, loginCompany, type CompanySession } from "@/lib/company";

export type LoginState = { error: string | null };

async function startSession(session: CompanySession) {
  (await cookies()).set(COMPANY_COOKIE, session.token, {
    httpOnly: true, // not readable by browser JavaScript
    secure: process.env.NODE_ENV === "production", // HTTPS only in production
    sameSite: "lax",
    path: "/",
    expires: new Date(session.expiresAt), // same lifetime as the JWT
  });
}

// Used with useActionState: returns an error to show, or redirects.
export async function loginCompanyAction(_prev: LoginState, formData: FormData): Promise<LoginState> {
  const email = String(formData.get("email") ?? "");
  const password = String(formData.get("password") ?? "");
  try {
    await startSession(await loginCompany(email, password));
  } catch (err) {
    if (err instanceof ApiError && err.status < 500) return { error: err.message };
    throw err;
  }
  redirect("/dashboard"); // outside try: redirect() works by throwing
}

export async function logoutCompanyAction() {
  (await cookies()).delete(COMPANY_COOKIE);
  redirect("/");
}
