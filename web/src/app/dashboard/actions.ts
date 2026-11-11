"use server";

import { refresh } from "next/cache";
import { ApiError } from "@/lib/api";
import {
  createJob,
  setApplicationStatus,
  setJobVisibility,
  withCompanySession,
  type ApplicationStatus,
  type NewJob,
} from "@/lib/company";

export type ActionResult = { ok: true } | { ok: false; message: string; fields?: Record<string, string> };

function failure(err: unknown): ActionResult {
  if (err instanceof ApiError && err.status < 500) return { ok: false, message: err.message, fields: err.fields };
  throw err;
}

export async function createJobAction(job: NewJob): Promise<ActionResult> {
  try {
    await withCompanySession(() => createJob(job));
  } catch (err) {
    return failure(err);
  }
  return { ok: true };
}

export async function setJobVisibilityAction(id: string, visible: boolean): Promise<ActionResult> {
  try {
    await withCompanySession(() => setJobVisibility(id, visible));
  } catch (err) {
    return failure(err);
  }
  refresh();
  return { ok: true };
}

export async function setApplicationStatusAction(id: string, status: ApplicationStatus): Promise<ActionResult> {
  try {
    await withCompanySession(() => setApplicationStatus(id, status));
  } catch (err) {
    return failure(err);
  }
  refresh();
  return { ok: true };
}
