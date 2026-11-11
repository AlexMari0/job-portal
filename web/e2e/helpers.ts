import { execFileSync } from "node:child_process";
import path from "node:path";
import { expect, type Page } from "@playwright/test";

export const API = "http://localhost:8090";

/** Resets the job-portal-e2e database to the sample data of `make seed`. */
export function reseed() {
  execFileSync("go", ["run", "./cmd/seed"], {
    cwd: path.join(__dirname, "..", "..", "api"),
    env: { ...process.env, MONGODB_DB: "job-portal-e2e" },
    stdio: "pipe",
  });
}

// Accounts created by the seed (api/cmd/seed).
export const ACME = { email: "hr@acme.test", password: "password123", name: "Acme Corp" };
export const GLOBEX = { email: "jobs@globex.test", password: "password123", name: "Globex" };

/** A Company session token straight from the Go API (for test setup only). */
export async function companyToken(email: string, password: string): Promise<string> {
  const res = await fetch(`${API}/companies/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  return ((await res.json()) as { token: string }).token;
}

/** Signs a Company in through the Recruiter Login modal, as a user would. */
export async function loginCompany(page: Page, email: string, password: string) {
  await page.goto("/");
  await page.getByRole("button", { name: "Recruiter Login" }).click();
  const form = page.getByRole("form", { name: "Recruiter Login" });
  await form.getByPlaceholder("Email Id").fill(email);
  await form.getByPlaceholder("Password").fill(password);
  await form.getByRole("button", { name: "login" }).click();
}

/** Job titles on the visible Job cards. */
export function jobCardTitles(page: Page) {
  return page.locator("#job-list ~ div h4");
}

export async function expectToast(page: Page, text: string) {
  await expect(page.locator("[data-sonner-toast]").filter({ hasText: text })).toBeVisible();
}
