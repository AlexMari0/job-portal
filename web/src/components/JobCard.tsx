import Image from "next/image";
import Link from "next/link";
import type { Job } from "@/lib/api";

export default function JobCard({ job }: { job: Job }) {
  const href = `/jobs/${job.id}`;
  return (
    <div className="border border-gray-200 p-6 shadow-sm rounded-sm">
      <Image className="h-8 w-auto" src={job.company.image} alt={job.company.name} width={64} height={32} />
      <h4 className="font-medium text-xl mt-2">{job.title}</h4>
      <div className="flex items-center gap-3 mt-2 text-xs">
        <span className="bg-blue-50 border border-blue-200 px-4 py-1.5 rounded-sm">{job.location}</span>
        <span className="bg-red-50 border border-red-200 px-4 py-1.5 rounded-sm">{job.level}</span>
      </div>
      {/* Plain-text excerpt from the API: no raw HTML on cards. */}
      <p className="text-gray-500 text-sm mt-4">{job.excerpt}</p>
      <div className="mt-4 flex gap-4 text-sm">
        <Link href={href} className="bg-blue-600 text-white px-4 py-2 rounded-sm">
          Apply now
        </Link>
        <Link href={href} className="text-gray-500 border border-gray-500 rounded-sm px-4 py-2">
          Learn more
        </Link>
      </div>
    </div>
  );
}
