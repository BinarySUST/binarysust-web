# Binary SUST web

The public site and the dashboard for Binary SUST. TanStack Start (React) built to **fully static files**
for Cloudflare Pages. Go (`../api`) is the only backend; read [`AGENTS.md`](AGENTS.md) and
[`../../docs/ARCHITECTURE.md`](../../docs/ARCHITECTURE.md) first.

Stack: TanStack Start + Router, React 19, TypeScript (strict), Tailwind CSS v4, shadcn/ui on Base UI
(`@binarysust/ui`), Motion, TanStack Query (dashboard only), `@binarysust/api-client` (openapi-fetch), Vitest,
Playwright, pnpm + Turborepo.

## Develop

The entry point is the monorepo root: `pnpm dev`, then open **http://binarysust.localhost**.
Caddy serves that one origin: `/api/*` and `/healthz` go to the Go API, and everything else to this dev
server on :3000 (`VITE_HMR_CLIENT_PORT=80` from the root `.env` routes HMR through Caddy). Uploads are served
at `http://media.binarysust.localhost`.

Only this app, without Docker or Caddy:

```sh
pnpm --filter @binarysust/web dev   # http://localhost:3000; proxies /api and /healthz to API_ORIGIN (default http://localhost:8080)
```

`pnpm dev` here runs `pnpm slices` first, which writes `public/data/**` from the snapshot. In dev, Start's SPA mode
serves the shell and every page loads its slice from `/data/*.json` in the browser; the disk-reading half
of `content.*` runs only during the build's pre-render. Re-run `pnpm slices` after changing the snapshot.

| Variable               | Where                     | Purpose                                                                                                                        |
| ---------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `VITE_HMR_CLIENT_PORT` | dev                       | HMR websocket port when behind Caddy (80). Unset for plain `pnpm dev`.                                                         |
| `API_ORIGIN`           | dev proxy, Pages Function | Where `/api/*` goes. In production, set it on the Pages project.                                                               |
| `VITE_MEDIA_BASE_URL`  | build                     | Base for media URLs the dashboard builds (default `https://media.binarysust.org`). Snapshot image fields are already absolute. |
| `SITE_URL`             | build                     | Canonical origin for sitemap, RSS and OpenGraph (default `https://binarysust.org`).                                            |

Package scripts (Turborepo runs them from the root: `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm test:e2e`):
`typecheck`, `lint`, `test` (Vitest, `src/**/*.test.ts`), `test:e2e` (Playwright, `e2e/`, against the static build
served by `wrangler pages dev`, with the API mocked), `preview`. API types come from `@binarysust/api-client`
(`pnpm gen` at the root regenerates them from the Go code). Snapshot types in `src/content/types.ts` are aliases
over the generated `site.Snapshot`, narrowed to the enums the Go side validates.

## Layout

```
src/routes/        file-based routes; each only declares createFileRoute({ loader, head, component })
src/features/      one folder per domain (events, blog, people, committee, talks, sponsors, media, portfolio):
                   components/ (public pages and parts), dashboard/ (its dashboard section), queries.ts
src/dashboard/     the dashboard shell, sections.tsx (THE list of sections), Overview, the form kit (ui.tsx)
src/content/       the snapshot loader (content.*), slices and snapshot types
src/components/    site chrome: header, footer, page heads, icons, landing page
e2e/               Playwright smoke tests
functions/         Cloudflare Pages Function (/api proxy)
```

UI primitives come from `@binarysust/ui` (`packages/ui`): add one with `pnpm dlx shadcn@latest add <name>` from
this directory.

## How content flows

```
.snapshot/site.json  (from `go run ./cmd/snapshot`; falls back to the committed .snapshot/fixture.json)
        │
        ├─ pnpm slices (scripts/slices.ts) ─► public/data/**.json, sitemap.xml, rss.xml, _redirects, _routes.json, version.json
        │
        └─ vite build ─► pre-render: every route's loader calls content.* (src/content/index.ts)
                          server half: slice the snapshot from disk with src/content/slices.ts
                          browser half: fetch /data/<slice>.json (same slicing code wrote it)
```

