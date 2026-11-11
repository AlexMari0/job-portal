import Link from "next/link";
import { Suspense } from "react";
import { listCompanyJobs, withCompanySession } from "@/lib/company";
import { formatDate } from "@/lib/jobs";
import VisibilityToggle from "./VisibilityToggle";

export default function ManageJobsPage() {
  return (
    <Suspense fallback={<p className="text-gray-500 p-4">Loading…</p>}>
      <ManageJobs />
    </Suspense>
  );
}

const cell = "py-2 px-4 border-b border-gray-200";

async function ManageJobs() {
  const jobs = await withCompanySession(listCompanyJobs);

  if (jobs.length === 0) {
    return (
      <div className="flex flex-col gap-6 items-center justify-center h-[70vh]">
        <p className="text-xl sm:text-2xl">No Jobs posted yet</p>
        <AddJobLink />
      </div>
    );
  }

  return (
    <div className="container p-4 max-w-5xl">
      <div className="overflow-x-auto">
        <table className="min-w-full bg-white border border-gray-200 max-sm:text-sm">
          <thead>
            <tr>
              <th className={`${cell} text-left max-sm:hidden`}>#</th>
              <th className={`${cell} text-left`}>Job Title</th>
              <th className={`${cell} text-left max-sm:hidden`}>Date</th>
              <th className={`${cell} text-left max-sm:hidden`}>Location</th>
              <th className={`${cell} text-center`}>Applicants</th>
              <th className={`${cell} text-left`}>Visible</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job, i) => (
              <tr key={job.id} className="text-gray-700">
                <td className={`${cell} max-sm:hidden`}>{i + 1}</td>
                <td className={cell}>{job.title}</td>
                <td className={`${cell} max-sm:hidden`}>{formatDate(job.postedAt)}</td>
                <td className={`${cell} max-sm:hidden`}>{job.location}</td>
                <td className={`${cell} text-center`}>{job.applicationCount}</td>
                <td className={cell}>
                  <VisibilityToggle jobId={job.id} visible={job.visible} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="mt-4 flex justify-end">
        <AddJobLink />
      </div>
    </div>
  );
}

function AddJobLink() {
  return (
    <Link href="/dashboard/add-job" className="bg-black text-white py-2 px-4 rounded-sm">
      Add new job
    </Link>
  );
}
