"use client";

import { useOptimistic, useTransition } from "react";
import { toast } from "sonner";
import { setJobVisibilityAction } from "../actions";

// Shows the new state immediately (useOptimistic) while the Server Action
// sends the explicit value; on failure React rolls it back.
export default function VisibilityToggle({ jobId, visible }: { jobId: string; visible: boolean }) {
  const [shown, setShown] = useOptimistic(visible);
  const [, startTransition] = useTransition();

  function onChange(next: boolean) {
    startTransition(async () => {
      setShown(next);
      const result = await setJobVisibilityAction(jobId, next);
      if (!result.ok) toast.error(result.message);
      else toast.success("Visibility changed");
    });
  }

  return (
    <input
      className="scale-125 ml-4"
      type="checkbox"
      aria-label="Visible"
      checked={shown}
      onChange={(e) => onChange(e.target.checked)}
    />
  );
}
