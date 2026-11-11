import { redirect } from "next/navigation";

// /dashboard opens on Manage Jobs, as before.
export default function DashboardHome() {
  redirect("/dashboard/manage-jobs");
}
