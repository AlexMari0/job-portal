import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { logoutCompanyAction } from "@/app/company/actions";
import { getCurrentCompany, SESSION_EXPIRED_PATH } from "@/lib/company";
import DashboardNav from "./DashboardNav";

export const metadata: Metadata = { title: "Dashboard | Job Portal" };

// Recruiter panel shell. proxy.ts already sent visitors without a Company
// cookie away; CompanyMenu checks the token with the Go API.
export default function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  return (
    <div className="min-h-screen">
      <div className="shadow-sm py-4">
        <div className="px-5 flex justify-between items-center">
          <Link href="/">
            <Image className="max-sm:w-32" src="/assets/logo.svg" alt="Job Portal" width={160} height={40} priority />
          </Link>
          <Suspense fallback={<div className="h-8" />}>
            <CompanyMenu />
          </Suspense>
        </div>
      </div>

      <div className="flex items-start">
        <div className="inline-block min-h-screen border-r-2 border-gray-200">
          <DashboardNav />
        </div>
        <div className="flex-1 h-full p-2 sm:p-5">{children}</div>
      </div>
    </div>
  );
}

async function CompanyMenu() {
  const company = await getCurrentCompany();
  if (!company) redirect(SESSION_EXPIRED_PATH); // token expired or Company gone

  return (
    <div className="flex items-center gap-3">
      <p className="max-sm:hidden">Welcome, {company.name}!</p>
      {/* Hover (or keyboard focus) shows the Logout menu. */}
      <div className="relative group">
        <button type="button" aria-label="Account menu">
          <Image className="w-8 h-8 border border-gray-200 rounded-full object-cover" src={company.image} alt="" width={32} height={32} />
        </button>
        <div className="absolute hidden group-hover:block group-focus-within:block top-0 right-0 z-10 text-black rounded-sm pt-12">
          <form action={logoutCompanyAction} className="m-0 p-2 bg-white rounded-md border border-gray-200 text-sm">
            <button type="submit" className="py-1 px-2 pr-10">
              Logout
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
