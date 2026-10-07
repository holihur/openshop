import AxeBuilder from "@axe-core/playwright";
import { test, expect, type Page } from "@playwright/test";

// Accessibility scans with axe-core against the WCAG 2 A/AA rule set. Run on
// Chromium only: the DOM and computed styles are engine-independent for these
// checks, so scanning once keeps the suite fast.
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:18081";
const opsBase = process.env.E2E_OPS_BASE_URL ?? "http://localhost:18082";

async function expectAccessible(page: Page) {
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  const summary = results.violations.map(
    (v) => `${v.id}: ${v.help} (${v.nodes.length} node(s)) — ${v.nodes[0]?.target?.join(" ")}`,
  );
  expect(summary, `axe violations on ${page.url()}`).toEqual([]);
}

test("the storefront home page has no accessibility violations", async ({ page }) => {
  await page.goto(`${baseURL}/`);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  await expectAccessible(page);
});

test("the product detail page has no accessibility violations", async ({ page }) => {
  await page.goto(`${baseURL}/`);
  await page.locator('a[href^="/products/"]').first().click();
  await expect(page).toHaveURL(/\/products\/[^/]+$/);
  await expectAccessible(page);
});

test("the sign-in page has no accessibility violations", async ({ page }) => {
  await page.goto(`${baseURL}/login`);
  await expect(page.getByLabel("Email or phone")).toBeVisible();
  await expectAccessible(page);
});

test("the ops dashboard has no accessibility violations", async ({ page }) => {
  await page.goto(`${opsBase}/login`);
  await page.getByLabel("Email or phone").fill("admin@openshop.local");
  await page.getByLabel("Password").fill("admin12345");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("link", { name: "Dashboard" })).toBeVisible();
  await expectAccessible(page);
});
