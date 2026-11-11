import Image from "next/image";
import Link from "next/link";
import { Suspense } from "react";
import NavAuth from "./NavAuth";

export default function Navbar() {
  return (
    <div className="shadow-sm py-4">
      <div className="container px-4 2xl:px-20 mx-auto flex justify-between items-center">
        <Link href="/">
          <Image src="/assets/logo.svg" alt="Job Portal" width={160} height={40} priority />
        </Link>
        {/* Session-dependent UI must sit inside Suspense with Cache Components. */}
        <Suspense fallback={<div className="h-10" />}>
          <NavAuth />
        </Suspense>
      </div>
    </div>
  );
}
