"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { toast } from "sonner";
import { applyAction } from "./actions";

type Props = {
  jobId: string;
  signedIn: boolean;
  hasResume: boolean;
  alreadyApplied: boolean;
};

export default function ApplyButton({ jobId, signedIn, hasResume, alreadyApplied }: Props) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  const goUploadResume = () => {
    router.push("/applications");
    toast.error("Upload resume to apply");
  };

  function onApply() {
    // Quick checks for a friendly message; the Go API enforces them anyway.
    if (!signedIn) return void toast.error("Login to apply for jobs");
    if (!hasResume) return goUploadResume();

    startTransition(async () => {
      const result = await applyAction(jobId);
      if (result.ok) return void toast.success("Applied Successfully");
      if (result.code === "resume_required") return goUploadResume();
      toast.error(result.message);
    });
  }

  return (
    <button
      type="button"
      onClick={onApply}
      disabled={alreadyApplied || pending}
      className="bg-blue-600 p-2.5 px-10 text-white rounded-sm disabled:opacity-70 disabled:cursor-not-allowed"
    >
      {alreadyApplied ? "Already Applied" : pending ? "Applying…" : "Apply Now"}
    </button>
  );
}
