"use client";

import Image from "next/image";
import { useJobSearch } from "./JobSearch";

export default function HeroSearch() {
  const { setFilter } = useJobSearch();

  // A real <form>: Enter submits, and FormData reads the inputs without refs.
  function onSearch(formData: FormData) {
    setFilter({
      title: String(formData.get("title") ?? "").trim(),
      location: String(formData.get("location") ?? "").trim(),
    });
    document.getElementById("job-list")?.scrollIntoView();
  }

  return (
    <form
      action={onSearch}
      className="flex items-center justify-between bg-white rounded-sm text-gray-600 max-w-xl pl-4 mx-4 sm:mx-auto"
    >
      <div className="flex items-center">
        <Image className="h-4 sm:h-5 w-auto" src="/assets/search_icon.svg" alt="" width={20} height={20} />
        <input name="title" type="text" placeholder="Search for jobs" className="max-sm:text-xs p-2 rounded-sm outline-hidden w-full" />
      </div>
      <div className="flex items-center">
        <Image className="h-4 sm:h-5 w-auto" src="/assets/location_icon.svg" alt="" width={20} height={20} />
        <input name="location" type="text" placeholder="Location" className="max-sm:text-xs p-2 rounded-sm outline-hidden w-full" />
      </div>
      <button type="submit" className="bg-blue-600 px-6 py-2 rounded-sm text-white m-1">
        Search
      </button>
    </form>
  );
}
