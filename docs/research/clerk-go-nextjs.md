# Research: Integrasi Clerk di Next.js (App Router) + Go

Ticket: #4 (child of #1 "Migrasi ke Next.js + Go + MongoDB"). Researched 2026-11-10.

Scope: Clerk authenticates **Candidates** only. Company login stays on its own JWT (`token` header, `server/middleware/authMiddleware.js`) and is out of scope here, except for CORS.

## TL;DR / recommended pattern

Next.js (`@clerk/nextjs`, `clerkMiddleware()`) gets the Clerk **session token** with `getToken()` (server: `await auth()`; client: `useAuth()`), and sends it to the Go API as `Authorization: Bearer <token>`. The Go API uses `clerk-sdk-go/v2` `clerkhttp.RequireHeaderAuthorization(clerkhttp.AuthorizedPartyMatches(<Next.js origin>))` and reads the user id from `claims.Subject`. Clerk webhooks hit a public Go route `POST /webhooks` verified with `svix-webhooks/go` on the **raw body**, syncing `user.created|updated|deleted` into the MongoDB `User` collection. No JWT template is needed: the default session token already carries `sub` = Clerk user id, which is the `User._id` we store today.

## Current behaviour (to preserve)

- Client: `@clerk/react` `useAuth().getToken()` then `Authorization: Bearer ${token}` via axios (`client/src/context/AppContext.jsx`, `client/src/pages/ApplyJob.jsx`, `client/src/pages/Applications.jsx`).
- Server: `app.use(clerkMiddleware())` from `@clerk/express`; controllers read `const { userId } = req.auth()` (`server/controllers/userController.js`).
- Webhooks: `POST /webhooks` (`server/controllers/webhooks.js`), verified with `svix` using `CLERK_WEBHOOK_SECRET`. Events:
  - `user.created` -> `User.create({_id: data.id, email: data.email_addresses[0].email_address, name: first_name + " " + last_name, image: data.image_url, resume: ""})`
  - `user.updated` -> `findByIdAndUpdate(data.id, {email, name, image})`
  - `user.deleted` -> `findByIdAndDelete(data.id)`
- Known issues in the current handler worth fixing in the port:
  - It verifies `JSON.stringify(req.body)` (re-serialized JSON), not the raw body. Svix warns this can break signature verification [svix-verify].
  - The `default:` case never sends a response, so the request hangs. Clerk then retries because there is no 2xx [clerk-sync].
  - `email_addresses[0]` is not necessarily the primary email (`primary_email_address_id` exists on the payload).
  - `first_name`/`last_name` can be null, which gives the name `"null null"`.

## 1. Next.js: getting the token and calling Go

- `auth()` from `@clerk/nextjs/server` is server-only (Server Components, Route Handlers, Server Actions), is **async** (`await auth()`), and needs `clerkMiddleware()` configured [clerk-auth].
- `getToken(options?: { template?: string }) => Promise<string | null>`. Without `template` you get the default session token. Passing `template` makes a network request and counts toward rate limits [clerk-auth-object].
- For cross-origin requests (our case: Go is on a different domain), the token must be sent as `Authorization: Bearer <token>`. Cookies are only sent automatically to the same origin [clerk-requests].
- The session token lives 60 s. The frontend SDK refreshes it about every 50 s [clerk-how]. So call `getToken()` **per request** and don't store the token.

```ts
// middleware.ts
import { clerkMiddleware } from '@clerk/nextjs/server'
export default clerkMiddleware()
export const config = { matcher: ['/((?!_next|.*\\..*).*)', '/(api|trpc)(.*)'] }
```

```ts
// lib/api.server.ts  (Server Components / Server Actions / Route Handlers)
import 'server-only'
import { auth } from '@clerk/nextjs/server'

export async function goFetch(path: string, init: RequestInit = {}) {
  const { getToken } = await auth()
  const token = await getToken() // default session token, no template
  return fetch(`${process.env.API_URL}${path}`, {
    ...init,
    headers: { ...init.headers, ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    cache: 'no-store', // per-user data: never cache across users
  })
}
```

```tsx
// client component
'use client'
import { useAuth } from '@clerk/nextjs'

export function useGoFetch() {
  const { getToken } = useAuth()
  return async (path: string, init: RequestInit = {}) => {
    const token = await getToken()
    return fetch(`${process.env.NEXT_PUBLIC_API_URL}${path}`, {
      ...init,
      headers: { ...init.headers, Authorization: `Bearer ${token}` },
    })
  }
}
```

