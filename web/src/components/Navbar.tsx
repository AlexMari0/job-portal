import Image from "next/image";
import Link from "next/link";

// The login buttons are wired up in later slices (Candidate login via Clerk,
// Company login via the Recruiter Login modal).
export default function Navbar() {
  return (
    <div className="shadow-sm py-4">
      <div className="container px-4 2xl:px-20 mx-auto flex justify-between items-center">
        <Link href="/">
          <Image src="/assets/logo.svg" alt="Job Portal" width={160} height={40} priority />
        </Link>
        <div className="flex gap-4 max-sm:text-xs">
          <button type="button" className="text-gray-600">
            Recruiter Login
          </button>
          <button type="button" className="bg-blue-600 text-white px-6 sm:px-9 py-2 rounded-full">
            Login
          </button>
        </div>
      </div>
    </div>
  );
}
