import path from "node:path";
import type { NextConfig } from "next";

// The app lives in a pnpm workspace, so dependencies are hoisted to the
// repository root. Both the bundler and the standalone tracer must be told that
// the workspace is the application's root.
const repoRoot = path.join(import.meta.dirname, "..");

const config: NextConfig = {
  // A standalone server keeps the runtime image small: only the traced files
  // and the production dependencies are copied into it.
  output: "standalone",
  outputFileTracingRoot: repoRoot,
  turbopack: { root: repoRoot },
  // The storefront talks to the Go API. In production this is the internal
  // service address; the browser never needs to know it because every mutating
  // call goes through a route handler on this origin.
};

export default config;
