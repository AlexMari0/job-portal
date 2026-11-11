import { defineConfig, devices } from "@playwright/test";

// End-to-end tests run against their own stack, so they never touch the
// `make dev` servers or data:
//   Go API   http://localhost:8090  database job-portal-e2e, uploads on disk
//   Next.js  http://localhost:3100  production build
// MongoDB must be running (`make db`). Run with `make e2e`.
const API = "http://localhost:8090";
const WEB = "http://localhost:3100";

export default defineConfig({
  testDir: "e2e",
  // Tests share one seeded database, so run them one at a time.
  workers: 1,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? "github" : "list",
  use: {
    baseURL: WEB,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: [
    {
      command: "go run ./cmd/api",
      cwd: "../api",
      url: `${API}/health`,
      reuseExistingServer: !!process.env.E2E_REUSE, // E2E_REUSE=1: use servers you started yourself
      timeout: 120_000,
      env: {
        PORT: "8090",
        PUBLIC_URL: API,
        MONGODB_DB: "job-portal-e2e",
        WEB_ORIGIN: WEB,
        // Empty values win over api/.env: no real Cloudinary or Sentry.
        CLOUDINARY_CLOUD_NAME: "",
        LOCAL_UPLOAD_DIR: "tmp/e2e-uploads",
        SENTRY_DSN: "",
      },
    },
    {
      command: "npm run build && npx next start -p 3100",
      url: WEB,
      reuseExistingServer: !!process.env.E2E_REUSE, // E2E_REUSE=1: use servers you started yourself
      timeout: 300_000,
      env: {
        API_URL: API,
        LOCAL_UPLOADS_ORIGIN: API, // let next/image load disk uploads
        NEXT_PUBLIC_SENTRY_DSN: "",
      },
    },
  ],
});
