# Job Portal

A job board where Companies post Jobs and Candidates apply to them.

The app is being migrated from React + Express (`client/`, `server/`) to **Next.js + Go** (`web/`, `api/`), keeping MongoDB, Clerk, Cloudinary and Sentry. The new stack has feature parity and runs locally; deployment has not happened yet, so production still runs the old app. Progress and decisions: [migration map](https://github.com/AlexMari0/job-portal/issues/1).

Domain terms (Company, Candidate, Job, Hidden Job, Application) are defined in [GLOSSARY.md](GLOSSARY.md).

## Quick start

Requirements: Go 1.26, Node 24 (`nvm use` in `web/`), Docker.

```sh
cp api/.env.example api/.env          # fill in the keys you need, see below
cp web/.env.example web/.env.local
(cd web && npm install)

make seed   # start MongoDB in Docker and load sample data
make dev    # MongoDB + Go API (hot reload) + Next.js
```

Open <http://localhost:3000>. The seed creates two Companies you can sign in with through **Recruiter Login**:

| Company | Email | Password |
|---|---|---|
| Acme Corp | `hr@acme.test` | `password123` |
| Globex | `jobs@globex.test` | `password123` |

Candidates sign in with Clerk (the **Login** button). Your Clerk user is copied into MongoDB the first time it is needed, so the Clerk webhook is not required locally.

## Commands

| Command | What it does |
|---|---|
| `make dev` | MongoDB, Go API on `:8080` (reloads on save), Next.js on `:3000` |
| `make db` / `make db-stop` | Start / stop the local MongoDB container |
| `make seed` | Reset the local database to the sample data (refuses non-local hosts) |
| `make api` / `make web` | Run one side only |
| `make test` | Go unit and integration tests (against the local MongoDB) |
| `make e2e` | Playwright browser tests (see [Testing](#testing)) |
| `make api-types` | Regenerate `web/src/lib/api-types.ts` from `api/openapi.yaml` |

## How it fits together

```
Browser ──▶ Next.js server ──▶ Go API ──▶ MongoDB
              (web/)             (api/)
                                   └────▶ Cloudinary, Clerk API
```

- **The browser never calls the Go API.** Server Components read data and Server Actions change it; both call Go from the Next.js server ([ADR 0001](docs/adr/0001-go-api-called-only-from-nextjs-server.md)). Go needs no CORS.
- **Candidates** sign in with Clerk. Next.js sends the Clerk session token to Go as `Authorization: Bearer …`; Go verifies it with `clerk-sdk-go`.
- **Companies** sign in with email and password. Go issues a JWT, which Next.js keeps in the httpOnly cookie `company_session` and forwards as a Bearer token.
- **The MongoDB documents keep the exact shape Mongoose wrote** (`date` as a double, `users._id` as the Clerk user id, `__v`), so the old and new apps can share one database during the switch.

## Repository layout

```
api/                          Go REST API
├── cmd/api/                  server entry point (config, wiring)
├── cmd/seed/                 local sample data + the indexes Mongoose creates
├── internal/handler/         HTTP handlers, router, errors (Express controllers)
├── internal/store/           MongoDB queries (Mongoose models)
├── internal/auth/            Clerk and Company session middleware
├── internal/upload/          Cloudinary uploader, local disk uploader
├── internal/htmlsafe/        Job description sanitizing and excerpts
├── internal/testdb/          throwaway databases for integration tests
└── openapi.yaml              the API contract
web/                          Next.js 16 (App Router, TypeScript, Tailwind 4)
├── src/app/                  routes: /, /jobs/[id], /applications, /dashboard/…
├── src/components/           shared UI (Navbar, JobListing, RecruiterLogin, …)
├── src/lib/api.ts            public + Candidate calls to the Go API
├── src/lib/company.ts        Company session cookie + Company calls
├── src/lib/api-types.ts      generated from openapi.yaml, do not edit
├── src/proxy.ts              Clerk + route protection (Next.js 16 "middleware")
└── e2e/                      Playwright tests
client/, server/              the old React + Express app (removed after cutover)
```

## Environment variables

Templates with comments: [`api/.env.example`](api/.env.example) and [`web/.env.example`](web/.env.example). Values are never committed (`.env`, `.env.local` are ignored).

**API (`api/.env`)**

| Variable | Needed for | If empty |
|---|---|---|
| `MONGODB_URI`, `MONGODB_DB` | everything | defaults to the local Docker MongoDB, `job-portal` |
| `CLERK_SECRET_KEY` | Candidate routes | they answer 503 |
| `WEB_ORIGIN` | Clerk token check (`azp`) | defaults to `http://localhost:3000` |
| `CLERK_WEBHOOK_SIGNING_SECRET` | `POST /webhooks/clerk` | it answers 503 |
| `COMPANY_JWT_SECRET` | Company sign-in (≥ 32 random characters) | Company routes answer 503 |
| `CLOUDINARY_CLOUD_NAME`, `_API_KEY`, `_API_SECRET` | resume and logo uploads | see `LOCAL_UPLOAD_DIR` |
| `LOCAL_UPLOAD_DIR`, `PUBLIC_URL` | uploads on disk instead of Cloudinary (development only) | uploads answer 503 if Cloudinary is also unset |
| `SENTRY_DSN`, `SENTRY_ENVIRONMENT` | error reporting | errors are only logged |

**Web (`web/.env.local`)**

| Variable | Needed for |
|---|---|
| `API_URL` | the Go API's address (server-side only) |
| `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`, `CLERK_SECRET_KEY` | Candidate sign-in (Clerk development instance) |
| `NEXT_PUBLIC_SENTRY_DSN`, `NEXT_PUBLIC_SENTRY_ENVIRONMENT` | error reporting (off when empty) |

## Adding or changing an endpoint

1. Describe it in [`api/openapi.yaml`](api/openapi.yaml): path, request, responses and error codes.
2. `make api-types` to regenerate the TypeScript types. CI fails if they are out of date.
3. Write an integration test in `api/internal/handler/*_test.go` first (`testdb.New(t)` gives a throwaway database; insert fixtures with production BSON types), then the handler, store query and route in `router.go`.
4. Call it from `web/src/lib/api.ts` or `company.ts` using the generated types, from a Server Component or a Server Action.

API conventions: errors are `{"error": {"code", "message", "fields"}}` with the right HTTP status; JSON is camelCase with `id` instead of `_id`; dates are ISO 8601; Job descriptions are sanitized on write and on read.

## Testing

- **Go:** `make test`. Unit tests plus integration tests against the local MongoDB; each test gets its own database and drops it afterwards.
- **Browser:** `make e2e`. Playwright builds the production Next.js app and starts it on `:3100` with an API on `:8090`, a separate `job-portal-e2e` database and uploads on disk, so it never touches your `make dev` data or Cloudinary. First run: `cd web && npx playwright install chromium`. `E2E_REUSE=1` reuses servers you started yourself. Candidate (Clerk) flows are not covered yet.
- **CI** (GitHub Actions) runs `gofmt`, `go vet`, `go test -race` (on MongoDB 7, because MongoDB 8 does not start on the runners' kernel), checks the generated types, then type-checks, lints and builds the web app. E2E tests do not run in CI yet.

## Things to know

- **`@clerk/nextjs` is pinned to 7.9.23.** Versions 7.9.24+ break Partial Prerendering ([clerk/javascript#10700](https://github.com/clerk/javascript/issues/10700)): every page re-renders in the browser. Do not upgrade until that issue is fixed; an e2e test fails if it comes back.
- **Read the bundled Next.js docs** in `web/node_modules/next/dist/docs/` before relying on older Next.js knowledge; this version (16, Cache Components) differs ([web/AGENTS.md](web/AGENTS.md)).
- **Data that changes per request must be read inside `<Suspense>`** (Cache Components). Pages stream: a Hidden Job's page shows "Job not found" with `noindex` but a 200 status.
- **The Clerk webhook cannot reach `localhost`.** That is fine: the API creates a missing Candidate from the Clerk API on demand.
- **Changing `web/src/lib/*.ts` while `next dev` runs** can leave a stale module (`… is not a function`); restart `make web`.
