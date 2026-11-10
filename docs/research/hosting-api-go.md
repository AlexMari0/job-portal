# Research: Where to host the Go API

Ticket: #3 (child of map #1, "Migrasi ke Next.js + Go + MongoDB"). Researched 2026-11-10 against official docs and pricing pages. Prices and limits change, so re-check the linked pages before you commit to one.

## Question

Fly.io, Railway, Render, or Go on Vercel: which should host a small Go REST API (~15 endpoints, a Clerk webhook, multipart upload to Cloudinary) next to a Next.js frontend on Vercel and an existing MongoDB Atlas cluster? Low cost and free tier come first.

## Recommendation

**Fly.io**: one `shared-cpu-1x` 256 MB Machine in the region nearest the Atlas cluster, deployed from the Dockerfile that `fly launch` generates, with `auto_stop_machines = "suspend"` (or `min_machines_running = 1` to keep it always on).

- An always-on 256 MB Machine is listed at $2.19/month in the cheapest regions. That is under Fly's $5 waiver for personal orgs, so the bill should be $0. A small region premium (for example `sin`) should still fit under $5. Fly does not publish the 256 MB price for each region, so check the first invoice.
- It is a real long-running Go server: no 4.5 MB body cap, no serverless handler shape, and no beta runtime. A cold start takes about 2 s from stop, or a few hundred ms from suspend. Render Free takes about 1 minute.
- The catch: Fly needs a **credit card** once the trial ends. It has no free tier for new orgs.

**Fallback if you can't add a credit card:** Go on **Vercel** (Go Framework Preset, beta), in the same account as the Next.js app. It is free on Hobby with no card. Set the function region to match Atlas. Keep uploads under 4.5 MB, or upload straight from the browser to Cloudinary with a signed upload.

**Not recommended:** Render Free. The service sleeps after 15 minutes and takes about 1 minute to wake, which gives a poor first-load UX. Railway Free/Trial gives $1/month of credit or a one-time $5 grant, which is too thin to rely on. Railway Hobby is $5/month and is fine if you are willing to pay.

## Comparison

| | Fly.io | Railway | Render | Vercel (Go) |
|---|---|---|---|---|
| Free tier | None for new orgs; bill < $5/month waived; $1 trial credit for 7 days, card needed after | Free plan: $1 credit/month; Trial: one-time $5 | Free web service: 750 instance-hours/month | Hobby free (non-commercial only) |
| Cheapest always-on | $2.19/month (256 MB, cheapest regions) | Hobby $5/month incl. $5 usage; RAM $10/GB-month, CPU $20/vCPU-month | Starter $7/month (512 MB, 0.5 CPU) | n/a (serverless, Fluid compute) |
| Sleep / cold start | Optional autostop; stop -> ~2 s+ start, suspend -> few hundred ms | Opt-in "serverless" sleep after 5-10 min idle; first request may 502 | Free sleeps after 15 min idle; ~1 min spin-up | Per-instance cold start; no idle sleep billing |
| Deploy method | `fly launch` detects Go and **generates a Dockerfile** + `fly.toml`; `fly deploy` | Railpack auto-detects `go.mod` (static binary, CGO off); uses a Dockerfile if one exists | Native Go runtime (always latest Go; pin with Docker) or Docker | Go preset: root `go.mod` + `main.go`/`cmd/api/main.go`, listen on `PORT`; runtime is **Beta** |
| Env vars | `fly secrets set` (CLI) | Service Variables (dashboard/CLI) | Dashboard env vars & secrets | Project Environment Variables (same place as Next.js) |
| Regions | Many incl. Asia; price varies by region | US West, US East, EU West (Amsterdam), Southeast Asia (Singapore) | Oregon, Ohio, Virginia, Frankfurt, Singapore | Default `iad1`; Hobby = single region, changeable |
| Request limits | none notable | none notable | Free: no scaling, filesystem ephemeral, may restart anytime | **4.5 MB** request/response body; 300 s max duration on Hobby |
| Webhooks (Clerk/Svix) | Good (always on or fast wake) | Good on Hobby without sleep | Free: first delivery may time out while waking; Svix retries | Good |

## Key facts with sources

