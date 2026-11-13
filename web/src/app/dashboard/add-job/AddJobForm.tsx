"use client";

import type Quill from "quill";
import { useRef, useState, useTransition } from "react";
import { toast } from "sonner";
import { JOB_CATEGORIES, JOB_LEVELS, JOB_LOCATIONS } from "@/lib/jobs";
import { createJobAction } from "../actions";
import QuillEditor from "./QuillEditor";

const selectClass = "w-full px-3 py-2 border-2 border-gray-300 rounded-sm";

export default function AddJobForm() {
  const editor = useRef<Quill | null>(null);
  const form = useRef<HTMLFormElement>(null);
  const [pending, startTransition] = useTransition();
  const [errors, setErrors] = useState<Record<string, string>>({});

  // A plain onSubmit, not <form action>: React 19 resets the fields after a
  // form action returns, which would wipe them even when validation fails.
  function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const job = {
      title: String(formData.get("title") ?? ""),
      description: editor.current?.root.innerHTML ?? "",
      location: String(formData.get("location")) as (typeof JOB_LOCATIONS)[number],
      category: String(formData.get("category")) as (typeof JOB_CATEGORIES)[number],
      level: String(formData.get("level")) as (typeof JOB_LEVELS)[number],
      salary: Number(formData.get("salary") || 0),
    };
    startTransition(async () => {
      const result = await createJobAction(job);
      if (!result.ok) {
        setErrors(result.fields ?? {});
        return void toast.error(result.message);
      }
      toast.success("Job Added");
      setErrors({});
      form.current?.reset();
      editor.current?.setText("");
    });
  }

  return (
    <form ref={form} onSubmit={onSubmit} className="container p-4 flex flex-col w-full items-start gap-3">
      <div className="w-full">
        <label className="block mb-2" htmlFor="title">Job Title</label>
        <input id="title" name="title" className="w-full max-w-lg px-3 py-2 border-2 border-gray-300 rounded-sm" type="text" placeholder="Type here" required />
        <FieldError message={errors.title} />
      </div>
      <div className="w-full max-w-lg">
        <p className="my-2">Job Description</p>
        <QuillEditor editorRef={editor} />
        <FieldError message={errors.description} />
      </div>
      <div className="flex flex-col sm:flex-row gap-2 w-full sm:gap-8">
        <Select label="Job Category" name="category" options={JOB_CATEGORIES} />
        <Select label="Job Location" name="location" options={JOB_LOCATIONS} />
        {/* Default is the first option, "Beginner level", which the API accepts. */}
        <Select label="Job Level" name="level" options={JOB_LEVELS} />
      </div>
      <div>
        <label className="block mb-2" htmlFor="salary">Job Salary</label>
        <input id="salary" name="salary" min={0} step={1} className="w-full px-3 py-2 border-2 border-gray-300 rounded-sm sm:w-[120px]" type="number" placeholder="2500" required />
        <FieldError message={errors.salary} />
      </div>
      <button type="submit" disabled={pending} className="w-28 py-3 mt-4 bg-black text-white rounded-sm disabled:opacity-70">
        {pending ? "ADDING…" : "ADD"}
      </button>
    </form>
  );
}

function Select({ label, name, options }: { label: string; name: string; options: readonly string[] }) {
  return (
    <div>
      <label className="block mb-2" htmlFor={name}>{label}</label>
      <select id={name} name={name} className={selectClass}>
        {options.map((o) => (
          <option key={o} value={o}>{o}</option>
        ))}
      </select>
    </div>
  );
}

function FieldError({ message }: { message?: string }) {
  return message ? <p className="text-sm text-red-600 mt-1">{message}</p> : null;
}
