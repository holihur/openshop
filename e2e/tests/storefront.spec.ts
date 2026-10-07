import { test, expect } from "@playwright/test";

test("home page renders the storefront", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveTitle(/OpenShop/);
  await expect(page.getByRole("link", { name: /OpenShop/i }).first()).toBeVisible();
});

test("browse the catalog and open a product", async ({ page }) => {
  await page.goto("/products");
  const card = page.locator('a[href^="/products/"]').first();
  await expect(card).toBeVisible();
  await card.click();
  await expect(page).toHaveURL(/\/products\/[^/]+$/);
});

test("a customer can sign in from the storefront", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email or phone").fill("customer@openshop.local");
  await page.getByLabel("Password").fill("customer12345");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("link", { name: "Wishlist", exact: true })).toBeVisible();
});

test("the storefront rejects an admin account (realm isolation)", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email or phone").fill("admin@openshop.local");
  await page.getByLabel("Password").fill("admin12345");
  await page.getByRole("button", { name: "Sign in" }).click();
  // Stays on the login page and surfaces an error.
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.locator("p.text-destructive")).toBeVisible();
});
