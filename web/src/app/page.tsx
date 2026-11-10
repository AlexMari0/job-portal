import { Suspense } from "react";
import { getHealth } from "@/lib/api";

// Server Component: runs on the server, so it can call the Go API directly.
// The status must be fresh on every request, so it is not cached; it streams
// in behind <Suspense> while the static shell renders immediately.
async function ApiStatus() {
  const health = await getHealth();

  if (!health) {
    return <p className="text-red-600">API Go tidak dapat dihubungi.</p>;
  }

  const ok = health.status === "ok";
  return (
    <dl className="grid grid-cols-2 gap-x-6 gap-y-1">
      <dt className="text-zinc-500">API</dt>
      <dd className={ok ? "text-green-600" : "text-amber-600"}>{health.status}</dd>
      <dt className="text-zinc-500">Database</dt>
      <dd className={health.database === "ok" ? "text-green-600" : "text-red-600"}>
        {health.database}
      </dd>
    </dl>
  );
}

export default function Home() {
  return (
    <main className="mx-auto flex w-full max-w-xl flex-1 flex-col justify-center gap-6 p-8">
      <h1 className="text-2xl font-semibold">Job Portal</h1>
      <p className="text-zinc-600">Status stack baru (Next.js + Go + MongoDB):</p>
      <Suspense fallback={<p className="text-zinc-500">Memeriksa API…</p>}>
        <ApiStatus />
      </Suspense>
    </main>
  );
}
