import { clerkMiddleware, createRouteMatcher } from "@clerk/nextjs/server";

// Pages only a signed-in Candidate may open.
const isCandidatePage = createRouteMatcher(["/applications(.*)"]);

// Next.js 16 "proxy" (formerly middleware): runs before each request and
// lets Clerk read the session cookie, so auth() works on the server.
// Protecting here redirects before rendering starts; inside a streamed
// page the response status would already be 200.
export default clerkMiddleware(async (auth, req) => {
  if (isCandidatePage(req)) await auth.protect();
});

export const config = {
  matcher: [
    // Everything except Next.js internals and static files.
    "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
  ],
};
