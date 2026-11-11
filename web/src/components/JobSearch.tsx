"use client";

import { createContext, useContext, useState, type ReactNode } from "react";

// The Hero search box and the Job list are separate parts of the page,
// so the search terms live in a small client-side context they share.
export type SearchFilter = { title: string; location: string };

type JobSearchValue = {
  filter: SearchFilter;
  setFilter: (f: SearchFilter) => void;
};

const JobSearchContext = createContext<JobSearchValue | null>(null);

export function JobSearchProvider({ children }: { children: ReactNode }) {
  const [filter, setFilter] = useState<SearchFilter>({ title: "", location: "" });
  return <JobSearchContext value={{ filter, setFilter }}>{children}</JobSearchContext>;
}

export function useJobSearch(): JobSearchValue {
  const value = useContext(JobSearchContext);
  if (!value) throw new Error("useJobSearch must be used inside <JobSearchProvider>");
  return value;
}
