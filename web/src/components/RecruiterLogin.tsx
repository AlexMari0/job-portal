"use client";

import Image from "next/image";
import { useActionState, useEffect, useState, useTransition } from "react";
import { toast } from "sonner";
import { loginCompanyAction, registerCompanyAction, type LoginState } from "@/app/company/actions";

const MAX_LOGO_BYTES = 1024 * 1024; // the API enforces the same limit
const LOGO_TYPES = ["image/png", "image/jpeg", "image/webp"];

// Company sign-in / sign-up modal. Sign up has two steps, as before:
// name, email and password, then the logo.
export default function RecruiterLogin({ onClose }: { onClose: () => void }) {
  const [mode, setMode] = useState<"Login" | "Sign Up">("Login");

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
      <div className="relative bg-white p-10 rounded-xl text-slate-500">
        <h1 className="text-center text-2xl text-neutral-700 font-medium">Recruiter {mode}</h1>
        <p className="text-sm">
          {mode === "Login" ? "Welcome Back! Please sign in to continue." : "Create your Company account."}
        </p>

        {mode === "Login" ? <LoginForm /> : <SignUpForm />}

        <p className="mt-5 text-center">
          {mode === "Login" ? "Don't have an account? " : "Already have an account? "}
          <button type="button" className="text-blue-600" onClick={() => setMode(mode === "Login" ? "Sign Up" : "Login")}>
            {mode === "Login" ? "Sign Up" : "Login"}
          </button>
        </p>

        <button type="button" onClick={onClose} className="absolute top-5 right-5" aria-label="Close">
          <Image src="/assets/cross_icon.svg" alt="" width={12} height={12} />
        </button>
      </div>
    </div>
  );
}

function LoginForm() {
  const [state, formAction, pending] = useActionState<LoginState, FormData>(loginCompanyAction, { error: null });
  useEffect(() => {
    if (state.error) toast.error(state.error);
  }, [state]);

  return (
    <form action={formAction} aria-label="Recruiter Login">
      <Field icon="email_icon.svg" name="email" type="email" placeholder="Email Id" autoComplete="email" />
      <Field icon="lock_icon.svg" name="password" type="password" placeholder="Password" autoComplete="current-password" />
      <SubmitButton pending={pending} label="login" pendingLabel="Signing in…" />
    </form>
  );
}

function SignUpForm() {
  const [step, setStep] = useState<"details" | "logo">("details");
  const [details, setDetails] = useState({ name: "", email: "", password: "" });
  const [logo, setLogo] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [pending, startTransition] = useTransition();

  // Free the object URL used for the preview when it changes or on close.
  useEffect(() => () => void (preview && URL.revokeObjectURL(preview)), [preview]);

  function chooseLogo(file: File | undefined) {
    if (!file) return;
    if (!LOGO_TYPES.includes(file.type)) return void toast.error("Logo must be a PNG, JPEG or WebP image");
    if (file.size > MAX_LOGO_BYTES) return void toast.error("Logo must be at most 1 MB");
    setLogo(file);
    setPreview(URL.createObjectURL(file));
  }

  function submit() {
    if (!logo) return void toast.error("Upload your Company logo");
    const form = new FormData();
    Object.entries(details).forEach(([k, v]) => form.append(k, v));
    form.append("image", logo);
    startTransition(async () => {
      const result = await registerCompanyAction(form); // redirects on success
      toast.error(result.message);
      // Problems with the text fields (or a taken email) are fixed on step one.
      if (result.code !== "invalid_file") setStep("details");
    });
  }

  if (step === "details") {
    return (
      <form
        aria-label="Recruiter Sign Up"
        onSubmit={(e) => {
          e.preventDefault();
          setStep("logo");
        }}
      >
        <Field icon="person_icon.svg" name="name" type="text" placeholder="Company Name" autoComplete="organization"
          value={details.name} onChange={(e) => setDetails({ ...details, name: e.target.value })} />
        <Field icon="email_icon.svg" name="email" type="email" placeholder="Email Id" autoComplete="email"
          value={details.email} onChange={(e) => setDetails({ ...details, email: e.target.value })} />
        <Field icon="lock_icon.svg" name="password" type="password" placeholder="Password (6+ characters)" autoComplete="new-password"
          minLength={6} value={details.password} onChange={(e) => setDetails({ ...details, password: e.target.value })} />
        <SubmitButton pending={false} label="next" pendingLabel="" />
      </form>
    );
  }

  return (
    <form
      aria-label="Upload Company Logo"
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
    >
      <div className="flex items-center gap-4 my-10">
        <label htmlFor="logo" className="cursor-pointer">
          {/* eslint-disable-next-line @next/next/no-img-element -- local blob: preview */}
          <img className="w-16 h-16 rounded-full object-cover" src={preview ?? "/assets/upload_area.svg"} alt="Company logo" />
          <input id="logo" type="file" accept="image/png,image/jpeg,image/webp" hidden onChange={(e) => chooseLogo(e.target.files?.[0])} />
        </label>
        <p>
          Upload Company <br /> Logo
        </p>
      </div>
      <SubmitButton pending={pending} label="create account" pendingLabel="Creating…" />
    </form>
  );
}

function SubmitButton({ pending, label, pendingLabel }: { pending: boolean; label: string; pendingLabel: string }) {
  return (
    <button type="submit" disabled={pending} className="bg-blue-600 w-full text-white py-2 rounded-full mt-6 disabled:opacity-70">
      {pending ? pendingLabel : label}
    </button>
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