### Fly.io
- "New organizations don't have a free tier or a monthly free usage allowance." / "if your personal organization's monthly bill comes to less than $5, we waive it." Trial: $1 credit, 7 days, no card to start, "After the trial ends, you'll need to add a credit card". [Fly pricing](https://docs.fly.io/about/pricing)
- `shared-cpu-1x` 256 MB: $2.19/month; 512 MB: $3.69/month (cheapest regions `iad`/`ewr`). "Machine prices vary by region": a 1 GB machine is $6.70 in Ashburn and $8.50 in Singapore. A stopped Machine bills only rootfs at $0.15/GB-month. [Fly pricing](https://docs.fly.io/about/pricing)
- Autostop: `stop` or `suspend` when idle "for several minutes". "A cold start takes about 2 seconds or more"; "Resuming a suspended Machine takes a few hundred milliseconds." `min_machines_running` keeps N machines up in the primary region. [Fly autostop/autostart](https://docs.fly.io/launch/autostop-autostart)
- `fly launch` detects Go and generates a Dockerfile and `fly.toml`. [Fly Go guide](https://docs.fly.io/languages-and-frameworks/golang)

### Railway
- Free: $0 with $1 credit/month, 1 replica, 0.5 GB RAM, 1 vCPU per service. Trial: one-time $5. Hobby: $5/month including $5 usage, overage charged. RAM $10/GB-month, CPU $20/vCPU-month, egress $0.05/GB. [Railway plans](https://docs.railway.com/reference/pricing/plans)
- App sleeping is an opt-in setting. A service is inactive after 5 minutes without outbound packets, and the first wake request may return 502. [Railway app sleeping](https://docs.railway.com/reference/app-sleeping)
- Regions: US West (California), US East (Virginia), EU West (Amsterdam), Southeast Asia (Singapore). [Railway regions](https://docs.railway.com/reference/deployment-regions)
- Railpack is the default builder and detects Go from `go.mod`. If a Dockerfile exists, Railway uses it. [Railpack](https://docs.railway.com/builds/railpack), [Railpack Go](https://railpack.com/languages/golang/)

### Render
- Free web service: "spins down a Free web service that goes 15 minutes without receiving any inbound traffic"; spin-up "takes about one minute"; 750 free instance hours/workspace/month; ephemeral filesystem; "Render may restart the service at any time". [Render Free](https://render.com/docs/free)
- Go is a native runtime that always uses the latest stable Go. To pin a version, deploy with Docker. [Render language support](https://render.com/docs/language-support)
- Regions: Oregon, Ohio, Virginia, Frankfurt, Singapore. [Render regions](https://render.com/docs/regions)
- The paid Starter instance is $7/month (512 MB, 0.5 CPU), on top of the workspace plan (Hobby $0). [Render pricing](https://render.com/pricing). The plan details on this page did not render for the fetch tool, so verify them in a browser.

### Vercel (Go)
- "The Go runtime is available in Beta on all plans." The Go Framework Preset runs a standard `net/http`, chi, or gin server from a root `go.mod`, and the server must listen on `PORT`. It can sit alongside Next.js in one project via Services. [Vercel Go runtime](https://vercel.com/docs/functions/runtimes/go)
- Request/response body maximum is **4.5 MB** (413 `FUNCTION_PAYLOAD_TOO_LARGE`). Hobby max duration is 300 s. [Vercel function limits](https://vercel.com/docs/functions/limitations)
- Default function region is `iad1`; Hobby gets a single region, which you can change in settings or `vercel.json`. [Vercel function regions](https://vercel.com/docs/functions/configuring-functions/region)
- Hobby is free, "non-commercial, personal use only", and includes 4 active-CPU-hours, 360 GB-hrs of memory, and 1M invocations. [Vercel Hobby](https://vercel.com/docs/plans/hobby)
- Note: the current Express backend (`server/vercel.json`) already runs on Vercel via `@vercel/node`.

### Webhooks (Clerk -> Svix) and Atlas
- Clerk delivers webhooks through Svix. Svix waits up to 30 s for a 2xx and makes up to 8 attempts: immediately, then after 5 s, 5 min, 30 min, 2 h, 5 h, 10 h, and 10 h. So a sleeping Render Free service would usually get the event on a retry, but `user.created` can arrive late. [Svix retries](https://docs.svix.com/retries)
- Atlas Free (M0): maximum of 500 connections, available only in a subset of regions, no private endpoints or peering. That means a public connection string plus an IP access list, often `0.0.0.0/0`, because none of these hosts give a fixed egress IP on the cheap tiers. [Atlas free limits](https://www.mongodb.com/docs/atlas/reference/free-shared-limitations/)

## Latency guidance

Put the Go API in the **same cloud region as the Atlas cluster**. Every API request makes one or more DB round trips, while the browser-to-API hop happens only once per request. Then set the Vercel function region for the Next.js server code to the same region.
- Atlas in AWS `us-east-1` (Virginia): Fly `iad`, Railway US East, Render Virginia, Vercel `iad1` (the default).
- Atlas in AWS `ap-southeast-1` (Singapore), the closest to Indonesian users: Fly `sin`, Railway Southeast Asia, Render Singapore, Vercel `sin1`.

Check which region the existing Atlas cluster uses (Atlas UI -> cluster -> region) before choosing.

## Open points for the owner
- Can you add a credit card? Yes means Fly.io. No means Vercel Go (beta), or Render Free if you accept about 1 minute cold starts.
- Find out the Atlas cluster region; it decides the host region.
- Find out the maximum upload size for resumes and logos. If it can exceed 4.5 MB, Vercel needs direct-to-Cloudinary signed uploads.
