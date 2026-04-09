import { defineConfig, mergeConfig } from "vitest/config";

import viteConfig from "./vite.config";

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      css: true,
      environment: "jsdom",
      exclude: ["tests/**"],
      globals: true,
      include: ["src/**/*.{test,spec}.{ts,tsx}"],
      setupFiles: "./src/setupTests.ts",
    },
  }),
);
