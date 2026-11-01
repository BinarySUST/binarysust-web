import { defineConfig, devices } from "@playwright/test";

// Smoke tests against the static build (dist/client), served by `wrangler pages dev` exactly as Cloudflare Pages
// serves it (.html without extensions, _redirects for the dashboard shell). The API is never reached: every test
// mocks /api/* and /healthz with page.route (e2e/api.ts). `turbo run test:e2e` builds first.
const PORT = 8788;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : [["list"]],
  use: { baseURL: `http://localhost:${PORT}`, trace: "retain-on-failure" },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `pnpm exec wrangler pages dev dist/client --port ${PORT} --binding API_ORIGIN=http://127.0.0.1:9`,
    url: `http://localhost:${PORT}`,
    reuseExistingServer: !process.env.CI,
    env: { WRANGLER_SEND_METRICS: "false" },
    timeout: 60_000,
  },
});
