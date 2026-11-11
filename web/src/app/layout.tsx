import { ClerkProvider } from "@clerk/nextjs";
import type { Metadata } from "next";
import { Outfit } from "next/font/google";
import { Toaster } from "sonner";
import "./globals.css";

const outfit = Outfit({ variable: "--font-outfit", subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Job Portal",
  description: "Find your next job from top companies",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={`${outfit.variable} h-full antialiased`}>
      <body className="min-h-full flex flex-col">
        {/* Clerk's provider goes inside <body> (Next.js 16 + Cache Components).
            @clerk/nextjs is pinned to 7.9.23: 7.9.24+ breaks the Partial Prerender
            resume (github.com/clerk/javascript/issues/10700); e2e/public.spec.ts
            fails if that regression comes back. */}
        <ClerkProvider>
          {children}
          <Toaster richColors position="top-right" />
        </ClerkProvider>
      </body>
    </html>
  );
}
