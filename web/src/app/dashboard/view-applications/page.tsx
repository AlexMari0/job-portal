import Image from "next/image";
import { Suspense } from "react";
import { listCompanyApplications, withCompanySession } from "@/lib/company";
import StatusMenu from "./StatusMenu";

export default function ViewApplicationsPage() {
  return (
    <Suspense fallback={<p className="text-gray-500 p-4">Loading…</p>}>
      <ViewApplications />
    </Suspense>
  );
}

const cell = "py-2 px-4 border-b border-gray-200";

async function ViewApplications() {
  // Rows whose Candidate or Job was deleted are already left out by the API.
  const applications = await withCompanySession(listCompanyApplications);

  if (applications.length === 0) {
    return (
      <div className="flex items-center justify-center h-[70vh]">
        <p className="text-xl sm:text-2xl">No Applications Available</p>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-4">
      <table className="w-full max-w-4xl bg-white border border-gray-200 max-sm:text-sm">
        <thead>
          <tr className="border-b border-gray-200">
            <th className="py-2 px-4 text-left">#</th>
            <th className="py-2 px-4 text-left">Username</th>
            <th className="py-2 px-4 text-left max-sm:hidden">Job Title</th>
            <th className="py-2 px-4 text-left max-sm:hidden">Location</th>
            <th className="py-2 px-4 text-left">Resume</th>
            <th className="py-2 px-4 text-left">Action</th>
          </tr>
        </thead>
        <tbody>
          {applications.map((a, i) => (
            <tr key={a.id} className="text-gray-700">
              <td className={`${cell} text-center`}>{i + 1}</td>
              <td className={cell}>
                <div className="flex items-center">
                  <Image className="w-10 h-10 rounded-full mr-3 max-sm:hidden object-cover" src={a.candidate.image} alt="" width={40} height={40} />
                  <span>{a.candidate.name}</span>
                </div>
              </td>
              <td className={`${cell} max-sm:hidden`}>{a.job.title}</td>
              <td className={`${cell} max-sm:hidden`}>{a.job.location}</td>
              <td className={cell}>
                {a.candidate.resume ? (
                  <a
                    href={a.candidate.resume}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="bg-blue-50 text-blue-400 px-3 py-1 rounded-sm inline-flex gap-2 items-center"
                  >
                    Resume <Image src="/assets/resume_download_icon.svg" alt="" width={16} height={16} />
                  </a>
                ) : (
                  <span className="text-gray-400">No resume</span>
                )}
              </td>
              <td className={`${cell} relative`}>
                {a.status === "Pending" ? <StatusMenu applicationId={a.id} /> : <span>{a.status}</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
