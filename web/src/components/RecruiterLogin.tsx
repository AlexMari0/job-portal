"use client";

import Image from "next/image";
import { useActionState, useEffect } from "react";
import { toast } from "sonner";
import { loginCompanyAction, type LoginState } from "@/app/company/actions";

// Company sign-in modal. Sign Up (with logo upload) comes with the
// Company registration slice.
export default function RecruiterLogin({ onClose }: { onClose: () => void }) {
  const [state, formAction, pending] = useActionState<LoginState, FormData>(loginCompanyAction, { error: null });

  useEffect(() => {
    if (state.error) toast.error(state.error);
  }, [state]);

  // Lock page scrolling while the modal is open; Escape closes it.
  useEffect(() => {
    document.body.style.overflow = "hidden";
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-10 backdrop-blur-sm bg-black/30 flex justify-center items-center">
      <form action={formAction} className="relative bg-white p-10 rounded-xl text-slate-500" aria-label="Recruiter Login">
        <h1 className="text-center text-2xl text-neutral-700 font-medium">Recruiter Login</h1>
        <p className="text-sm">Welcome Back! Please sign in to continue.</p>

        <Field icon="email_icon.svg" name="email" type="email" placeholder="Email Id" autoComplete="email" />
        <Field icon="lock_icon.svg" name="password" type="password" placeholder="Password" autoComplete="current-password" />

        <button type="submit" disabled={pending} className="bg-blue-600 w-full text-white py-2 rounded-full mt-6 disabled:opacity-70">
          {pending ? "Signing in…" : "login"}
        </button>

        <button type="button" onClick={onClose} className="absolute top-5 right-5" aria-label="Close">
          <Image src="/assets/cross_icon.svg" alt="" width={12} height={12} />
        </button>
      </form>
    </div>
  );
}

function Field({ icon, ...input }: { icon: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <div className="border border-gray-200 px-4 py-2 flex items-center gap-2 rounded-full mt-5">
      <Image src={`/assets/${icon}`} alt="" width={16} height={16} />
      <input className="outline-hidden text-sm" required {...input} />
    </div>
  );
}
