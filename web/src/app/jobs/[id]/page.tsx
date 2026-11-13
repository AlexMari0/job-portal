import { auth } from "@clerk/nextjs/server";
import type { Metadata } from "next";
import Image from "next/image";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import Footer from "@/components/Footer";
import JobCard from "@/components/JobCard";
import Navbar from "@/components/Navbar";
import { getJob, getJobs, getMe, getMyApplications, optional, type Job } from "@/lib/api";
import { timeAgo } from "@/lib/jobs";
import ApplyButton from "./ApplyButton";

export async function generateMetadata({ params }: PageProps<"/jobs/[id]">): Promise<Metadata> {
  const job = await getJob((await params).id);
  if (!job) return { title: "Job not found | Job Portal" };
  return { title: `${job.title} at ${job.company.name} | Job Portal`, description: job.excerpt };
}

export default function JobPage({ params }: PageProps<"/jobs/[id]">) {
  return (
    <>
      <Navbar />
      {/* params are only known at request time, so read them inside Suspense. */}
      <Suspense fallback={<Spinner />}>
        <JobDetail params={params} />
      </Suspense>
      <Footer />
    </>
  );
}

async function JobDetail({ params }: { params: PageProps<"/jobs/[id]">["params"] }) {
  const { id } = await params;
  // Independent requests run in parallel.
  // The Job is public: if the Candidate endpoints fail, show it as for a
  // signed-out visitor rather than an error page.
  const [job, session, me, applications] = await Promise.all([
    getJob(id),
    auth(),
    optional(getMe(), null),
    optional(getMyApplications(), []),
  ]);
  if (!job) notFound(); // unknown or Hidden Job → 404 page

  const appliedJobIds = new Set(applications.map((a) => a.job.id));
  const apply = (
    <ApplyButton jobId={job.id} signedIn={session.userId !== null} hasResume={!!me?.resume} alreadyApplied={appliedJobIds.has(job.id)} />
  );

  return (
    <div className="min-h-screen flex flex-col py-10 container px-4 2xl:px-20 mx-auto">
      <div className="bg-white text-black rounded-lg w-full">
        <div className="flex justify-center md:justify-between flex-wrap gap-8 px-14 py-20 mb-6 bg-sky-50 border border-sky-400 rounded-xl">
          <div className="flex flex-col md:flex-row items-center">
            <Image
              className="h-24 w-auto bg-white rounded-lg p-4 mr-4 max-md:mb-4 border border-gray-200"
              src={job.company.image}
              alt={job.company.name}
              width={96}
              height={96}
            />
            <div className="text-center md:text-left text-neutral-700">
              <h1 className="text-2xl sm:text-4xl font-medium">{job.title}</h1>
              <div className="flex flex-row flex-wrap max-md:justify-center gap-y-2 gap-6 items-center text-gray-600 mt-2">
                <Fact icon="suitcase_icon.svg">{job.company.name}</Fact>
                <Fact icon="location_icon.svg">{job.location}</Fact>
                <Fact icon="person_icon.svg">{job.level}</Fact>
                <Fact icon="money_icon.svg">CTC: {job.salary.toLocaleString("en-US")}</Fact>
              </div>
            </div>
          </div>
          <div className="flex flex-col justify-center text-end text-sm max-md:mx-auto max-md:text-center">
            {apply}
            <p className="mt-1 text-gray-600">Posted: {timeAgo(job.postedAt)}</p>
          </div>
        </div>

        <div className="flex flex-col lg:flex-row justify-between items-start">
          <div className="w-full lg:w-2/3">
            <h2 className="font-bold text-2xl mb-4">Job Description</h2>
            {/* Safe: the Go API sanitizes description HTML with an allowlist. */}
            <div className="rich-text" dangerouslySetInnerHTML={{ __html: job.description }} />
            <div className="mt-10">{apply}</div>
          </div>
          <MoreJobs job={job} appliedJobIds={appliedJobIds} />
        </div>
      </div>
    </div>
  );
}

// Same Company's other Jobs the Candidate has not applied to, at most 4.
async function MoreJobs({ job, appliedJobIds }: { job: Job; appliedJobIds: Set<string> }) {
  const more = (await getJobs())
    .filter((j) => j.id !== job.id && j.company.id === job.company.id && !appliedJobIds.has(j.id))
    .slice(0, 4);
  return (
    <div className="w-full lg:w-1/3 mt-8 lg:mt-0 lg:ml-8 space-y-5">
      <h2>More jobs from {job.company.name}</h2>
      {more.map((j) => (
        <JobCard key={j.id} job={j} />
      ))}
    </div>
  );
}

function Fact({ icon, children }: { icon: string; children: React.ReactNode }) {
  return (
    <span className="flex items-center gap-1">
      <Image src={`/assets/${icon}`} alt="" width={20} height={20} />
      {children}
    </span>
  );
}

function Spinner() {
  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="w-20 h-20 border-4 border-gray-300 border-t-blue-400 rounded-full animate-spin" />
    </div>
  );
}
