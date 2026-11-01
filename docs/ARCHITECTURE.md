# Binary SUST website architecture

## Problem

Binary SUST needs:

- A public site: a landing page, people and portfolios, committee history, events with RSVP, a blog, advisor and guest talks, and sponsors.
- One dashboard, where each role (admin, blog editor, event manager, people manager, sponsor manager) sees only its own sections.

Constraints:

- Hosting costs **$0, with no credit card on any service**. The Go API can only run on Render's free tier, which sleeps after 15 idle minutes and takes about a minute to wake.
- The backend is built on [ekman](https://github.com/sakib412/ekman), an open-source, API-only Go framework. It knows nothing about clubs.
- Maintainers are student volunteers who change every year, or agents that copy the nearest file. The design has to make the obvious change the correct one.

## Usage (maintainer's view)

**Add a content type** (for example "Projects showcase"):

1. Copy `api/internal/sponsors/` to `api/internal/showcase/`. It is the smallest complete module. Rename the types, write `migrations/0001_showcase.sql`, and declare `const Manage rbac.Permission = "showcase.manage"`.
2. Add `showcase.New(db).Module()` to the list in `api/internal/club/club.go`. Add a `Showcase []showcase.Public` field to `site.Snapshot`, and add one line to `site.Build`.
3. Run `make gen`. It regenerates `openapi.json` and the TypeScript types, so the new section is typed in the web app.
4. Write the public page and add a dashboard section (one entry in `web/src/dashboard/sections.tsx`).
5. Publishing needs no extra work. Every successful write republishes the site unless its route opts out.

**Give a role a new permission:** add the permission to the role in `club.Roles`. If you name a permission that no module declares, the API refuses to boot, and CI catches it.

**Yearly handover:** in _Committee & roles_:

1. Create term `2026-27`.
2. Fill its committee seats. Each position carries roles; for example, President grants `admin`.
3. Click _Make current_.

Last year's committee loses dashboard access on its next request. Nobody revokes anything by hand.

Call sites:

```go
// api/internal/events/events.go: the access rule sits in the route table, never inside the handler
ekman.Handle(r, "PUT /events/{id}", ekman.Requires(Manage), s.update)
ekman.Handle(r, "POST /events/{slug}/rsvp", ekman.SignedIn(), s.rsvp, republish.Skip())
```

```tsx
// web/src/routes/events/$slug.tsx: pre-rendered at build time from the snapshot; never calls the API
export const Route = createFileRoute("/events/$slug")({
  loader: ({ params }) => content.event(params.slug), // snapshot slice: disk at build, static JSON in the browser
  component: EventPage,
});
function EventPage() {
  const e = Route.useLoaderData();
  return (
    <>
      <article dangerouslySetInnerHTML={{ __html: e.bodyHtml }} />
      <Rsvp slug={e.slug} />
    </>
  ); // live seats come from the API
}
```

## Shape

```
Visitor ─► Cloudflare Pages (static HTML from the snapshot) ─── never touches the API
Member  ─► Pages Function /api/* (same-origin proxy) ─► Render: Go API on ekman ─► Neon Postgres
Write   ─► API commit ─► republish middleware ─► GitHub repository_dispatch
CI      ─► go run ./cmd/snapshot (reads Neon directly, read-only role) ─► vite build (TanStack Start prerender) ─► wrangler pages deploy
```

### Load-bearing decisions

1. **The public site is built from a single snapshot of the database.**
   - `site.Snapshot` is the public contract. It holds every public view, with markdown already rendered to sanitized HTML by the one Go renderer (`internal/md`). The dashboard preview uses the same renderer, so what authors see is exactly what gets published.
   - CI produces the snapshot by reading Neon directly, so a build never waits on the sleeping API.
2. **Republishing is on by default.** After every 2xx response to a non-GET request, middleware updates `site_settings.content_changed_at` and fires a GitHub `repository_dispatch`, unless the route carries `republish.Skip()` (RSVP, check-in, uploads). Forgetting `Skip` costs only a no-op build, because builds whose content hash hasn't changed skip the deploy. GitHub's concurrency group merges bursts of edits into one running build plus one pending. _Is my change live?_ compares `content_changed_at` with the deployed `/version.json`.
3. **Roles follow the committee.**
   - Roles are defined in code (`club.Roles`).
   - A person holds a role through one of two sources:
     - a **seat** in the _current_ term whose position carries it (`position_roles`)
     - an explicit **grant** that is either permanent or scoped to the current term
   - `site_settings.current_term_id` is the only pointer to the current term. Handover is a single-row write.
   - Only `roles.manage` can edit `position_roles` and grants, so editing a roster can't escalate anyone's privileges.
4. **Every route states its access.** `ekman.Handle` takes `Public()`, `SignedIn()` or `Requires(perm)` as a required argument. No route is open unless it says so.

### Data model (one owner per table)

| Module   | Tables                                                               | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| -------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| people   | `persons`, `person_handles`, `terms`, `positions`, `committee_seats` | One person row for life. Status is **derived**, never stored: committee/advisor means a seat in the current term; alumni means `graduated_on <= today`; member means a student with `member_since` set. `affiliation` is student, faculty or guest. `positions` are the constitution's executive and advisory posts, seeded by a migration as reference data, each with a rank, a display `group` (patron, adviser, leadership, secretary, assistant, member) and a `lifetime` flag (President, General Secretary, Director of Training: holders are lifetime members, derived like status). Several people may hold one position in a term. A person with seats, posts or talks cannot be deleted (409; unlist instead). |
| access   | `accounts`, `position_roles`, `role_grants`, `site_settings`         | Any Google user can sign in, which creates an account (needed for RSVP). `accounts.person_id` links to a person when the email matches an invited member. Implements ekman's `GrantSource` and `Directory`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| events   | `events`, `rsvps`, `event_photos`                                    | The registration window and capacity are set together or not at all (CHECK). `UNIQUE(event, account)` makes RSVP idempotent. Admission runs under `FOR UPDATE`. Cancelling promotes the oldest person on the waitlist. Each RSVP gets a random QR `ticket` for check-in. Only drafts, or events with no active RSVPs, can be deleted (409; cancel instead).                                                                                                                                                                                                                                                                                                                                                               |
| blog     | `posts`, `post_tags`                                                 | One pure `Transition(state, action, caller)`: draft → in_review → published ⇄ unpublished. `first_published_at` is never cleared.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| talks    | `talks`                                                              | Speaker is a `person` (advisor or guest). Holds video and slide links.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| sponsors | `sponsors`                                                           | Tier, logo, link and active years. The template to copy for new modules.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| media    | none (files live in the `binarysust-media` repo)                     | One upload endpoint. Each purpose maps to a permission and size/type rules.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| site     | none (reads every module through the modules' own query functions)   | `Build() Snapshot`, `GET /site/status`.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |

## Frontend

**TanStack Start for the whole frontend, built as fully static output** (the club's choice: one React framework for students to learn).

- **Public routes are pre-rendered** at build time (`prerender.enabled`, `crawlLinks`, plus the explicit page list from the snapshot). Each one is real HTML for SEO, and hydrates into the React app.
- **`/dashboard/*` is client-only** (`spa.enabled`). Start emits `_shell.html`, and `_redirects` serves it for every dashboard path.
- **Content has one loader path.** `content.*` is a `createIsomorphicFn`:
  - At pre-render time, its server half reads the snapshot slice from disk.
  - In the browser, its client half fetches the same slice as static JSON (`/data/...json`, written by `cmd/snapshot`).

  The full snapshot is never bundled into client JavaScript, and a page never calls the API for content.

- **No server functions and no server routes.** Go is the only backend, and the output is static, so neither exists at runtime. A lint rule bans `createServerFn` and `createServerFileRoute` (`AGENTS.md` says why).
- **The dashboard is routed from one list** (`src/dashboard/sections.tsx`). The sidebar, routes and permission filtering are derived from it. Data uses TanStack Query against the API.
- **API types are generated** from ekman's OpenAPI (`openapi-typescript` + `openapi-fetch`), including the `Permission` union. A typo in a dashboard permission fails `tsc`.
- **Landing page:** React components with Motion for animation. Heavy visuals load lazily so the first paint stays fast.
- **What you lose compared with Astro:** public pages ship React (~100–150 KB gzipped), and we add RSS, the sitemap and the image pipeline ourselves (generated in `cmd/snapshot`).

## Hosting ($0, no card anywhere)

| Piece          | Service                                                                                     | Why                                                                                                            |
| -------------- | ------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Public site    | Cloudflare Pages (direct upload from Actions) at `binarysust.org`, DNS on Cloudflare free   | Unlimited static bandwidth; Actions builds sidestep the Pages build quota                                      |
| `/api/*` proxy | Pages Functions                                                                             | Same-origin cookies, no CORS                                                                                   |
| Go API         | Render free (Singapore, Docker)                                                             | The only card-free Go host; cold starts only affect signed-in actions                                          |
| Postgres       | Neon free (Singapore)                                                                       | Suspends instead of billing; 0.5 GB is far more than needed                                                    |
| Media          | GitHub repo `binarysust-media`, deployed to its own Pages project at `media.binarysust.org` | Org-owned and versioned. Kept out of the site build, so the site stays under Pages' 20k-files-per-deploy limit |
| Builds         | GitHub Actions (public repo, unlimited minutes)                                             | Snapshot, build and deploy                                                                                     |
| Bot token      | GitHub App installed on the org                                                             | Outlives any one student's personal token                                                                      |
| Sign-in        | Google OAuth (Cloud project without billing)                                                | No passwords, and every SUST student already has a Google account                                              |
| Mail           | Brevo free (300/day, single verified sender)                                                | RSVP confirmation and waitlist promotion. Optional. Resend was rejected because it needs a verified domain     |

## Synthesis decision

Two structurally different designs were produced and judged independently:

- **A (static-first):** a CI-built snapshot served by a CDN.
- **B (dynamic):** Astro rendered on a Cloudflare Worker, with a durable D1 page cache and signed invalidation pushes.

**Base: A.** It scored higher on hosting cost and cold-API resilience, on single source of truth, on operational simplicity and on how easy it is to implement. B's cold-miss path relied on background work that is cut off at about 30 s, while Render takes about 60 s to wake. B also needed D1, a cron Worker, a shared HMAC secret and a custom cache protocol, which is too much for student maintainers.

**Grafts from B:**

- The permission is declared when the route is registered, and `Caller` is passed into the handler. Combined with A's "nothing is open by default", this became the required `Access` argument.
- OpenAPI generation replaced a custom TypeScript generator, which is the better choice for an open-source framework.
- `db.AfterCommit` and `App.Main` (serve, migrate and openapi subcommands).
- Positions as a table, a person's status derived instead of stored, and `position_roles`, so the committee is the role handover.
- RSVP cancellation with waitlist promotion, QR tickets and tighter CHECK constraints.
- Requiring sign-in to RSVP, so guests don't depend on email delivery.

**Fixes for bugs the review found:**

- A sent mail from an address on a domain no one can verify. Mail now goes through Brevo with a single verified sender.
- A copied all media into every deploy, which runs into the 20k-files limit. Media now deploys as its own Pages project.
- A's status endpoint ran a full build on every poll. It now compares a timestamp.
- B's term EXCLUDE constraint rejected back-to-back terms. It is replaced by the current-term pointer.
- B cleared `published_at` when a post was unpublished. `first_published_at` is now never cleared.
- B listed advisors as members. Status now uses `affiliation`.

**Dropped from A:** mail and storage wiring inside the framework setup (now only drivers), the duplicate database URL config, and the custom `tsgen`.

## Tradeoffs accepted

- We accept 2–4 minutes of publish latency in exchange for a public site that is fast, free, and still up when the API is asleep or broken.
- We accept a read-only Neon credential in GitHub Actions secrets in exchange for builds that never wait on Render.
- We accept over-triggered CI runs (republish on by default) in exchange for never shipping a stale site because someone forgot to opt in.
- We accept that `site.Snapshot` is a second list next to `club.Modules()`. It is checked at compile time, and it is the explicit public contract.
- We accept up to a minute of "Waking the server…" on the first sign-in or RSVP after a quiet period. Islands start waking the API as soon as they mount, so by the time the user acts, it's usually awake.
- We accept Google-only sign-in in exchange for storing no passwords and having no email quota on the sign-in path.

- We accept React on every public page (TanStack Start rather than Astro) in exchange for one framework, one router and shared components across the site and dashboard.

## Alternatives considered

- **Astro for the public site plus a TanStack Router dashboard island.** It gives lighter public pages and built-in content tooling, but students would have to learn two page models. Recommended by the design runners; the club chose one framework instead.

- **Dynamic SSR with an edge cache (Next.js on Vercel, or Astro or TanStack Start on Workers).** Every cache miss waits on a sleeping API. It also needs a server runtime under a 10 ms CPU limit, and caching rules leak into every page.
- **Content stored in git (markdown plus a git-based CMS).** Portfolios edited by hundreds of members, role-gated review and committee history are relational data with permissions. Putting them in git hands merge conflicts and access control to students.
- **The build pulls the snapshot from the API.** Every build would first wait about a minute for Render to wake, and would fail when Render is down.

## Open questions and risks

- The club owns `binarysust.org` (Namecheap, expires 2027-09-20). Who keeps renewing it? An expired domain takes the site, sign-in and media down together.
- GitHub disables scheduled workflows on public repos after 60 days with no commits, so the nightly rebuild could silently stop. Should a Cloudflare cron trigger the nightly rebuild instead? Note that the client-side "hide past events" script already covers stale pages.
- Should non-SUST guests be able to RSVP? Today anyone with a Google account can.

## Next implementation step

Build ekman v0.1 (`Handle`, `Access`, `Module`, `db.Migrate`, rbac, auth). Then build `sponsors` end to end as the template to copy: migration, handlers, snapshot field, and the `/sponsors` page.
