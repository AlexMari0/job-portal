"use client";

import Image from "next/image";
import { useState, useTransition } from "react";
import { toast } from "sonner";
import { uploadResumeAction } from "./actions";

const MAX_BYTES = 2 * 1024 * 1024; // the Go API enforces the same limit

export default function ResumeForm({ resume }: { resume: string | null }) {
  const [editing, setEditing] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [pending, startTransition] = useTransition();

  if (resume && !editing) {
    return (
      <div className="flex gap-2">
        <a className="bg-blue-100 text-blue-600 px-4 py-2 rounded-lg" href={resume} target="_blank" rel="noopener noreferrer">
          Resume
        </a>
        <button type="button" onClick={() => setEditing(true)} className="text-gray-500 border border-gray-300 rounded-lg px-4 py-2">
          Edit
        </button>
      </div>
    );
  }

  function onSave() {
    if (!file) return void toast.error("Select a PDF resume");
    // Early, friendly checks; the API checks the actual content.
    if (file.type !== "application/pdf") return void toast.error("Resume must be a PDF");
    if (file.size > MAX_BYTES) return void toast.error("Resume must be at most 2 MB");

    const form = new FormData();
    form.append("resume", file);
    startTransition(async () => {
      const result = await uploadResumeAction(form);
      if (!result.ok) return void toast.error(result.message);
      toast.success("Resume Updated");
      setEditing(false);
      setFile(null);
    });
  }

  return (
    <>
      <label className="flex items-center cursor-pointer" htmlFor="resumeUpload">
        <p className="bg-blue-100 text-blue-600 px-4 py-2 rounded-lg mr-2">{file ? file.name : "Select Resume"}</p>
        <input
          id="resumeUpload"
          type="file"
          accept="application/pdf"
          hidden
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
        <Image src="/assets/profile_upload_icon.svg" alt="Choose file" width={40} height={40} />
      </label>
      <button
        type="button"
        onClick={onSave}
        disabled={pending}
        className="bg-green-100 border border-green-400 rounded-lg px-4 py-2 disabled:opacity-70"
      >
        {pending ? "Saving…" : "Save"}
      </button>
    </>
  );
}
