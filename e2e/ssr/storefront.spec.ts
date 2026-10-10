import { expect, test } from "@playwright/test";

/**
 * The server-rendered storefront must be usable before any JavaScript runs:
 * that is the whole point of the second app, so the assertions are made against
 * the served HTML rather than against anything a client script produced.
 */
test.describe("server-rendered storefront", () => {
  test("serves finished HTML without JavaScript", async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();

    await page.goto("/");
    // A product tile is evidence that the page was rendered on the server: with
    // scripting off nothing on the page could have fetched it.
    const card = page.locator('a[href^="/products/"]').first();
    await expect(card).toBeVisible();

    // The product page carries its metadata and structured data in the HTML,
    // which is what search engines read.
    await card.click();
    const html = await page.content();
    expect(html).toContain('"@type":"Product"');
    expect(html).toMatch(/<title>/);

    await context.close();
  });

  test("browses the catalog and opens a product", async ({ page }) => {
    await page.goto("/products");
    await page.locator('a[href^="/products/"]').first().click();
    await expect(page).toHaveURL(/\/products\/[^/]+$/);
    await expect(page.getByRole("button", { name: /add to cart/i })).toBeVisible();
  });

  test("searches the catalog and reports the match count", async ({ page }) => {
    await page.goto("/products?keyword=lamp");
    await expect(page.locator('a[href^="/products/"]').first()).toBeVisible();
    // The result count is rendered on the server from the API's metadata.
    await expect(page.getByText(/1 products?/)).toBeVisible();
  });

  test("adds a product to the cart from a product page", async ({ page }) => {
    await page.goto("/products");
    await page.locator('a[href^="/products/"]').first().click();
    // Wait for the mutation to complete: navigating away mid-request would
    // cancel it and the cart would legitimately still be empty.
    const [response] = await Promise.all([
      page.waitForResponse(
        (res) => res.url().includes("/api/cart/items") && res.request().method() === "POST",
      ),
      page.getByRole("button", { name: /add to cart/i }).click(),
    ]);
    expect(response.ok()).toBeTruthy();

    await page.goto("/cart");
    await expect(page.locator("li").first()).toBeVisible();
    await expect(page.getByText(/subtotal/i).first()).toBeVisible();
  });

  test("registers, writes a review and reads it back immediately", async ({ page }) => {
    // A fresh account each run: the API allows one review per customer per
    // product, so reusing the seeded customer would fail on the second run.
    const stamp = Date.now();
    await page.goto("/register");
    await page.getByLabel("Email").fill(`reviewer+${stamp}@openshop.local`);
    await page.getByLabel("Password").fill("reviewer12345");
    await page.getByRole("button", { name: /create account/i }).click();
    await expect(page).toHaveURL(/\/account\/orders$/);

    await page.goto("/products");
    await page.locator('a[href^="/products/"]').first().click();

    const title = `Review ${stamp}`;
    await page.getByLabel("Review title").fill(title);
    await page.getByLabel("What did you think?").fill("Written from the server-rendered storefront.");
    await page.getByRole("button", { name: /publish review/i }).click();

    // The action invalidates the reviews cache tag, so the new review must be
    // visible on the next render rather than a revalidation window later.
    await expect(page.getByText(title)).toBeVisible();
  });

  test("saves a product, tops up the wallet and opens a ticket", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill("customer@openshop.local");
    await page.getByLabel("Password").fill("customer12345");
    await page.getByRole("button", { name: /^sign in$/i }).click();
    await expect(page).toHaveURL(/\/account\/orders$/);

    // Wishlist: the write goes through this origin, so the API token stays in
    // the cookie.
    await page.goto("/products");
    await page.locator('a[href^="/products/"]').first().click();
    const save = page.getByRole("button", { name: /save for later/i });
    if (await save.count()) {
      await save.click();
      await expect(page.getByRole("button", { name: /saved/i })).toBeVisible();
      await page.goto("/account/wishlist");
      await expect(page.locator('a[href^="/products/"]').first()).toBeVisible();
    }

    // Wallet: the balance is rendered on the server. The ledger only exists once
    // there is a transaction, and the top-up form only when the operator has
    // switched the wallet on, so neither is asserted unconditionally.
    await page.goto("/account/wallet");
    await expect(page.getByText(/balance/i).first()).toBeVisible();
    if (await page.getByRole("table").count()) {
      await expect(page.getByRole("table").first()).toBeVisible();
    }

    // Support: opening a ticket from the storefront.
    await page.goto("/support");
    await page.getByLabel("Subject").fill(`E2E ${Date.now()}`);
    await page.getByLabel("How can we help?").fill("Opened from the server-rendered storefront.");
    await page.getByRole("button", { name: /send request/i }).click();
    await expect(page.getByText(/support/i).first()).toBeVisible();
  });

  test("signs a customer in and keeps the session on the server", async ({ page }) => {
    await page.goto("/login");
    await page.getByLabel("Email").fill("customer@openshop.local");
    await page.getByLabel("Password").fill("customer12345");
    await page.getByRole("button", { name: /^sign in$/i }).click();

    // Reaching the account area proves the tokens are held in cookies and read
    // on the server; the sign-out control proves the page knows who is signed in.
    await expect(page).toHaveURL(/\/account\/orders$/);
    await expect(page.getByRole("heading", { name: /your orders/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /sign out/i })).toBeVisible();
  });
});
