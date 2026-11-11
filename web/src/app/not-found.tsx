import Link from "next/link";
import Navbar from "@/components/Navbar";

export default function NotFound() {
  return (
    <>
      <Navbar />
      <main className="container mx-auto px-4 py-32 text-center">
        <h1 className="text-3xl font-medium mb-4">Job not found</h1>
        <p className="text-gray-600 mb-8">This Job does not exist or is no longer open.</p>
        <Link href="/" className="bg-blue-600 text-white px-6 py-2 rounded-sm">
          Browse jobs
        </Link>
      </main>
    </>
  );
}
