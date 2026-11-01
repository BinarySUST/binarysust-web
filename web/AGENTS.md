# AGENTS.md (web/)

1. This app builds to static files only (Cloudflare Pages); there is no Node server at runtime, so `createServerFn`, `createServerFileRoute` and route `server.handlers` cannot run. ESLint bans them.
2. Go (`../api`) is the only backend. Anything dynamic goes through `/api/*` (proxied by `functions/api/[[path]].ts`) via `src/lib/api/client.ts`.
3. Public pages load content only through `src/lib/content.ts` (snapshot slice: disk at pre-render, `/data/*.json` in the browser). Never call the API for content.
4. New slice: edit `../docs/snapshot.md`, `src/lib/slices.ts` and `content.ts`. New dashboard section: one entry in `src/dashboard/sections.tsx`.
5. Do not hand-edit `src/lib/api/schema.d.ts`, `src/routeTree.gen.ts` or `public/data`; they are generated (`pnpm gen:api`, the router plugin, `pnpm slices`).
