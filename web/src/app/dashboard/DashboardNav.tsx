"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";

const links = [
  { href: "/dashboard/manage-jobs", icon: "home_icon.svg", label: "Manage Jobs" },
  { href: "/dashboard/add-job", icon: "add_icon.svg", label: "Add Job" },
  { href: "/dashboard/view-applications", icon: "person_tick_icon.svg", label: "View Applications" },
];

// Sidebar; a Client Component only because it highlights the current page.
export default function DashboardNav() {
  const pathname = usePathname();
  return (
    <ul className="flex flex-col items-start pt-5 text-gray-800">
      {links.map(({ href, icon, label }) => {
        const active = pathname === href;
        return (
          <li key={href} className="w-full">
            <Link
              href={href}
              aria-current={active ? "page" : undefined}
              className={`flex items-center p-3 sm:px-6 gap-2 w-full hover:bg-gray-100 ${active ? "bg-blue-100 border-r-4 border-blue-500" : ""}`}
            >
              <Image className="min-w-4" src={`/assets/${icon}`} alt="" width={16} height={16} />
              <p className="max-sm:hidden">{label}</p>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
