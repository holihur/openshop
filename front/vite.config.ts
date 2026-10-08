import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The dev server proxies API and upload traffic to the Go backend, so the SPA
// and API share an origin during development and cookies/CORS are trivial.
const target = process.env.VITE_API_TARGET ?? "http://localhost:8080";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
      "@lib": path.resolve(import.meta.dirname, "../lib"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": { target, changeOrigin: true },
      "/uploads": { target, changeOrigin: true },
      "/robots.txt": { target, changeOrigin: true },
      "/sitemap.xml": { target, changeOrigin: true },
    },
  },
});
