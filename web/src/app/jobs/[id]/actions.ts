"use server";

import { refresh } from "next/cache";
import { ApiError, applyToJob } from "@/lib/api";

export type ApplyResult = { ok: true } | { ok: false; code: string; message: string };

// Server Action: runs on the Next.js server, which calls the Go API with the
// Candidate's Clerk token (the browser never talks to Go, see ADR 0001).
export async function applyAction(jobId: string): Promise<ApplyResult> {
  try {
    await applyToJob(jobId);
  } catch (err) {
    if (err instanceof ApiError) return { ok: false, code: err.code, message: err.message };
    throw err;
  }
  refresh(); // re-render the page so it shows "Already Applied"
  return { ok: true };
}
