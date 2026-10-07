import { test, expect, type Page } from "@playwright/test";

// The ops console is a separate binary on its own (internal) port.
const opsBase = process.env.E2E_OPS_BASE_URL ?? "http://localhost:18082";

async function signIn(page: Page, identifier = "admin@openshop.local", password = "admin12345") {
  await page.goto(`${opsBase}/login`);
  await page.getByLabel("Email or phone").fill(identifier);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
}

test("the ops console requires sign-in and then shows the dashboard", async ({ page }) => {
  await page.goto(`${opsBase}/`);
  await expect(page).toHaveURL(/\/login$/);
  await signIn(page);
  await expect(page.getByText(/Low stock/)).toBeVisible();
});

test("the ops sidebar navigates between sections", async ({ page }) => {
  await signIn(page);
  await page.getByRole("link", { name: "Products" }).click();
  await expect(page).toHaveURL(/\/products$/);

  await page.getByRole("link", { name: "Orders" }).click();
  await expect(page).toHaveURL(/\/orders$/);

  await page.getByRole("link", { name: "Shipping" }).click();
  await expect(page).toHaveURL(/\/shipping$/);
});

test("the ops console rejects a customer account (realm isolation)", async ({ page }) => {
  await signIn(page, "customer@openshop.local", "customer12345");
  await expect(page).toHaveURL(/\/login$/);
});
