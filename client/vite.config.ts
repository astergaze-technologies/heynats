import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { msw } from "msw/vite";
import path from "path";
import { defineConfig } from "vitest/config";

export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    tailwindcss(),
    mode === "mock" && msw({ mode: "worker-only" }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
  test: {
    environment: "jsdom",
    setupFiles: ["src/mocks/setup.ts"],
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:5000",
        changeOrigin: true,
      },
    },
  },
}));
