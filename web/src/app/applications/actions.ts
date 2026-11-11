"use server";

import { refresh } from "next/cache";
import { ApiError, uploadResume } from "@/lib/api";

export type UploadResult = { ok: true } | { ok: false; message: string };

// Server Action: receives the file from the browser and forwards it to the
// Go API, which validates it and stores it in Cloudinary.
export async function uploadResumeAction(formData: FormData): Promise<UploadResult> {
  const file = formData.get("resume");
  if (!(file instanceof File) || file.size === 0) return { ok: false, message: "Select a PDF resume" };

  try {
    await uploadResume(file);
  } catch (err) {
    if (err instanceof ApiError) return { ok: false, message: err.message };
    throw err;
  }
  refresh();
  return { ok: true };
}
