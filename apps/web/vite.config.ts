import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { allPaths } from "./src/content/slices.ts";
import { loadSnapshot } from "./src/content/snapshot.server.ts";

// Every public URL in the snapshot is pre-rendered, linked or not. /dashboard/* and /login are
// client-only: they are served by _shell.html through public/_redirects.
const CLIENT_ONLY = /^\/(dashboard|login)(\/|$)/;

export default defineConfig({
  // One instant shared by the pre-render and the browser bundle, so "upcoming" hydrates without a mismatch.
  define: { __BUILD_TIME__: JSON.stringify(Date.now()) },
  resolve: { alias: { "~": fileURLToPath(new URL("./src", import.meta.url)) } },
  server: {
    // `make dev` puts Caddy on http://binarysust.localhost (port 80) in front of this server (from Docker), so bind
    // every interface and let HMR's websocket go through Caddy's port when VITE_HMR_CLIENT_PORT is set.
    host: true,
    port: 3000,
    allowedHosts: ["localhost", "binarysust.localhost", ".binarysust.localhost", "host.docker.internal"],
    hmr: process.env.VITE_HMR_CLIENT_PORT ? { clientPort: Number(process.env.VITE_HMR_CLIENT_PORT) } : undefined,
    // Without Caddy (plain `pnpm dev`), forward /api and /healthz to the Go API directly.
    proxy: Object.fromEntries(
      ["/api", "/healthz"].map((p) => [p, { target: process.env.API_ORIGIN ?? "http://localhost:8080", changeOrigin: true }]),
    ),
  },
  plugins: [
    tailwindcss(),
    tanstackStart({
      // The SPA shell (dist/client/_shell.html) is rendered at a client-only path, so it does not collide
      // with the pre-rendered "/" page. It serves /login and /dashboard/* via public/_redirects.
      spa: { enabled: true, maskPath: "/login" },
      sitemap: { enabled: false }, // scripts/slices.ts writes sitemap.xml from the snapshot
      prerender: {
        enabled: true,
        crawlLinks: true,
        failOnError: true,
        // /people/x.html rather than /people/x/index.html: Cloudflare Pages serves it at /people/x with no
        // trailing-slash redirect, so links, canonical URLs and the sitemap all match the served URL.
        autoSubfolderIndex: false,
        // Crawled links to client-only routes, static files (rss.xml) and query variants are not pages.
        filter: (page) =>
          page.prerender?.headers?.["X-TSS_SHELL"] === "true" ||
          (!CLIENT_ONLY.test(page.path) && !/[?#]/.test(page.path) && !/\.[a-z0-9]+$/i.test(page.path)),
      },
      pages: [...allPaths(loadSnapshot()).map((path) => ({ path })), { path: "/404", prerender: { outputPath: "/404.html" } }],
    }),
    viteReact(),
  ],
});