**JWT templates: not needed.** The default v2 session token has `sub` (user id), `sid`, `azp`, `iss`, `exp`, `nbf`, `iat`, `jti`, `v`, `sts`, and so on. It has **no email** [clerk-tokens]. We already sync email/name/image into Mongo through webhooks, so the user id is enough. If extra claims are ever needed, prefer customizing the session token over templates. Keep custom claims under about 1.2 KB, because the token is stored in a cookie with a 4 KB limit [clerk-tokens].

## 2. Go: verifying the token with clerk-sdk-go v2

Module: `github.com/clerk/clerk-sdk-go/v2`. HTTP middleware: `github.com/clerk/clerk-sdk-go/v2/http` [clerk-go-readme] [clerk-go-http].

- `clerk.SetKey(secretKey)` once at startup (`CLERK_SECRET_KEY`).
- `clerkhttp.WithHeaderAuthorization(opts...)` parses `Authorization: Bearer` and puts the claims in the context. It does not reject the request.
- `clerkhttp.RequireHeaderAuthorization(opts...)` does the same but returns **403** when valid claims are missing [clerk-go-readme]. You can override this with `AuthorizationFailureHandler` (for example to return 401 JSON).
- `clerk.SessionClaimsFromContext(r.Context())` returns `(*clerk.SessionClaims, bool)`. The user id is `claims.Subject`.
- Options: `AuthorizedPartyMatches(parties...)` / `AuthorizedParty(func)`, `Leeway(d)`, `JWKSClient(c)`, `JSONWebKey(pem)`, `CustomClaimsConstructor`, `AuthorizationJWTExtractor`, `ProxyURL`, `Satellite`, `Clock` [clerk-go-http].
- **JWKS caching:** if no `JSONWebKey` is given, the middleware fetches the JWKS from the Clerk API and **caches it for 1 hour** [clerk-go-http]. With `JSONWebKey(pem)` (Dashboard -> API keys -> JWKS Public Key) verification makes no network calls, but you must update the key yourself when it rotates [clerk-go-http] [clerk-manual-jwt].
- **Set `azp`.** Clerk "highly recommends" setting authorized parties, and warns that without it the app is open to CSRF [clerk-manual-jwt]. Pass the Next.js origin(s).

```go
import (
    "github.com/clerk/clerk-sdk-go/v2"
    clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func main() {
    clerk.SetKey(os.Getenv("CLERK_SECRET_KEY"))

    requireCandidate := clerkhttp.RequireHeaderAuthorization(
        clerkhttp.AuthorizedPartyMatches(os.Getenv("FRONTEND_ORIGIN")), // e.g. https://jobportal.example.com
        clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            http.Error(w, `{"success":false,"message":"unauthorized"}`, http.StatusUnauthorized)
        })),
    )

    mux := http.NewServeMux()
    mux.Handle("GET /api/users/user", requireCandidate(http.HandlerFunc(getUserData)))
    mux.HandleFunc("POST /webhooks", clerkWebhook) // public, NOT behind Clerk middleware
    http.ListenAndServe(":8080", withCORS(mux))
}

func getUserData(w http.ResponseWriter, r *http.Request) {
    claims, ok := clerk.SessionClaimsFromContext(r.Context())
    if !ok {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }
    userID := claims.Subject // == Mongo User._id (e.g. "user_2abc...")
    _ = userID
}
```

## 3. Clerk webhooks in Go (svix)

- Clerk webhooks are delivered by Svix. Use `github.com/svix/svix-webhooks/go`: `svix.NewWebhook(secret)` and `wh.Verify(payload []byte, headers http.Header) error` [svix-verify] [svix-go-src].
- You **must** pass the raw body (`io.ReadAll(r.Body)`). Decode the JSON only after verification [svix-verify].
- `NewWebhook` accepts the secret with or without the `whsec_` prefix. The timestamp tolerance is **5 minutes** (replay protection) [svix-go-src].
- Clerk's docs now name the env var `CLERK_WEBHOOK_SIGNING_SECRET`. The current code uses `CLERK_WEBHOOK_SECRET`; either works, so pick one at migration time [clerk-sync].
- The route must be public, with no Clerk auth middleware. A 2xx response marks the event delivered. 4xx/5xx or no response triggers a retry [clerk-sync]. Handlers should be idempotent: use upsert on `user.created`, and ignore not-found on delete.
- Relevant events: `user.created`, `user.updated`, `user.deleted` (the current code handles these). Clerk also lists `user.merged` [clerk-sync]. Unknown events should get a `200` and be ignored.

