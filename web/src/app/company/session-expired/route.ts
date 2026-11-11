import { cookies } from "next/headers";
import { NextResponse, type NextRequest } from "next/server";
import { COMPANY_COOKIE } from "@/lib/company";

// The API rejected the Company's token (expired, or the Company is gone):
// drop the cookie and go back to the home page to sign in again.
export async function GET(request: NextRequest) {
  (await cookies()).delete(COMPANY_COOKIE);
  return NextResponse.redirect(new URL("/", request.url));
}
