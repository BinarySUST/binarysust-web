# Binary SUST

The website of **Binary SUST**, the student technology club of Shahjalal University of Science and Technology,
Sylhet: a public site (events with RSVP, people and portfolios, committee history, blog, talks, sponsors) and one
dashboard where each committee role sees only its own sections. It costs $0 to run and needs no credit card
anywhere, and it is built so that next year's committee can maintain it by copying the nearest example.

## Architecture

```
                         ┌─────────────────────── GitHub Actions ───────────────────────┐
  content write ──────►  │ go run ./cmd/snapshot ──► vite build (prerender) ──► wrangler │
  (repository_dispatch)  └───────────────────────────────┬──────────────────────────────┘
                                                         ▼
  Visitor ─────────────────────────────►  Cloudflare Pages  (static HTML + /data/*.json)
  Member  ──► /api/* Pages Function ──►  Render: Go API on ekman (apps/api) ──► Neon Postgres
```

- **Public pages never call the API.** They are pre-rendered from one snapshot of the database, so the site stays
  fast and up while the free-tier API sleeps. Only the RSVP button and the dashboard talk to the API.
- **The Go API** (`apps/api`, on [ekman](https://github.com/sakib412/ekman)) owns the data, the roles and the
  OpenAPI contract. **The web app** (`apps/web`, TanStack Start) reaches it only through the generated
  `@binarysust/api-client`.
- **Roles follow the committee:** making a new term current hands over every dashboard permission.

Read [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the why, [`docs/adr/0001-monorepo-structure.md`](docs/adr/0001-monorepo-structure.md)
for the repository rules, and [`docs/DEPLOY.md`](docs/DEPLOY.md) to set up hosting.

## Quickstart

Needs Node 26 (`.nvmrc`), pnpm 12 (`corepack enable` or the standalone installer), Go 1.27 and Docker.

```sh
pnpm i && pnpm dev   # Postgres + Caddy in Docker, then the API (:8080, migrates on start) and the web dev server (:3000)
pnpm db:seed         # optional, once: demo people, events, posts and the default committee roles
```

Open **http://binarysust.localhost**. Caddy serves the site there exactly as production does (`/api/*` to the Go
API, everything else to Vite). Sign in without Google at
<http://binarysust.localhost/api/v1/auth/dev/start> (as `DEV_ADMIN_EMAIL` from `.env`, an admin).

`pnpm dev` creates `.env` from `.env.example` on first run. Every root script that needs it loads it with
dotenv-cli, so the API, the web dev server, tests and the `db:*` scripts read the same file.

## Structure

```
apps/
  web/            TanStack Start: pre-rendered public site + client-only /dashboard (Cloudflare Pages)
  api/            Go API on ekman (Render); openapi.json is its generated contract
packages/
  ui/             @binarysust/ui: shadcn/ui on Base UI, design tokens (globals.css), brand <Logo/>
  api-client/     @binarysust/api-client: types generated from openapi.json, the one API client, ApiError
  eslint-config/  shared ESLint flat configs and the import-boundary rules
  typescript-config/  shared tsconfigs (base, react-library, vite-app)
infra/
  docker/         compose.yaml: local Postgres + Caddy
  caddy/          Caddyfile: the binarysust.localhost edge, shaped like production
docs/             architecture, deploy guide, snapshot contract, ADRs
```

Turborepo runs every task across the JS packages and the Go module (native Go workspace support through
`go.work`), with caching locally and in CI.

## Scripts

Run everything from the repository root. These are the only entry points.

| Script                              | What it does                                                                                   |
| ----------------------------------- | ---------------------------------------------------------------------------------------------- |
| `pnpm dev`                          | Docker (Postgres, Caddy) up, then `turbo dev`: Go API on :8080 and web on :3000, behind Caddy  |
| `pnpm build`                        | Build everything: the API binary (`apps/api/bin/server`) and the static site (`apps/web/dist`) |
| `pnpm test`                         | Go tests with `-race` (database tests use `EKMAN_TEST_DATABASE_URL`) and Vitest unit tests     |
| `pnpm test:e2e`                     | Playwright smoke tests against the static build with the API mocked                            |
| `pnpm lint`                         | ESLint (with the import-boundary rules) and golangci-lint                                      |
| `pnpm typecheck`                    | `tsc` in every TypeScript package                                                              |
| `pnpm format` / `pnpm format:check` | Prettier and `go fmt` / check only                                                             |
| `pnpm gen`                          | Regenerate `apps/api/openapi.json` from the Go code, then the TypeScript API types             |
| `pnpm db:up` / `pnpm db:down`       | Start or stop Postgres and Caddy (data is kept)                                                |
| `pnpm db:migrate` / `pnpm db:seed`  | Apply migrations / migrations plus demo data                                                   |
| `pnpm db:reset`                     | Delete the local database volume, then migrate and seed again                                  |
| `pnpm snapshot`                     | Write `apps/web/.snapshot/site.json` from the local database (the next build uses it)          |

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md): how to add a content type, a dashboard section or a UI component, and the
commit convention. Please read the [code of conduct](CODE_OF_CONDUCT.md). Report security problems privately as
described in [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE) © 2026 Binary SUST
