import { test, expect } from "@playwright/test";

// Responsive-layout checks on real mobile viewports (Pixel 5, iPhone 13).
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:18081";

test("the storefront is usable on a mobile viewport", async ({ page }) => {
  await page.goto(`${baseURL}/`);
  // The catalog renders and products are reachable.
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  const firstProduct = page.locator('a[href^="/products/"]').first();
  await expect(firstProduct).toBeVisible();
  await firstProduct.click();
  await expect(page).toHaveURL(/\/products\/[^/]+$/);

  // Nothing overflows the viewport horizontally (no sideways scroll).
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
});

test("the catalog is usable on a mobile viewport", async ({ page }) => {
  await page.goto(`${baseURL}/products`);
  await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
});