```go
import svix "github.com/svix/svix-webhooks/go"

var wh, _ = svix.NewWebhook(os.Getenv("CLERK_WEBHOOK_SECRET"))

type clerkEvent struct {
    Type string          `json:"type"`
    Data json.RawMessage `json:"data"`
}

func clerkWebhook(w http.ResponseWriter, r *http.Request) {
    payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
    if err != nil {
        http.Error(w, "bad body", http.StatusBadRequest)
        return
    }
    if err := wh.Verify(payload, r.Header); err != nil {
        http.Error(w, "invalid signature", http.StatusBadRequest)
        return
    }

    var ev clerkEvent
    if err := json.Unmarshal(payload, &ev); err != nil {
        http.Error(w, "bad json", http.StatusBadRequest)
        return
    }
    switch ev.Type {
    case "user.created", "user.updated": // upsert {_id: data.id, email (primary), name, image}
    case "user.deleted": // delete by data.id
    }
    w.WriteHeader(http.StatusOK) // always 2xx for handled/ignored events
}
```

For local testing, use the Clerk Dashboard webhook "Testing" tab or a tunnel (e.g. ngrok) to reach `POST /webhooks`.

## 4. CORS / cross-origin

- Because Go is on a different origin, the browser does **not** send the Clerk `__session` cookie to it. That cookie is set on the app's domain and is only used for same-origin requests [clerk-how] [clerk-manual-jwt]. Always use the `Authorization: Bearer` header. Don't use cookies, and don't set `credentials: 'include'`.
- The Go API needs CORS for browser calls from client components:
  - `Access-Control-Allow-Origin: <Next.js origin>` (explicit, not `*`, to match `azp`)
  - `Access-Control-Allow-Headers: Authorization, Content-Type, token`. `token` is the Company JWT header we use today.
  - `Access-Control-Allow-Methods: GET, POST, OPTIONS`
  - Handle `OPTIONS` preflight before the auth middleware. Preflights carry no `Authorization` header.
- Calls from Server Components, Server Actions and Route Handlers are server-to-server, so CORS does not apply. `azp` validation still does: the session token's `azp` is the origin where the token was minted, which is the Next.js app, so the same `AuthorizedPartyMatches` value works for both paths.
- Clerk's own Frontend API domain does not change. Only Next.js talks to Clerk directly.

## Env vars (proposed)

| Where | Var |
|---|---|
| Next.js | `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`, `CLERK_SECRET_KEY`, `NEXT_PUBLIC_API_URL` / `API_URL` |
| Go | `CLERK_SECRET_KEY` (or a PEM JWKS public key for networkless verification), `CLERK_WEBHOOK_SECRET`, `FRONTEND_ORIGIN` |

## Sources

- [clerk-auth] Clerk, `auth()` (Next.js App Router): https://clerk.com/docs/reference/nextjs/app-router/auth
- [clerk-auth-object] Clerk, Auth object / `getToken()`: https://clerk.com/docs/reference/backend/types/auth-object
- [clerk-requests] Clerk, Making requests (same-origin vs cross-origin): https://clerk.com/docs/guides/development/making-requests
- [clerk-how] Clerk, How Clerk works (60 s token, `__session` / `__client`): https://clerk.com/docs/guides/how-clerk-works/overview
- [clerk-tokens] Clerk, Session tokens (default claims v2, size limits): https://clerk.com/docs/guides/sessions/session-tokens
- [clerk-manual-jwt] Clerk, Manual JWT verification (azp, JWKS endpoints, networkless): https://clerk.com/docs/guides/sessions/manual-jwt-verification
- [clerk-sync] Clerk, Sync data with webhooks: https://clerk.com/docs/guides/development/webhooks/syncing
- [clerk-go-readme] clerk-sdk-go v2 README: https://github.com/clerk/clerk-sdk-go/blob/v2/README.md
- [clerk-go-http] clerk-sdk-go v2 `http` package (AuthorizationParams, options, 1 h JWKS cache): https://pkg.go.dev/github.com/clerk/clerk-sdk-go/v2/http
- [svix-verify] Svix, Verifying payloads: https://docs.svix.com/receiving/verifying-payloads/how
- [svix-go-src] svix-webhooks Go source (`webhook.go`, 5 min tolerance, `whsec_` handling): https://github.com/svix/svix-webhooks/blob/main/go/webhook.go
