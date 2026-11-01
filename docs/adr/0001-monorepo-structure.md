# ADR 0001: Monorepo structure (Turborepo + pnpm, shadcn/Base UI, Go API)

- Status: accepted
- Date: 2026-11-02

## Context

The club asked for an enterprise-grade project structure: Turborepo, shadcn/ui on Base UI primitives,
and room to grow (a second app, shared packages). Contributors change every year, and many are agents
that copy the nearest example. So every task has exactly one way to do it, every package owns one
concern, and CI enforces the boundaries instead of relying on prose.

## Decision

```
binarysust-web/
├── apps/
│   ├── web/                    TanStack Start (static: prerendered public site + client-only /dashboard)
│   │   ├── src/routes/         public routes + dashboard routes (file-based)
│   │   ├── src/features/       one folder per domain: events/, blog/, people/, committee/, talks/,
│   │   │                       sponsors/, media/, portfolio/ — components, hooks and queries for that domain
│   │   ├── src/dashboard/      shell + sections.tsx (THE list of dashboard sections)
│   │   ├── src/content/        snapshot loader (createIsomorphicFn) + slices
│   │   ├── functions/          Cloudflare Pages Function (/api proxy)
│   │   └── e2e/                Playwright tests
│   └── api/                    Go module github.com/BinarySUST/binarysust-web/apps/api (on ekman)
│       ├── cmd/{server,seed,snapshot}
│       ├── internal/<module>/  one ekman Module per domain (sponsors = the template to copy)
│       ├── openapi.json        generated; the API contract
│       └── turbo.json          per-package overrides (dev = go run ./cmd/server, gen, -race tests)
├── packages/
│   ├── ui/                     @binarysust/ui — shadcn/ui (style base-nova = Base UI primitives),
│   │                           design tokens (globals.css), brand <Logo/>, shared hooks. No app logic.
│   ├── api-client/             @binarysust/api-client — types generated from apps/api/openapi.json
│   │                           (openapi-typescript) + the one openapi-fetch client + ApiError. Generated
│   │                           code is committed and CI-checked for drift.
│   ├── eslint-config/          @binarysust/eslint-config — shared flat configs (react, node), import-boundary rules
│   └── typescript-config/      @binarysust/typescript-config — base/react/library tsconfigs
├── infra/
│   ├── docker/compose.yaml     local Postgres + Caddy
│   └── caddy/Caddyfile         binarysust.localhost edge (mirrors production)
├── docs/                       ARCHITECTURE.md, DEPLOY.md, snapshot.md, adr/
├── .github/                    workflows (ci, publish), CODEOWNERS, PR + issue templates, dependabot
├── turbo.json, go.work, pnpm-workspace.yaml, package.json (root scripts only), render.yaml
├── .editorconfig, .nvmrc, .golangci.yml, commitlint + lefthook (Conventional Commits, pre-commit format/lint)
└── README.md, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md, LICENSE
```

### Go in the task graph: Turborepo native Go workspaces

Turborepo 2.11+ discovers Go modules from a root `go.work` (`futureFlags.experimentalGoWorkspaces`), adds
them to the package graph, and maps `build`/`test`/`lint`/`format` to `go build`/`go test ./...`/`go vet`/`go fmt`.
`experimentalTaskCommand` lets `apps/api/turbo.json` override commands: `dev` runs `go run ./cmd/server`
(the module has several `main` packages, so there is no default `dev`), `test` adds `-race`, and `gen`
writes `openapi.json`. The Go module's Turborepo name is derived from its path (`apps/api` → `api`). No JS
package may be named `api`; JS packages are scoped `@binarysust/*`. The feature is experimental. If a
Turborepo release breaks it, the fallback is a thin `package.json` adapter in `apps/api`, with no other
changes.

### Local environment: one `.env`, loaded by dotenv-cli

The root `.env` (copied from `.env.example` on first `pnpm dev`) is the only local configuration. Root
scripts that need it run under `dotenv -e .env -- …` (`dev`, `test`, `db:*`, `snapshot`); the variables
never come from a Go-side loader, so `apps/api` keeps reading plain `os.Getenv` and production (Render)
is unchanged. Turborepo runs in strict env mode: `dev` tasks pass the environment through
(`passThroughEnv`), `api#test` declares `EKMAN_TEST_DATABASE_URL`, and `@binarysust/web#build` hashes
`SITE_URL` and `VITE_*`. `pnpm build` and `pnpm lint` do not read `.env`, so CI and local builds match.

### Versions

Always the latest stable release of every tool and dependency at the time of the change (Go, Node LTS, pnpm,
Turborepo, TanStack, React, Tailwind, shadcn). Dependabot keeps them current. Pin exact versions in lockfiles.

### Rules (enforced, not advisory)

1. **One entry point.** `pnpm dev` (Docker up + `turbo dev` for api and web), `pnpm build`, `pnpm test`,
   `pnpm lint`, `pnpm typecheck`, `pnpm gen`, `pnpm db:migrate|seed|reset`. No Makefile, so there are not
   two ways to run things. Go developers can still run `go test ./...` inside `apps/api`.
2. **Dependency direction:** `apps/* → packages/*`, never `packages → apps`, never `apps/web ↔ apps/api`
   imports (they talk only through `@binarysust/api-client`, which is generated from `openapi.json`).
   ESLint import rules enforce it.
3. **UI primitives live only in `@binarysust/ui`.** Apps compose them. Adding one = `pnpm dlx shadcn add <c>`
   from `apps/web` (the CLI puts it in `packages/ui`). No hand-rolled Button/Dialog/Sheet in apps.
4. **One API client.** Apps never call `fetch('/api/...')` directly; they use `@binarysust/api-client`.
5. **Generated code is committed and verified.** CI regenerates `openapi.json` and the TS types and fails
   on drift.
6. **Feature folders own their domain.** A dashboard section and a public page about events both live
   under `features/events/`; routes stay thin.
7. **Quality gates in CI (turbo, cached):** format, lint (ESLint, golangci-lint), typecheck, unit
   (Vitest, `go test -race`), build, e2e smoke (Playwright against the static build with a mocked API),
   commitlint on PR titles.

## Consequences

- Adding a second app (e.g. a contest portal) = `apps/<name>` reusing `ui` and `api-client`.
- Turbo caches tasks locally and in CI (`actions/cache` on `.turbo`), with no Vercel account needed.
- More files at the root than a minimal repo. Every one has a single purpose, documented in CONTRIBUTING.md.
- Render: `rootDir: apps/api`. Cloudflare Pages: output `apps/web/dist/client`.
