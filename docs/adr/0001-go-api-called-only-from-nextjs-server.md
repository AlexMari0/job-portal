# The Go API is called only from the Next.js server

The browser never talks to the Go API. Every request to Go comes from the Next.js server: Server Components for reads, Server Actions for writes. Next.js is therefore the API's only client, and the Go API needs no CORS.

## Why

- A Company's session token can sit in an httpOnly cookie on the Next.js domain and be forwarded server-side, so browser JavaScript never sees it.
- One trust boundary: Go only has to accept server-to-server calls carrying a Clerk session token (Candidate) or a Company token.
- Server Actions are a core concept of modern Next.js, and the project owner wants to learn them.

## Considered options

- **Browser calls Go directly from client components with a Bearer token.** Rejected: it needs CORS on Go, and the Company token would have to live in browser-readable storage (as `localStorage` does today).

## Consequences

- Each request makes an extra hop (browser → Next.js → Go). Deploy Next.js and Go in nearby regions.
- File uploads also pass through Next.js unless they go straight to Cloudinary. Vercel's function body limit applies; see the upload-flow decision.
