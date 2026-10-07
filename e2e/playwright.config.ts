import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig, devices } from "@playwright/test";

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, "..");
const baseURL = process.env.E2E_BASE_URL ?? "http://localhost:18081";

// By default Playwright builds the SPAs, migrates, seeds and starts the Go
// server for us. Set E2E_BASE_URL to test an already-running deployment.
export default defineConfig({
  testDir: "./tests",
  timeout: 30_000,
  expect: { timeout: 10_000 },
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : [["list"]],
  use: {
    baseURL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    // Full suite on Chromium, including accessibility scans.
    { name: "chromium", use: { ...devices["Desktop Chrome"] }, testIgnore: /mobile\.spec\.ts/ },
    // Cross-browser coverage of the core flows (a11y is engine-independent).
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, testIgnore: [/mobile\.spec\.ts/, /a11y\.spec\.ts/] },
    { name: "webkit", use: { ...devices["Desktop Safari"] }, testIgnore: [/mobile\.spec\.ts/, /a11y\.spec\.ts/] },
    // Responsive layout on real mobile viewports.
    { name: "mobile-chrome", use: { ...devices["Pixel 5"] }, testMatch: /mobile\.spec\.ts/ },
    { name: "mobile-safari", use: { ...devices["iPhone 13"] }, testMatch: /mobile\.spec\.ts/ },
  ],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : [
        {
          command: "sh scripts/e2e-server.sh server",
          cwd: root,
          url: "http://localhost:18081/healthz",
          reuseExistingServer: !process.env.CI,
          timeout: 180_000,
        },
        {
          command: "sh scripts/e2e-server.sh ops",
          cwd: root,
          url: "http://localhost:18082/healthz",
          reuseExistingServer: !process.env.CI,
          timeout: 180_000,
        },
      ],
});