- **One loader path.** Pages call `content.event(slug)` and friends, never the API. The full snapshot is
  never bundled into client JavaScript.
- **Every snapshot URL is pre-rendered**, linked or not: `vite.config.ts` feeds `allPaths(snapshot)` to
  Start's `pages` list, and `crawlLinks` finds the rest.
- **Upcoming vs past** is decided at build time and re-checked in the browser, so a page built days ago
  never lists a finished event as upcoming.
- **Live data** on public pages is limited to the `<Rsvp>` island (`src/features/events/components/Rsvp.tsx`). It pings
  `/healthz` on mount to wake Render, reads availability when the `ekman_hint` cookie says you are signed
  in, and sends you to `/login?next=…` otherwise.
- **The fixture** (`.snapshot/fixture.json`) is synthetic: every person, company, number and sponsor in it
  is made up (the club block matches the API). It has exactly the shape of `site.Snapshot`, so builds without
  a database work; when `cmd/snapshot` has written `.snapshot/site.json` (git-ignored), that file wins.

## Dashboard

Client-only SPA under `/dashboard`, served by `_shell.html`. `src/dashboard/sections.tsx` is the single list
of sections; the sidebar, the `/dashboard/<path>` routes and permission filtering (from `GET /api/v1/auth/me`)
all derive from it. Adding a section is one entry there.

Sections (each shown only when `GET /api/v1/auth/me` grants its `requires`; name, handle and avatar come
from `GET /api/v1/me`):

| Section           | What it does                                                                                                                                                                       |
| ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Overview          | Publish status from `GET /site/status` (`live` or `publishing`); flips right after any content write.                                                                              |
| My portfolio      | `PUT /me/portfolio`, with photo and CV upload.                                                                                                                                     |
| My RSVPs          | Your registrations; cancel.                                                                                                                                                        |
| Events            | List, create, edit (cover, photos, speakers, recap); registrants with CSV download and check-in by ticket.                                                                         |
| Blog              | My posts; for reviewers a review queue. Markdown editor with live preview from `GET /posts/{id}/preview` (drafts autosave); submit, withdraw, request changes, publish, unpublish. |
| People            | Search, create and edit people (affiliation, batch, member since, graduated on, listing).                                                                                          |
| Committee & roles | Terms and seats, positions and the roles each carries, grants, and _Make current_ (the handover).                                                                                  |
| Talks, Sponsors   | List, create, edit (sponsor logo upload).                                                                                                                                          |
| Media             | Upload any allowed purpose and copy the URL.                                                                                                                                       |

Forms show the API's 422 `fields` next to the input they belong to. Uploads are resized in the browser to WebP,
at most 1600 px, before `POST /api/v1/media`.
Any request still pending after 2.5 s shows "Waking the server (up to a minute)…".

## Build and deploy

```sh
pnpm build          # pnpm slices && vite build
```

Output: **`dist/client/`** (the only directory to deploy; `dist/server/` is the pre-render bundle and is
not deployed). It contains one `.html` per public URL (`people/tahmid.html` is served at `/people/tahmid`),
`404.html`, `_shell.html`, `data/`, `sitemap.xml`, `rss.xml`, `_redirects`, `_routes.json`, `version.json`
and hashed assets.

Deploy (GitHub Actions, after `cmd/snapshot` writes `.snapshot/site.json`):

```sh
pnpm exec wrangler pages deploy dist/client --project-name binarysust
```

`wrangler.toml` points `pages_build_output_dir` at `dist/client`. The `functions/` directory ships with it:
`functions/api/[[path]].ts` proxies `/api/*` to `API_ORIGIN` (method, headers, body and cookies upstream;
status, headers and every `Set-Cookie` back), so session cookies stay first-party.

Check a build locally, with the same routing rules as production:

```sh
pnpm exec wrangler pages dev dist/client --binding API_ORIGIN=http://localhost:8080
```
