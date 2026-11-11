import { expect, test } from "@playwright/test";
import { API, GLOBEX, companyToken, jobCardTitles, reseed } from "./helpers";

test.beforeAll(reseed);

test("home lists visible Jobs, newest first, without Hidden Jobs", async ({ page }) => {
  await page.goto("/");
  const titles = jobCardTitles(page);
  await expect(titles).toHaveCount(6);
  await expect(titles.first()).toHaveText("Backend Engineer (Go)"); // posted most recently
  await expect(page.getByText("Marketing Lead (hidden)")).toHaveCount(0);
});

test("search by title and filter by category", async ({ page }) => {
  await page.goto("/");
  await page.getByPlaceholder("Search for jobs").fill("engineer");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(jobCardTitles(page)).toHaveText(["Backend Engineer (Go)", "Frontend Engineer (React)", "Network Engineer"]);

  // Remove the search chip, then filter by a category instead.
  await page.getByRole("button", { name: "Remove engineer" }).click();
  await page.getByLabel("Designing").check();
  await expect(jobCardTitles(page)).toHaveText(["Product Designer"]);
});

test("Job detail shows the Company's other Jobs", async ({ page }) => {
  await page.goto("/");
  await page.locator("#job-list ~ div > div").filter({ hasText: "Backend Engineer (Go)" }).getByRole("link", { name: "Learn more" }).click();

  await expect(page).toHaveURL(/\/jobs\/[0-9a-f]{24}$/);
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("Backend Engineer (Go)");
  await expect(page).toHaveTitle("Backend Engineer (Go) at Acme Corp | Job Portal");
  await expect(page.getByText("More jobs from Acme Corp")).toBeVisible();
  await expect(page.getByRole("heading", { level: 4 })).toHaveText(["Frontend Engineer (React)", "Data Analyst"]);
  // Description HTML is rendered (sanitized by the API).
  await expect(page.locator(".rich-text strong")).toHaveText("Backend Engineer (Go)");
});

test("Hidden Job page says not found; old URLs redirect", async ({ page }) => {
  const token = await companyToken(GLOBEX.email, GLOBEX.password);
  const jobs = (await (await fetch(`${API}/company/jobs`, { headers: { Authorization: `Bearer ${token}` } })).json()) as {
    id: string;
    visible: boolean;
  }[];
  const hidden = jobs.find((j) => !j.visible)!;

  await page.goto(`/jobs/${hidden.id}`);
  await expect(page.getByRole("heading", { name: "Job not found" })).toBeVisible();

  const visible = jobs.find((j) => j.visible)!;
  await page.goto(`/apply-job/${visible.id}`);
  await expect(page).toHaveURL(`/jobs/${visible.id}`);
});

// Regression guard for github.com/clerk/javascript/issues/10700: when the
// prerendered shell cannot be resumed, React logs error #419 in the browser
// and re-renders the whole page on the client (losing server rendering).
test("pages resume their prerendered shell (no React #419)", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error" && /#419|resume|switched to client rendering/i.test(msg.text())) errors.push(msg.text());
  });
  for (let i = 0; i < 3; i++) {
    await page.goto("/");
    await expect(jobCardTitles(page).first()).toBeVisible();
  }
  expect(errors).toEqual([]);
});
