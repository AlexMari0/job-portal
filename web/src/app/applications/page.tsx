import { auth } from "@clerk/nextjs/server";
import type { Metadata } from "next";
import Image from "next/image";
import { Suspense } from "react";
import Footer from "@/components/Footer";
import Navbar from "@/components/Navbar";
import { getMe, getMyApplications, type Application } from "@/lib/api";
import ResumeForm from "./ResumeForm";

export const metadata: Metadata = { title: "Applied Jobs | Job Portal" };

export default function ApplicationsPage() {
  return (
    <>
      <Navbar />
      <div className="container px-4 min-h-[65vh] 2xl:px-20 mx-auto my-10">
        <Suspense fallback={<p className="text-gray-500">Loading…</p>}>
          <Applications />
        </Suspense>
      </div>
      <Footer />
    </>
  );
}

async function Applications() {
  // Signed-out visitors are sent to Clerk's sign-in page and back here.
  await auth.protect();
  const [me, applications] = await Promise.all([getMe(), getMyApplications()]);

  return (
    <>
      <h2 className="text-xl font-semibold">Your Resume</h2>
      <div className="flex gap-2 mb-6 mt-3">
        <ResumeForm resume={me?.resume ?? null} />
      </div>

      <h2 className="text-xl font-semibold mb-4">Jobs Applied</h2>
      <table className="min-w-full bg-white border border-gray-200 rounded-lg">
        <thead>
          <tr>
            <th className="py-3 px-4 border-b border-gray-200 text-left">Company</th>
            <th className="py-3 px-4 border-b border-gray-200 text-left">Job Title</th>
            <th className="py-3 px-4 border-b border-gray-200 text-left max-sm:hidden">Location</th>
            <th className="py-3 px-4 border-b border-gray-200 text-left max-sm:hidden">Date</th>
            <th className="py-3 px-4 border-b border-gray-200 text-left">Status</th>
          </tr>
        </thead>
        <tbody>
          {applications.map((a) => (
            <ApplicationRow key={a.id} application={a} />
          ))}
        </tbody>
      </table>
      {applications.length === 0 && <p className="text-gray-500 mt-4">You have not applied to any Jobs yet.</p>}
    </>
  );
}

// "Nov 10, 2026", like moment's format("ll").
const dateFormat = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric" });

const statusColor: Record<Application["status"], string> = {
  Accepted: "bg-green-100",
  Rejected: "bg-red-100",
  Pending: "bg-blue-100",
};

function ApplicationRow({ application: a }: { application: Application }) {
  return (
    <tr>
      <td className="py-3 px-4 flex items-center gap-2 border-b border-gray-200">
        <Image className="w-8 h-8 object-contain" src={a.company.image} alt="" width={32} height={32} />
        {a.company.name}
      </td>
      <td className="py-2 px-4 border-b border-gray-200">{a.job.title}</td>
      <td className="py-2 px-4 border-b border-gray-200 max-sm:hidden">{a.job.location}</td>
      <td className="py-2 px-4 border-b border-gray-200 max-sm:hidden">{dateFormat.format(new Date(a.appliedAt))}</td>
      <td className="py-2 px-4 border-b border-gray-200">
        <span className={`${statusColor[a.status]} px-4 py-1.5 rounded-sm`}>{a.status}</span>
      </td>
    </tr>
  );
}
