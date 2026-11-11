import type { components } from "./api-types";

type Category = components["schemas"]["Category"];
type Location = components["schemas"]["Location"];

// `satisfies` makes TypeScript check these lists against the API contract:
// a typo or a value the API does not accept fails the type check.
export const JOB_CATEGORIES = [
  "Programming",
  "Data Science",
  "Designing",
  "Networking",
  "Management",
  "Marketing",
  "Cybersecurity",
] as const satisfies readonly Category[];

export const JOB_LOCATIONS = [
  "Bangalore",
  "Washington",
  "Hyderabad",
  "Mumbai",
  "California",
  "Chennai",
  "New York",
] as const satisfies readonly Location[];

export const JOBS_PER_PAGE = 6;

const units: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 365 * 24 * 3600],
  ["month", 30 * 24 * 3600],
  ["week", 7 * 24 * 3600],
  ["day", 24 * 3600],
  ["hour", 3600],
  ["minute", 60],
];

/** "3 days ago" style text (replaces moment's fromNow()). */
export function timeAgo(iso: string, now = Date.now()): string {
  const seconds = Math.round((new Date(iso).getTime() - now) / 1000);
  const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  for (const [unit, size] of units) {
    if (Math.abs(seconds) >= size) return rtf.format(Math.round(seconds / size), unit);
  }
  return rtf.format(seconds, "second");
}
