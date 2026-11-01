# AGENTS.md (apps/web)

1. This app builds to static files only (Cloudflare Pages); there is no Node server at runtime, so `createServerFn`, `createServerFileRoute` and route `server.handlers` cannot run. ESLint bans them.
2. Go (`../api`) is the only backend. Anything dynamic goes through `/api/*` (proxied by `functions/api/[[path]].ts`) via `@binarysust/api-client`. Never `fetch('/api/...')` and never import `openapi-fetch` here; ESLint bans both.
3. Public pages load content only through `src/content` (`content.*`: snapshot slice from disk at pre-render, `/data/*.json` in the browser). Never call the API for content.
4. Code lives in `src/features/<domain>/` (`components/`, `dashboard/`, `queries.ts`); `src/routes/*` only declare the route (loader, head, component). Copy `features/sponsors` for a new domain.
5. New slice: edit `../../docs/snapshot.md`, `src/content/slices.ts` and `src/content/index.ts`. New dashboard section: one entry in `src/dashboard/sections.tsx`.
6. UI primitives come from `@binarysust/ui` only. Add one with `pnpm dlx shadcn@latest add <name>` from this directory (the CLI writes it to `packages/ui`). Re-skin through tokens and variants, never by forking a component into the app.
7. Do not hand-edit `packages/api-client/src/schema.d.ts`, `src/routeTree.gen.ts` or `public/data`; they are generated (`pnpm gen` at the root, the router plugin, `pnpm slices`).
