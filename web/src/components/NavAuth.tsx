"use client";

import { Show, SignInButton, UserButton, useUser } from "@clerk/nextjs";
import Link from "next/link";

// The signed-in / signed-out part of the Navbar. Clerk components read the
// session in the browser, so this is a Client Component.
export default function NavAuth() {
  const { user } = useUser();
  const name = [user?.firstName, user?.lastName].filter(Boolean).join(" ");

  return (
    <>
      <Show when="signed-in">
        <div className="flex items-center gap-3">
          <Link href="/applications">Applied Jobs</Link>
          <p>|</p>
          {name && <p className="max-sm:hidden">Hi, {name}</p>}
          <UserButton />
        </div>
      </Show>
      <Show when="signed-out">
        <div className="flex gap-4 max-sm:text-xs">
          {/* Wired to the Recruiter Login modal in the Company login slice. */}
          <button type="button" className="text-gray-600">
            Recruiter Login
          </button>
          <SignInButton mode="modal">
            <button type="button" className="bg-blue-600 text-white px-6 sm:px-9 py-2 rounded-full">
              Login
            </button>
          </SignInButton>
        </div>
      </Show>
    </>
  );
}
