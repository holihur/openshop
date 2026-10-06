import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The admin console is served by the Go binary under /ops, so Vite builds it
// with a matching base path. In dev it runs on its own port and proxies the API
// to the backend.
const target = process.env.VITE_API_TARGET ?? "http://localhost:8080";

export default defineConfig({
  base: "/ops/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@lib": path.resolve(__dirname, "../lib"),
    },
  },
  server: {
    port: 5174,
    proxy: {
      "/api": { target, changeOrigin: true },
      "/uploads": { target, changeOrigin: true },
    },
  },
});
