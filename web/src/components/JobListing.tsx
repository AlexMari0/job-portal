"use client";

import Image from "next/image";
import { useState } from "react";
import type { Job } from "@/lib/api";
import { JOB_CATEGORIES, JOB_LOCATIONS, JOBS_PER_PAGE } from "@/lib/jobs";
import JobCard from "./JobCard";
import { useJobSearch } from "./JobSearch";

function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
}

// Filtering and pagination run in the browser over the full list from
// GET /jobs (already newest first), as in the original app.
export default function JobListing({ jobs }: { jobs: Job[] }) {
  const { filter, setFilter } = useJobSearch();
  const [showFilter, setShowFilter] = useState(false);
  const [categories, setCategories] = useState<string[]>([]);
  const [locations, setLocations] = useState<string[]>([]);
  const [page, setPage] = useState(1);

  // Derived during render instead of stored in state + synced with useEffect.
  const filtered = jobs.filter(
    (job) =>
      (categories.length === 0 || categories.includes(job.category)) &&
      (locations.length === 0 || locations.includes(job.location)) &&
      job.title.toLowerCase().includes(filter.title.toLowerCase()) &&
      job.location.toLowerCase().includes(filter.location.toLowerCase()),
  );
  const pageCount = Math.ceil(filtered.length / JOBS_PER_PAGE);
  const current = Math.min(page, Math.max(pageCount, 1));
  const visible = filtered.slice((current - 1) * JOBS_PER_PAGE, current * JOBS_PER_PAGE);

  // Any filter change goes back to page 1.
  const changeFilter = (fn: () => void) => {
    fn();
    setPage(1);
  };

  return (
    <div className="container 2xl:px-20 mx-auto flex flex-col lg:flex-row max-lg:space-y-8 py-8">
      {/* Sidebar */}
      <div className="w-full lg:w-1/4 bg-white px-4">
        {(filter.title || filter.location) && (
          <>
            <h3 className="font-medium text-lg mb-4">Current Search</h3>
            <div className="mb-4 text-gray-600 flex flex-wrap gap-2">
              {filter.title && (
                <SearchChip
                  label={filter.title}
                  className="bg-blue-50 border-blue-200"
                  onRemove={() => changeFilter(() => setFilter({ ...filter, title: "" }))}
                />
              )}
              {filter.location && (
                <SearchChip
                  label={filter.location}
                  className="bg-red-50 border-red-200"
                  onRemove={() => changeFilter(() => setFilter({ ...filter, location: "" }))}
                />
              )}
            </div>
          </>
        )}

        <button
          type="button"
          onClick={() => setShowFilter((v) => !v)}
          className="px-6 py-1.5 rounded-sm border border-gray-400 lg:hidden"
        >
          {showFilter ? "Close" : "Filters"}
        </button>

        <div className={showFilter ? "" : "max-lg:hidden"}>
          <h4 className="font-medium text-lg py-4">Search by Categories</h4>
          <CheckboxList
            options={JOB_CATEGORIES}
            selected={categories}
            onToggle={(c) => changeFilter(() => setCategories((prev) => toggle(prev, c)))}
          />
        </div>

        <div className={showFilter ? "" : "max-lg:hidden"}>
          <h4 className="font-medium text-lg py-4 pt-14">Search by Location</h4>
          <CheckboxList
            options={JOB_LOCATIONS}
            selected={locations}
            onToggle={(l) => changeFilter(() => setLocations((prev) => toggle(prev, l)))}
          />
        </div>
      </div>

      {/* Job list */}
      <section className="w-full lg:w-3/4 text-gray-800 max-lg:px-4">
        <h3 className="font-medium text-3xl py-2 scroll-mt-4" id="job-list">
          Latest Jobs
        </h3>
        <p className="mb-8">Get your desired job from top companies</p>
        <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
          {visible.map((job) => (
            <JobCard key={job.id} job={job} />
          ))}
        </div>

        {pageCount > 0 && (
          <nav className="flex items-center justify-center space-x-2 mt-10" aria-label="Pagination">
            <a href="#job-list" onClick={() => setPage(Math.max(current - 1, 1))} aria-label="Previous page">
              <Image src="/assets/left_arrow_icon.svg" alt="" width={24} height={24} />
            </a>
            {Array.from({ length: pageCount }, (_, i) => i + 1).map((n) => (
              <a
                key={n}
                href="#job-list"
                onClick={() => setPage(n)}
                aria-current={n === current ? "page" : undefined}
                className={`w-10 h-10 flex items-center justify-center border border-gray-300 rounded-sm ${
                  n === current ? "bg-blue-100 text-blue-500" : "text-gray-500"
                }`}
              >
                {n}
              </a>
            ))}
            <a href="#job-list" onClick={() => setPage(Math.min(current + 1, pageCount))} aria-label="Next page">
              <Image src="/assets/right_arrow_icon.svg" alt="" width={24} height={24} />
            </a>
          </nav>
        )}
      </section>
    </div>
  );
}

function SearchChip({ label, className, onRemove }: { label: string; className: string; onRemove: () => void }) {
  return (
    <span className={`inline-flex items-center gap-2.5 border px-4 py-1.5 rounded-sm max-w-full ${className}`}>
      <span className="truncate max-w-[150px] sm:max-w-xs">{label}</span>
      <button type="button" onClick={onRemove} className="shrink-0 cursor-pointer" aria-label={`Remove ${label}`}>
        <Image src="/assets/cross_icon.svg" alt="" width={12} height={12} />
      </button>
    </span>
  );
}

function CheckboxList({
  options,
  selected,
  onToggle,
}: {
  options: readonly string[];
  selected: string[];
  onToggle: (value: string) => void;
}) {
  return (
    <ul className="space-y-4 text-gray-600">
      {options.map((option) => (
        <li key={option}>
          <label className="flex gap-3 items-center cursor-pointer">
            <input
              className="scale-125"
              type="checkbox"
              checked={selected.includes(option)}
              onChange={() => onToggle(option)}
            />
            {option}
          </label>
        </li>
      ))}
    </ul>
  );
}
