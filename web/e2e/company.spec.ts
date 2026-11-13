import path from "node:path";
import { expect, test } from "@playwright/test";
import { ACME, expectToast, jobCardTitles, loginCompany, reseed } from "./helpers";

// One Company session through the whole file, as a recruiter would work.
test.describe.configure({ mode: "serial" });
test.beforeAll(reseed);

test("wrong password is rejected", async ({ page }) => {
  await loginCompany(page, ACME.email, "wrong-password");
  await expectToast(page, "Invalid email or password");
  await expect(page).toHaveURL("/");
});

test.describe("signed in as Acme", () => {
  test.beforeEach(async ({ page }) => {
    await loginCompany(page, ACME.email, ACME.password);
    await expect(page).toHaveURL("/dashboard/manage-jobs");
  });

  test("dashboard greets the Company and lists its Jobs", async ({ page }) => {
    await expect(page.getByText(`Welcome, ${ACME.name}!`)).toBeVisible();
    const rows = page.locator("tbody tr");
    await expect(rows).toHaveCount(3);
    await expect(rows.filter({ hasText: "Backend Engineer (Go)" }).locator("td").nth(4)).toHaveText("1"); // applicants
  });

  test("hiding a Job removes it from the public list", async ({ page }) => {
    const row = page.locator("tbody tr").filter({ hasText: "Data Analyst" });
    await row.getByLabel("Visible").uncheck();
    await expectToast(page, "Visibility changed");
    await page.reload();
    await expect(page.locator("tbody tr").filter({ hasText: "Data Analyst" }).getByLabel("Visible")).not.toBeChecked();

    await page.goto("/");
    await expect(jobCardTitles(page)).not.toContainText(["Data Analyst"]);
  });

  test("posting a Job with the Quill editor", async ({ page }) => {
    await page.getByRole("link", { name: "Add Job" }).click();
    await page.getByLabel("Job Title").fill("E2E Platform Engineer");
    const editor = page.locator(".ql-editor");
    await editor.click();
    await page.keyboard.type("Run the ");
    await page.keyboard.press("ControlOrMeta+b");
    await page.keyboard.type("platform");
    await page.getByLabel("Job Category").selectOption("Programming");
    await page.getByLabel("Job Location").selectOption("Chennai");
    await page.getByLabel("Job Salary").fill("120000");
    await page.getByRole("button", { name: "ADD" }).click();
    await expectToast(page, "Job Added");
    await expect(page.getByLabel("Job Title")).toHaveValue(""); // form cleared

    await page.getByRole("link", { name: "Manage Jobs" }).click();
    await expect(page.locator("tbody tr").first()).toContainText("E2E Platform Engineer"); // newest first

    await page.goto("/");
    await page.locator("#job-list ~ div > div").filter({ hasText: "E2E Platform Engineer" }).getByRole("link", { name: "Learn more" }).click();
    await expect(page.locator(".rich-text strong")).toHaveText("platform"); // bold kept
    // The default level was accepted (shown next to the title).
    await expect(page.locator("span.flex").filter({ hasText: "Beginner level" })).toBeVisible();
  });

  test("an empty description is refused", async ({ page }) => {
    await page.getByRole("link", { name: "Add Job" }).click();
    await page.getByLabel("Job Title").fill("No description");
    await page.getByLabel("Job Salary").fill("1");
    await page.getByRole("button", { name: "ADD" }).click();
    await expect(page.getByText("required")).toBeVisible(); // field error under the editor
    // What was typed is kept, so the recruiter only has to add the description.
    await expect(page.getByLabel("Job Title")).toHaveValue("No description");
    await expect(page.getByLabel("Job Salary")).toHaveValue("1");
  });

  test("accepting an Application", async ({ page }) => {
    await page.getByRole("link", { name: "View Applications" }).click();
    const row = page.locator("tbody tr").filter({ hasText: "Alice Santoso" });
    await expect(row).toContainText("Backend Engineer (Go)");
    await row.getByRole("button", { name: "Change status" }).hover();
    await row.getByRole("button", { name: "Accept" }).click();
    await expectToast(page, "Status Changed");
    await expect(row.locator("td").last()).toHaveText("Accepted");
  });

  test("logging out", async ({ page }) => {
    await page.getByRole("button", { name: "Account menu" }).hover();
    await page.getByRole("button", { name: "Logout" }).click();
    await expect(page).toHaveURL("/");
    await page.goto("/dashboard/manage-jobs");
    await expect(page).toHaveURL("/"); // no session: sent home by proxy.ts
  });
});

test("signing up a new Company with a logo", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Recruiter Login" }).click();
  await page.getByRole("button", { name: "Sign Up" }).click();

  const details = page.getByRole("form", { name: "Recruiter Sign Up" });
  await details.getByPlaceholder("Company Name").fill("E2E Company");
  await details.getByPlaceholder("Email Id").fill("e2e@company.test");
  await details.getByPlaceholder(/Password/).fill("secret123");
  await details.getByRole("button", { name: "next" }).click();

  const logo = page.getByRole("form", { name: "Upload Company Logo" });
  await logo.locator('input[type="file"]').setInputFiles(path.join(__dirname, "fixtures", "logo.png"));
  await expect(logo.getByAltText("Company logo")).toHaveAttribute("src", /^blob:/); // preview
  await logo.getByRole("button", { name: "create account" }).click();

  await expect(page).toHaveURL("/dashboard/manage-jobs");
  await expect(page.getByText("Welcome, E2E Company!")).toBeVisible();
  await expect(page.getByText("No Jobs posted yet")).toBeVisible();
  // The logo was stored (on disk in e2e) and is shown in the header.
  const logoImg = page.getByRole("button", { name: "Account menu" }).locator("img");
  await expect(logoImg).toHaveAttribute("src", /uploads/);
  // ...and actually loads (a broken image has naturalWidth 0).
  await expect.poll(() => logoImg.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth)).toBeGreaterThan(0);
});

test("signing up with a taken email goes back to step one", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Recruiter Login" }).click();
  await page.getByRole("button", { name: "Sign Up" }).click();
  const details = page.getByRole("form", { name: "Recruiter Sign Up" });
  await details.getByPlaceholder("Company Name").fill("Acme Again");
  await details.getByPlaceholder("Email Id").fill(ACME.email);
  await details.getByPlaceholder(/Password/).fill("secret123");
  await details.getByRole("button", { name: "next" }).click();
  await page.locator('input[type="file"]').setInputFiles(path.join(__dirname, "fixtures", "logo.png"));
  await page.getByRole("button", { name: "create account" }).click();

  await expectToast(page, "already registered");
  await expect(page.getByRole("form", { name: "Recruiter Sign Up" }).getByPlaceholder("Email Id")).toHaveValue(ACME.email);
});
