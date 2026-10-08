import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

const here = path.dirname(fileURLToPath(import.meta.url));

// Page-level tests for the storefront. The aliases mirror vite.config.ts so a
// test imports exactly what the application does.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(here, "./src"),
      "@lib": path.resolve(here, "../lib"),
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: [path.resolve(here, "../lib/test/setup.ts")],
    include: ["src/**/*.test.{ts,tsx}"],
    exclude: ["node_modules/**"],
  },
});
