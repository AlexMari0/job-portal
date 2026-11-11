"use client";

import { useTransition } from "react";
import { toast } from "sonner";
import type { ApplicationStatus } from "@/lib/company";
import { setApplicationStatusAction } from "../actions";

// Accept / Reject menu for a Pending Application (shown on hover or focus).
export default function StatusMenu({ applicationId }: { applicationId: string }) {
  const [pending, startTransition] = useTransition();

  function set(status: ApplicationStatus) {
    startTransition(async () => {
      const result = await setApplicationStatusAction(applicationId, status);
      if (!result.ok) toast.error(result.message);
      else toast.success("Status Changed");
    });
  }

  if (pending) return <span className="text-gray-400">Saving…</span>;

  return (
    <div className="relative inline-block text-left group">
      <button type="button" className="text-gray-500" aria-label="Change status">
        ...
      </button>
      <div className="z-10 hidden group-hover:block group-focus-within:block absolute right-0 md:left-0 top-0 mt-2 w-32 bg-white border border-gray-200 rounded-sm shadow-sm">
        <button type="button" onClick={() => set("Accepted")} className="block w-full text-left px-4 py-2 text-blue-500 hover:bg-gray-100">
          Accept
        </button>
        <button type="button" onClick={() => set("Rejected")} className="block w-full text-left px-4 py-2 text-red-500 hover:bg-gray-100">
          Reject
        </button>
      </div>
    </div>
  );
}
