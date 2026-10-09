import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "..");
const ssrBase = process.env.E2E_SSR_BASE_URL ?? "http://localhost:18083";

/**
 * The server-rendered storefront is a separate application, so it gets its own
 * Playwright configuration rather than sharing projects with the SPA suite. It
 * boots the Go API first (the SSR app renders from it) and then the Next server.
 *
 * Set E2E_SSR_BASE_URL to test an already-running deployment.
 */
export default defineConfig({
  testDir: "./ssr",
  timeout: 45_000,
  expect: { timeout: 15_000 },
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : [["list"]],
  use: {
    baseURL: ssrBase,
    navigationTimeout: 45_000,
    actionTimeout: 15_000,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "ssr-chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: process.env.E2E_SSR_BASE_URL
    ? undefined
    : [
        {
          command: "sh scripts/e2e-server.sh server",
          cwd: root,
          url: "http://localhost:18081/healthz",
          reuseExistingServer: !process.env.CI,
          timeout: 240_000,
        },
        {
          command: "sh scripts/e2e-server.sh ssr",
          cwd: root,
          url: "http://localhost:18083/",
          reuseExistingServer: !process.env.CI,
          timeout: 300_000,
        },
      ],
});
