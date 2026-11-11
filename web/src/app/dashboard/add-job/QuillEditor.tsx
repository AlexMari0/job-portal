"use client";

import "quill/dist/quill.snow.css";
import type Quill from "quill";
import { useEffect, useRef } from "react";

// Quill manipulates the DOM directly, so it is created in the browser only
// (inside useEffect) and loaded on demand with import().
export default function QuillEditor({ editorRef }: { editorRef: React.RefObject<Quill | null> }) {
  const container = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = container.current;
    let cancelled = false;
    let quill: Quill | null = null;
    import("quill").then(({ default: QuillClass }) => {
      if (cancelled || !el) return;
      quill = new QuillClass(el, { theme: "snow" });
      editorRef.current = quill;
    });
    return () => {
      // React may mount twice in development: remove the toolbar Quill added.
      cancelled = true;
      editorRef.current = null;
      if (quill) el?.previousElementSibling?.remove();
    };
  }, [editorRef]);

  return <div ref={container} className="min-h-40" />;
}
