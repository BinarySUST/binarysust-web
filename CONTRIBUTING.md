# Contributing to Binary SUST

Thanks for helping. Every task in this repository has one way to do it, and the nearest example is the one to copy.
Read [`docs/adr/0001-monorepo-structure.md`](docs/adr/0001-monorepo-structure.md) once: its rules are enforced by
ESLint and CI, not by review.

## Setup

```sh
pnpm i && pnpm dev   # http://binarysust.localhost
pnpm db:seed         # demo data; sign in at /api/v1/auth/dev/start as an admin
```

You need Node 26 (`.nvmrc`), pnpm 12, Go 1.27 and Docker. `pnpm i` also installs the git hooks (lefthook):
Prettier, gofmt and ESLint run on the files you commit, and commitlint checks the message.

Before opening a pull request: `pnpm lint && pnpm typecheck && pnpm test`. CI runs the same plus `pnpm test:e2e`,
`pnpm build` and the generated-code drift check.

## Add a content type (a module): copy `sponsors`

Sponsors is the smallest complete module, end to end. For a "Projects showcase":

1. **API.** `cp -r apps/api/internal/sponsors apps/api/internal/showcase`. Rename the package and types, rewrite
   `migrations/0001_showcase.sql`, and declare `const Manage rbac.Permission = "showcase.manage"`. Every route states
   its access (`ekman.Public()`, `ekman.SignedIn()` or `ekman.Requires(Manage)`). Never edit a migration that has
   been applied anywhere; add `0002_*.sql`.
2. **Wire it.** In `apps/api/internal/club/club.go`: construct the service, add its `Module()` to `Modules()`, and
   give a role the permission in `Roles`. For public content add a `Showcase` field to `site.Snapshot`, one line in
   `site.Build`, and the field to [`docs/snapshot.md`](docs/snapshot.md).
3. **Contract.** `pnpm gen` regenerates `apps/api/openapi.json` and `packages/api-client/src/schema.d.ts`. Commit
   both; CI fails if they drift.
4. **Web.** `cp -r apps/web/src/features/sponsors apps/web/src/features/showcase` and rename:
   - `queries.ts`: the dashboard's API hooks (query keys, `useShowcase`, `useSaveShowcase`, ...).
   - `dashboard/ShowcaseSection.tsx`: the dashboard section (list + editor form).
   - `components/`: the public page and its parts, reading the snapshot through `content.*` (`src/content`).
   - A route file in `src/routes/` that only declares `createFileRoute({ loader, head, component })`.
5. **Dashboard entry.** One line in `apps/web/src/dashboard/sections.tsx` (see below).

Publishing needs no code: every successful write republishes the site unless the route opts out with
`republish.Skip()`.

## Add a dashboard section

`apps/web/src/dashboard/sections.tsx` is THE list. The sidebar, the `/dashboard/<path>` routes and permission
filtering all derive from it:

```tsx
{ path: "showcase", title: "Showcase", icon: "image", requires: "showcase.manage", component: ShowcaseSection },
```

`requires` is typed from the API's `Permission` union, so a typo fails `pnpm typecheck`. Use `"signed_in"` for
sections every account sees, or an array for "any one of these". The section receives `{ rest }`, the path below it
(`/dashboard/showcase/42` gives `["42"]`). Build forms from the kit in `src/dashboard/ui.tsx` (`Field`, `SaveBar`,
`DeleteButton`, `Table`), which already shows the API's 422 field errors next to their inputs.

## Add a UI component (shadcn/ui on Base UI)

UI primitives live only in `packages/ui` (`@binarysust/ui`). Add one with the shadcn CLI from the app:

```sh
cd apps/web
pnpm dlx shadcn@latest add accordion   # written to packages/ui/src/components/accordion.tsx
```

Then import it as `@binarysust/ui/components/accordion`. Re-skin only through the design tokens in
`packages/ui/src/styles/globals.css` and the component's variant classes, following
[`apps/web/DESIGN.md`](apps/web/DESIGN.md): hairline borders, 4px radius (6px for panels and dialogs), the dark
green primary, no shadows, no glass. ESLint stops apps from importing `@base-ui/react` (or Radix, sonner, ...)
directly, and from hand-rolling a Button, Dialog or Sheet.

## Rules the linters enforce

- `apps/* → packages/*` only. Packages never import apps; `apps/web` never imports `apps/api`.
- The Go API is reached only through `@binarysust/api-client`; no `fetch('/api/...')` and no `openapi-fetch` in apps.
- The site is static: no `createServerFn`, `createServerFileRoute` or server route handlers.
- Generated files are committed and never edited by hand: `apps/api/openapi.json`,
  `packages/api-client/src/schema.d.ts`, `apps/web/src/routeTree.gen.ts`.

## Commits and pull requests

[Conventional Commits 1.0.0](https://www.conventionalcommits.org): `type(scope): description`, in the imperative,
lower case, no trailing period.

```
feat(events): email the waitlist when a seat frees up
fix(api): reject RSVPs after registration closes
docs: explain the yearly handover
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`. Scopes are
optional; use the area you touched (`web`, `api`, `ui`, `api-client`, `events`, `blog`, ...). Breaking changes add
`!` (`feat(api)!: ...`) and a `BREAKING CHANGE:` footer.

Pull requests are squash-merged, so the **PR title** becomes the commit on `main`; CI checks it with commitlint.
Keep pull requests small and focused, fill in the template, and expect a review from `CODEOWNERS`.
