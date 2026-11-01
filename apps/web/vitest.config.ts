import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

// Unit tests only (src/**/*.test.ts). The Playwright smoke tests live in e2e/ and run against the static build.
export default defineConfig({
  define: { __BUILD_TIME__: JSON.stringify(Date.parse("2026-10-30T14:15:00Z")) },
  resolve: { alias: { "~": fileURLToPath(new URL("./src", import.meta.url)) } },
  test: { include: ["src/**/*.test.{ts,tsx}"], environment: "node" },
});
