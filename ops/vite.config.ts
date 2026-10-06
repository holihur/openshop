import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The admin console is served at the root of the internal ops listener, so Vite
// builds it with the default base. In dev it runs on its own port and proxies
// the API to the ops binary.
const target = process.env.VITE_API_TARGET ?? "http://localhost:8081";

export default defineConfig({
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
