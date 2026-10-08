import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

const here = path.dirname(fileURLToPath(import.meta.url));

// Component and helper tests for the shared layer, run in a DOM environment.
// The alias mirrors the one the apps use, so tests import exactly what the
// application does.
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@lib": here },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./test/setup.ts"],
    include: ["**/*.test.{ts,tsx}"],
    exclude: ["node_modules/**"],
  },
});
