# Binary SUST API

The Go backend for the Binary SUST website, built on [ekman](https://github.com/sakib412/ekman).
The design is in [`../docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md); the public data contract is
[`../docs/snapshot.md`](../docs/snapshot.md).

Public pages never call this API. They are pre-rendered from a snapshot of the database
(`cmd/snapshot`). The API serves the dashboard, sign-in, RSVPs and uploads.

## Commands

Run these from the repository root (they read `.env`):

```sh
make up        # Postgres + Caddy (http://binarysust.localhost)
make migrate   # go run ./cmd/server migrate
make seed      # demo data; DEV_ADMIN_EMAIL becomes an admin
make api       # serve on :8080
make snapshot  # write web/.snapshot/site.json from the local database
make gen       # regenerate openapi.json (enums included) and the web's API types
```

Or from `api/`:

```sh
go run ./cmd/server                    # serve (default command)
go run ./cmd/server serve --migrate    # migrate, then serve (what Render runs)
go run ./cmd/server migrate
go run ./cmd/server openapi > openapi.json   # ekman's document + enums (internal/apidoc)
go run ./cmd/seed
go run ./cmd/snapshot -out ../web      # prints the version JSON on stdout

go test ./...                          # unit tests; database tests skip
EKMAN_TEST_DATABASE_URL=postgres://binary:binary@localhost:5433/binarysust?sslmode=disable \
  go test -race ./...                  # everything; each test gets its own schema
```

Sign in locally without Google: open <http://binarysust.localhost/api/v1/auth/dev/start>. It signs
in `DEV_ADMIN_EMAIL` through ekman's normal session flow, and it only exists when `PUBLIC_URL` is a
local origin (`localhost`, `*.localhost`).

## Layout

```
cmd/server      config from env, ekman app, `serve | migrate | openapi`
cmd/seed        demo data (internal/demo)
cmd/snapshot    site.Build -> web/.snapshot/site.json
internal/club   THE module list and THE role table (Roles, Baseline)
internal/<module>/<module>.go, migrations/NNNN_*.sql, *_test.go
  people        persons, handles, terms, positions, committee seats, portfolios
  access        accounts, position roles, grants, site_settings (current term, content_changed_at)
  events        events, speakers, photos, RSVPs, waitlist, check-in, CSV export
  blog          posts, tags, review workflow (Transition)
  talks         talks
  sponsors      sponsors: the smallest module, the one to copy
  media         POST /media (purposes); GitHubBucket for production
  site          Snapshot, Build, /site/status, /site/preview
  apidoc        finishes ekman's OpenAPI document: enums, multipart bodies
  republish     rebuild-on-write middleware, GitHub repository_dispatch
  md            the one markdown renderer (goldmark + bluemonday)
  pg            Postgres helpers (id checks, constraint errors, arrays)
  testapp       integration-test harness (schema per test)
```

## Add a module (copy `sponsors`)

1. `cp -r internal/sponsors internal/showcase`, rename the package and types, and rewrite
   `migrations/0001_showcase.sql`. Never edit a migration that has been applied anywhere; add
   `0002_*.sql` instead (ekman checks checksums).
2. Declare its permission: `const Manage rbac.Permission = "showcase.manage"`, listed in
   `Module().Permissions`. Every route states its access: `ekman.Public()`, `ekman.SignedIn()` or
   `ekman.Requires(Manage)`.
3. In `internal/club/club.go`: construct the service in `New`, add `s.Showcase.Module()` to
   `Modules()` (order = migration order), and give a role the permission in `Roles`. A role or
   route naming an undeclared permission stops the API at boot.
4. For public content, add a `Published(ctx)` read function, a `Showcase []showcase.Public` field to
   `site.Snapshot`, one line in `site.Build`, and the field to `docs/snapshot.md`.
5. `make gen` regenerates `openapi.json` and the web's TypeScript types.

Publishing needs no code: every 2xx non-GET request republishes the site unless the route passes
`republish.Skip()` (use it for writes that change nothing public, like RSVPs).

## Roles and permissions

Roles live in code (`club.Roles`); who holds them is data. A person holds a role through a seat in
the **current** term whose position carries it (`position_roles`), or through a grant that is
permanent or scoped to the current term. Handover is `POST /terms/{id}/make-current`.
Editing a seat in the current term on a position that carries roles needs `roles.manage` as well
as `people.manage`, so editing a roster can never escalate privileges.

| Permission                       | Who                                  |
| -------------------------------- | ------------------------------------ |
| `profile.edit_own`, `blog.write` | every signed-in user (baseline)      |
| `blog.review`                    | blog_editor                          |
| `events.manage`                  | event_manager                        |
| `events.checkin`                 | event_manager, checkin_volunteer     |
| `people.manage`                  | people_manager                       |
| `sponsors.manage`                | sponsor_manager                      |
| `talks.manage`                   | talks_manager                        |
| `media.upload`                   | every manager role (general uploads) |
| `roles.manage`                   | admin only (admin holds everything)  |

`GET /roles` never returns `*`: a role holding every permission has `allPermissions: true` and lists
every declared permission.

## Committee positions

The positions are the club constitution's, created by `internal/people/migrations/0002_positions.sql`
as reference data (they exist in production too). Several people can hold one position in a term.
`group` is how the public committee page groups seats; `lifetime` positions make their holders
lifetime members (`PublicPerson.lifetimeMember`). Seat counts are typical, not enforced.

| Rank          | Position                                | Group      | Seats     | Default roles (seed)           |
| ------------- | --------------------------------------- | ---------- | --------- | ------------------------------ |
| 10            | President                               | leadership | 1         | admin                          |
| 20            | Vice President                          | leadership | 0-2       |                                |
| 30            | General Secretary                       | leadership | 1         | admin                          |
| 40            | Director of Training                    | leadership | 1         | event_manager                  |
| 50            | Project Co-ordinator                    | leadership | 1         |                                |
| 60            | Assistant General Secretary             | assistant  | up to 3   |                                |
| 70            | Organizing Secretary                    | secretary  | 1         | sponsor_manager                |
| 80            | Assistant Organizing Secretary          | assistant  | 2         |                                |
| 90            | Event Management Secretary              | secretary  | 1         | event_manager                  |
| 100           | Assistant Event Management Secretary    | assistant  | 2         | event_manager                  |
| 110           | Publication Secretary                   | secretary  | 1         | blog_editor                    |
| 120           | Assistant Publication Secretary         | assistant  | 1         | blog_editor                    |
| 130           | IT & Design Secretary                   | secretary  | 1         | sponsor_manager, talks_manager |
| 140           | Assistant IT & Design Secretary         | assistant  | 1         |                                |
| 150           | Finance Secretary                       | secretary  | 1         |                                |
| 160           | Office Secretary                        | secretary  | 1         | people_manager                 |
| 170           | Assistant Office Secretary              | assistant  | 1         |                                |
| 180           | Secretary of Internal Affairs           | secretary  | 1         | people_manager                 |
| 190           | Assistant Secretary of Internal Affairs | assistant  | 1         |                                |
| 200           | Senior Executive Member                 | member     | 2         | checkin_volunteer              |
| 210           | Executive Member                        | member     | 5         | checkin_volunteer              |
| 10 (advisory) | Chief Patron (the Vice-Chancellor)      | patron     | 1         |                                |
| 20 (advisory) | Chief Adviser                           | adviser    | 1         |                                |
| 30 (advisory) | Adviser                                 | adviser    | 5 or more |                                |

Lifetime: President, General Secretary, Director of Training. Which roles a position carries is data
(`position_roles`), edited by `roles.manage` in _Committee & roles_. `make seed` fills the defaults above
(`demo.PositionRoles`, added if missing, never removed); a fresh production database starts with none,
so the first admin (a permanent grant) sets them in the dashboard.

## Enums in OpenAPI

ekman v0.1 documents every Go string type as `string`. A string type with `func (T) Enum() []string`
becomes a named enum schema (`people.Group`, `events.Kind`, ...) and every field of that type points at
it; role names come from `club.Roles`. That happens in `server openapi` (`club.OpenAPI`), so `make gen`
and the CI check see it. The document ekman serves at runtime (`GET /api/v1/openapi.json`) does not.

## Environment

| Variable                                                               | Meaning                                                                                          |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `DATABASE_URL`                                                         | Postgres URL                                                                                     |
| `PUBLIC_URL`                                                           | the site's origin (`/api/*` is proxied there); OAuth redirects, cookie security, `/version.json` |
| `EKMAN_SECRET`                                                         | at least 32 bytes; signs the OAuth state cookie                                                  |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`                             | Google sign-in (redirect URI `$PUBLIC_URL/api/v1/auth/google/callback`)                          |
| `DEV_ADMIN_EMAIL`                                                      | local dev sign-in; ignored unless `PUBLIC_URL` is local                                          |
| `MEDIA_DIR`, `MEDIA_BASE_URL`                                          | local uploads directory (relative to `api/` when run from there) and its URL                     |
| `GITHUB_MEDIA_REPOSITORY`, `GITHUB_MEDIA_TOKEN`, `GITHUB_MEDIA_BRANCH` | production uploads: commit to this repo (Contents API); `MEDIA_BASE_URL` is where it is served   |
| `GITHUB_REPOSITORY`, `GITHUB_DISPATCH_TOKEN`                           | send `repository_dispatch` `publish` after content writes; unset logs instead                    |
| `BREVO_API_KEY`, `MAIL_FROM`                                           | RSVP mail through Brevo; an empty key logs mail                                                  |
| `CORS_ORIGINS`                                                         | comma-separated extra origins (normally empty: the site proxies `/api/*`)                        |
| `PORT`                                                                 | listen port, default 8080                                                                        |

## Deploy

`../render.yaml` deploys `Dockerfile` (distroless, static binary) to Render's free plan in
Singapore with a `/healthz` health check. The container runs `server serve --migrate`.

## Known gaps in ekman v0.1 (worked around here)

- `ekmantest.Client.Do` cannot set request headers, so multipart uploads are tested with a hand-built
  request (`internal/media/media_test.go`).
- `ekmantest.New` builds the handler itself, so app middleware (`App.Use`) cannot be added; the
  republish middleware is tested on its own app.
- `db.Tx` takes no `sql.TxOptions`; `site.Build` issues `SET TRANSACTION ... REPEATABLE READ, READ ONLY`
  as its first statement instead.
- `GET /auth/me` returns only `{userId, permissions, roles}`; the account's name and handle come
  from `GET /api/v1/me`.
- No enum support in the OpenAPI generator and no hook to post-process it: `internal/apidoc` rewrites
  the document for `server openapi`, but the runtime `/api/v1/openapi.json` stays enum-free.
- `Handle` cannot read the request, so a typed route with a multipart body is impossible; `POST /media`
  uses `ekman.Handle` plus a route middleware (`withRequest`) that puts the request in the context, and
  `apidoc` documents the body (`Multipart` interface). `Raw` routes get no request or response schema.
