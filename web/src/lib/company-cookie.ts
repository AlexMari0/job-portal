// Name of the httpOnly cookie holding a Company's session token. In its own
// module (no server-only imports) so proxy.ts can share it with lib/company.ts.
export const COMPANY_COOKIE = "company_session";
