import { Suspense } from "react";
import AppDownload from "@/components/AppDownload";
import Footer from "@/components/Footer";
import Hero from "@/components/Hero";
import JobListing from "@/components/JobListing";
import { JobSearchProvider } from "@/components/JobSearch";
import Navbar from "@/components/Navbar";
import { getJobs } from "@/lib/api";

// Server Component: fetches Jobs from the Go API on the server and hands
// them to the interactive (client) JobListing as plain props.
async function Jobs() {
  const jobs = await getJobs();
  return <JobListing jobs={jobs} />;
}

export default function Home() {
  return (
    <div>
      <Navbar />
      <JobSearchProvider>
        <Hero />
        {/* Not cached: always fresh, streamed in after the static shell. */}
        <Suspense fallback={<JobsFallback />}>
          <Jobs />
        </Suspense>
      </JobSearchProvider>
      <AppDownload />
      <Footer />
    </div>
  );
}

function JobsFallback() {
  return (
    <div className="container 2xl:px-20 mx-auto py-8 flex justify-center">
      <div className="w-12 h-12 border-4 border-gray-300 border-t-blue-400 rounded-full animate-spin" />
    </div>
  );
}
