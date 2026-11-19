# Where should the publish pipeline get its content snapshot?

> **Decision (2026-11-20):** the maintainers chose a push model, as in option A, but storing the snapshot
> in **Workers KV** rather than R2, because R2 needs a payment method on the Cloudflare account and the
> project has a "no card anywhere" rule. See docs/adr/0002-snapshot-in-workers-kv.md and
> BinarySUST/binarysust-web#7. The comparison below is kept as written.

Research note, 2026-11-20. Question: how `.github/workflows/publish.yml` should obtain the `site.Snapshot`
that the static site is pre-rendered from. Option B (CI fetches it from an authenticated API endpoint) is the
main candidate. Every external claim links to its primary source. Claims about this repo cite the file.

## TL;DR

- **Recommendation: B, authenticated with GitHub Actions OIDC, not a static bearer token.** The API verifies
  a short-lived JWT from `https://token.actions.githubusercontent.com` and pins `repository_id`, `ref` and
  `workflow_ref`. This removes `SNAPSHOT_DATABASE_URL` from GitHub, and no new long-lived secret replaces it.
- B has two real costs. A build now depends on the API being up. And on a push that changes the snapshot shape,
  the build reads from whatever API version is deployed. Fix the first with curl retries plus a cached
  last-good snapshot. Fix the second by having publish run after `deploy-api` succeeds instead of racing it.
- `docs/ARCHITECTURE.md` lists "The build pulls the snapshot from the API" as a rejected alternative. The
  reason was "wait about a minute for Render to wake". That premise is gone: the API now runs on an always-on VPS
  (`docs/DEPLOY.md` §3). The other reason, "would fail when Render is down", still applies, and the fallback
  below addresses it. Update that section and the "Tradeoffs accepted" bullet when B lands.
- **The `go: downloading` lines found along the way are a cache bug, not a slow network.** The setup-go cache
  entry is 7.29 KiB and empty. The cache key never changes, so it is never re-saved (details under Q6). The
  same entry affects `deploy-api`.
- A (R2) adds a bucket, a token and an upload path to the API with no benefit over B at this scale. C only fixes
  speed and keeps the DB credential in CI. D conflicts with the architecture's core decision.

## Current state (repo facts)

- `publish.yml` runs `go run ./cmd/snapshot -out ../web` with `DATABASE_URL=${{ secrets.SNAPSHOT_DATABASE_URL }}`.
  It reads the printed `version.hash`, compares it with `$SITE_URL/version.json`, and skips the deploy when they
  match. The skip is not used for `push` or `workflow_dispatch`.
  (`.github/workflows/publish.yml`)
- `cmd/snapshot` opens Postgres, calls `site.Build(ctx, svc.Sources, time.Now())` and writes
  `apps/web/.snapshot/site.json`. (`apps/api/cmd/snapshot/main.go`)
- `site.Build` is the same function the dashboard's `GET /api/v1/site/preview` (signed-in) calls.
  It runs in one read-only, repeatable-read transaction, and `Version.Hash` is the sha256 of everything except
  `Version`. (`apps/api/internal/site/site.go`)
- The republish middleware fires `repository_dispatch {event_type: "publish", client_payload: {reason}}` after
  every 2xx non-GET request unless the route opts out with `republish.Skip()`. It uses a fine-grained PAT
  (`GITHUB_DISPATCH_TOKEN`). (`apps/api/internal/republish/republish.go`, `docs/DEPLOY.md` §5)
- The API is a systemd service on an Oracle VM. Cloudflare Tunnel publishes it as `api.binarysust.org`, and the
  site's `/api/*` Pages Function proxies to it. (`docs/DEPLOY.md` §3, `apps/web/functions/api/[[path]].ts`,
  `.github/workflows/deploy-api.yml`)
- Measured on run 44619272155 (2026-11-19): the snapshot step took about 26 s. Module downloads finished in
  under 0.5 s, so nearly all of the 26 s was compiling with a cold build cache. The real snapshot was small
  (2 people, 0 events).

## Options compared

|                         | B: CI GETs snapshot from API | A: API pushes to R2, CI downloads           | C: keep direct DB, cache/prebuild Go      | D: render at request time    |
| ----------------------- | ---------------------------- | ------------------------------------------- | ----------------------------------------- | ---------------------------- |
| DB credential in GitHub | none                         | none                                        | yes (`site_reader`)                       | none                         |
| New CI secret           | none with OIDC               | R2 read token (or public bucket)            | none                                      | n/a                          |
| New API code            | 1 route + OIDC verifier      | R2 client, upload after each write, retries | none                                      | whole SSR path               |
| Build depends on API up | yes (mitigated by fallback)  | no, if the object exists                    | no                                        | every cache miss             |
| Snapshot code version   | the deployed API's           | the deployed API's                          | the commit being built                    | the deployed API's           |
| Extra infra             | none                         | R2 bucket + token                           | none                                      | Workers runtime, cache rules |
| Speed of the step       | one HTTP GET (~1 s)          | one GET                                     | ~26 s today, a few s with a working cache | n/a                          |

### B: authenticated endpoint (main candidate)

- **Removes a credential class.** A Postgres login for Supabase that GitHub runners can reach cannot be
  IP-restricted in any useful way. GitHub says hosted-runner IP ranges are "shared with unrelated services and
  customers" and advises against allowlisting them
  ([GitHub-hosted runners reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)).
  Supabase network restrictions cover "all connection routes, whether pooled or direct"
  ([Supabase network restrictions](https://supabase.com/docs/guides/platform/network-restrictions)). With B,
  only the VPS connects to the DB, so restrictions could be set to the VPS's IP.
- **Keeps one renderer.** It serves the same `site.Build` the dashboard preview already uses
  (`apps/api/internal/site/site.go`).
- **Cost: the snapshot comes from the deployed API, not from `github.sha`.** `publish.yml` also triggers on pushes
  touching `apps/api/internal/site/**` and `apps/api/cmd/snapshot/**`, and `deploy-api.yml` triggers on the same
  push (`apps/api/**`). With B, publish could fetch an old-shape snapshot from an API that is still mid-deploy.
  Fix: publish on `workflow_run` of `deploy-api` with `conclusion == 'success'`. Both workflows must be on
  the default branch, and chaining is capped at three levels
  ([events that trigger workflows: workflow_run](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)).
  Alternatively, `deploy-api` can fire the `publish` dispatch itself after its health check passes.
- **Cost: API down means no fresh snapshot.** See Q5.

### A: API renders and uploads to R2 after each write

- R2 is strongly consistent for read-after-write: "readers will immediately see the latest object globally"
  ([R2 consistency](https://developers.cloudflare.com/r2/reference/consistency/)). It is free for this size
  (10 GB-month storage, 1M Class A and 10M Class B operations per month, no egress fee)
  ([R2 pricing](https://developers.cloudflare.com/r2/pricing/)).
- Tokens can be scoped to one bucket with "Object Read only" or "Object Read & Write". Those object-level
  permissions work only with the S3-compatible API
  ([R2 API tokens](https://developers.cloudflare.com/r2/api/tokens/)). So the API needs an S3 client, and CI
  needs either a read token (a new long-lived secret) or a public bucket.
- The upload would run in the write path. The republish middleware deliberately runs inside the request so a
  sleeping instance cannot drop it (`republish.go`). A full `site.Build` plus upload on every write is more work
  in that path than B, which builds only when CI asks.
- A's benefit is that builds succeed while the API is down. B with a cached fallback gets most of that benefit
  without new infrastructure. A also has the same "snapshot from the deployed API version" issue as B.

### C: keep direct DB access, fix the Go cache

- This keeps the read-only Postgres credential in GitHub. That is the tradeoff `docs/ARCHITECTURE.md` already
  accepts ("We accept a read-only Supabase credential in GitHub Actions secrets").
- Its only gain is speed, and fixing the broken cache (Q6) gets that with no design change. Worth doing in any
  case, because `deploy-api` shares the same broken cache entry.
- C is the only option that builds the snapshot with the code at `github.sha`.

### D: render at request time

- Pages Functions can read R2 (`context.env.BUCKET.get(...)`)
  ([Pages Functions bindings](https://developers.cloudflare.com/pages/functions/bindings/)). Free Workers allow
  100,000 requests per day and 10 ms CPU per request
  ([Workers limits](https://developers.cloudflare.com/workers/platform/limits/)).
- This reverses the architecture's central decision: static pre-rendered pages "still up when the API is down".
  "Dynamic SSR with an edge cache" is already a rejected alternative (`docs/ARCHITECTURE.md`). Not pursued.

## Q1. GitHub Actions secrets vs OIDC

**Secrets.** Secrets are not passed to workflows triggered from forks (`GITHUB_TOKEN` is the exception), and
values over 48 KB need a workaround
([Using secrets in GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/using-secrets-in-github-actions)).
A static bearer token for B would be a second long-lived shared secret, living in GitHub and in
`/etc/binarysust-api.env`. It would need rotation in the yearly handover, like `GITHUB_DISPATCH_TOKEN`
(`docs/DEPLOY.md`, handover checklist).

**OIDC.** GitHub's stated benefits: "You won't need to duplicate your cloud credentials as long-lived GitHub
secrets", and the provider-issued token "is only valid for a single job, and then automatically expires"
([OpenID Connect concepts](https://docs.github.com/en/actions/concepts/security/openid-connect)).

How to get a token in a job:

- The job needs `permissions: id-token: write`. That permission "only enables fetching and setting the OIDC
  token; it does not grant write access to other resources"
  ([OIDC reference](https://docs.github.com/en/actions/reference/security/oidc)).
- The runner exposes `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN`. A custom audience is
  passed as `&audience=...` on that URL ([OIDC reference](https://docs.github.com/en/actions/reference/security/oidc)).
  The response's `.value` field is the JWT
  ([OIDC in cloud providers](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-cloud-providers)).

How the API verifies it:

1. **Issuer** must equal `https://token.actions.githubusercontent.com`. Discovery is at
   `/.well-known/openid-configuration`. It gives `jwks_uri = https://token.actions.githubusercontent.com/.well-known/jwks`
   and the only signing algorithm, `RS256`
   ([discovery document](https://token.actions.githubusercontent.com/.well-known/openid-configuration)).
2. **Signature** is checked against the JWKS, and the algorithm is pinned. RFC 8725 §3.1 says libraries "MUST
   enable the caller to specify a supported set of algorithms", and §3.8 says the keys must belong to the issuer
   ([RFC 8725](https://www.rfc-editor.org/rfc/rfc8725)).
3. **Audience.** The default `aud` is the repository owner's URL
   ([OIDC reference](https://docs.github.com/en/actions/reference/security/oidc)). Request a custom one such as
   `binarysust-api` and require it. RFC 8725 §3.9 says recipients must reject a missing or mismatched `aud`.
4. **Expiry** (`exp`, `nbf`, `iat`). GitHub does not document a fixed lifetime. Its example token spans 300 s
   ([OpenID Connect concepts](https://docs.github.com/en/actions/concepts/security/openid-connect)).
5. **Repository claims.** GitHub says to "define at least one trust condition" so untrusted repositories can't
   use your trust, and recommends `repository_id` / `repository_owner_id` because they survive renames and
   name recycling ([OIDC reference](https://docs.github.com/en/actions/reference/security/oidc)). For this repo,
   require:
   - `repository_id` = BinarySUST/binarysust-web's numeric ID (and/or `repository_owner_id`)
   - `ref` = `refs/heads/main`. `repository_dispatch` always runs on the default branch with
     `GITHUB_REF` = the default branch
     ([events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows))
   - `workflow_ref` = `BinarySUST/binarysust-web/.github/workflows/publish.yml@refs/heads/main`. The format is
     shown in the [OIDC reference](https://docs.github.com/en/actions/reference/security/oidc).
   - optionally `event_name` in `{repository_dispatch, push, schedule, workflow_dispatch, workflow_run}`

In Go, `github.com/coreos/go-oidc/v3/oidc` does this. `NewProvider` performs discovery and rejects an
issuer mismatch, and `Provider.Verifier(&oidc.Config{ClientID: "binarysust-api"})` checks signature, issuer,
audience and expiry. Its `RemoteKeySet` caches keys and re-fetches when it sees an unknown key ID. `IDToken.Claims`
decodes the repository claims ([go-oidc docs](https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc)). It would be
the API's first JWT dependency (`apps/api/go.mod` has none today).

**Is OIDC better than a static bearer token for B? Yes, with one caveat.** There is nothing to store, rotate or
hand over. A leaked token is useless within minutes. The claims prove the request came from `publish.yml` on
`main` of this repo. The caveat: anyone who can merge a workflow change to `main` can mint a valid token. The same
is true of any secret available to that workflow, so OIDC is no weaker.

The snapshot holds only public data (published content, listed people; `docs/snapshot.md`). So auth on this
endpoint mainly prevents abuse: it stops anyone from triggering a full-database repeatable-read `Build` at will.
It is not mainly about confidentiality.

## Q2. repository_dispatch payload and concurrency

- **Limits:** `client_payload` allows at most 10 top-level properties and "must be less than 64KB", and
  `event_type` is at most 100 characters
  ([REST: create a repository dispatch event](https://docs.github.com/en/rest/repos/repos#create-a-repository-dispatch-event)).
  The events reference gives the payload limit as 65,535 characters
  ([events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)).
  A fine-grained PAT needs **Contents: write**
  ([fine-grained PAT permissions](https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens)),
  which matches `docs/DEPLOY.md`.
- **Could the snapshot ride in the payload?** No. The committed fixture is already 35,677 bytes
  (`apps/web/.snapshot/fixture.json`). A real club with hundreds of people, bios and posts will pass 64 KB.
  It would also force the API to run a full `Build` inside every write request.
- **Could the hash ride in the payload?** Possible, but of little use. The middleware does not build a snapshot
  (`republish.go`), so it would have to compute one per write. More importantly, under concurrency the pending run
  is replaced by the newest one, so a payload describes one write, not "what to build". The run should always
  fetch the current state when it starts. The current design already relies on that ("the pending one sees all
  edits", `publish.yml`).
- **Concurrency semantics:** with `cancel-in-progress: false`, there is "at most one running job or workflow
  in a concurrency group at any time". Under the default `queue: single`, "any existing `pending` job or workflow
  run in the same group is canceled and replaced". `queue: max` allows up to 100 pending runs
  ([workflow syntax: concurrency](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)).
  With B this still holds: the surviving pending run fetches the snapshot at its own start, after every write
  that triggered the runs it replaced. Keep `queue: single`; `max` would only cause redundant builds.
- **Default branch:** `repository_dispatch` runs only if the workflow file is on the default branch, with
  `GITHUB_SHA` = the default branch's last commit
  ([events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)).
  This is why pinning `ref` = `refs/heads/main` in the OIDC check is safe.
- **Scheduled trigger:** scheduled runs may be delayed or dropped under load, and "are automatically disabled"
  after 60 days of no repository activity on public repos
  ([events that trigger workflows](https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows)).
  `docs/ARCHITECTURE.md` already lists this as an open risk. B doesn't change it.

## Q3. Supabase from CI

- Supabase lists GitHub Actions among services that "only accept IPv4 connections"
  ([Supabase IPv4/IPv6 compatibility](https://supabase.com/docs/guides/troubleshooting/supabase--your-network-ipv4-and-ipv6-compatibility-cHe3BP)).
  The direct connection is IPv6 on Free/Pro unless you buy the IPv4 add-on. The shared pooler in session mode
  (port 5432) is reachable over IPv4 on every plan and supports prepared statements. Transaction mode (6543)
  suits many short-lived connections but has no prepared statements
  ([Connecting to Postgres](https://supabase.com/docs/guides/database/connecting-to-postgres)). The repo's choice
  of the session pooler for `SNAPSHOT_DATABASE_URL` (`docs/DEPLOY.md` §1.3) is the documented route for an
  IPv4-only client that uses pgx's prepared statements.
- Read-only roles: Supabase says never to hand out the `postgres` password and to "create a new user for every
  service". Example pooler usernames take the form `role.projectref`
  ([Postgres roles](https://supabase.com/docs/guides/database/postgres/roles)). That matches the
  `site_reader.<project-ref>` setup.
- **Risks of the current approach:**
  1. A Postgres login reachable from the whole internet, because hosted-runner IPs can't be usefully allowlisted
     (Q1 sources). It is read-only, but it can still read every column of every table in `public`, including
     emails and grants, not just public fields. `GRANT SELECT ON ALL TABLES` (`docs/DEPLOY.md` §1.2).
  2. A second credential to rotate. A third, if you count the API's own.
  3. Builds use whatever `cmd/snapshot` is at `github.sha`, against a schema migrated by whatever API is
     deployed. The same version-skew issue as B, in the other direction.
  4. It prevents turning on Supabase network restrictions, which with B could be locked to the VPS.

## Q4. Cloudflare: R2, Pages, deploy hooks, Workers

- **Pages Direct Upload is permanent:** "If you choose Direct Upload, you cannot switch to Git integration
  later". Wrangler uploads allow up to 20,000 files of up to 25 MiB each
  ([Direct Upload](https://developers.cloudflare.com/pages/get-started/direct-upload/)). The free plan allows
  500 builds per month, one concurrent build, and a 20-minute timeout. Those build limits are written for
  Git-integrated builds ([Pages limits](https://developers.cloudflare.com/pages/platform/limits/)).
- **Deploy hooks** are URLs that start a new _build_ on POST
  ([deploy hooks](https://developers.cloudflare.com/pages/configuration/deploy-hooks/)). Setting one up requires
  a "branch to build", that is, a Git-connected project. The docs do not say they work for Direct Upload
  projects, which is what `binarysust` is (`publish.yml` uses `wrangler pages deploy`). Treat them as
  unavailable. Even if they worked, they would move the build onto Cloudflare's 500/month quota. That quota is
  what `docs/ARCHITECTURE.md` says Actions builds sidestep.
- **R2** for option A: strongly consistent, free at this size, and bucket-scoped tokens only on the S3 API
  (sources under option A).
- **Workers Cron Triggers** could replace the GitHub `schedule` trigger that may go dormant. They run in UTC
  ([cron triggers](https://developers.cloudflare.com/workers/configuration/cron-triggers/)), the free plan
  allows 5 per account and 10 ms CPU per run
  ([Workers limits](https://developers.cloudflare.com/workers/platform/limits/)). A cron Worker would just POST
  the `publish` dispatch. That needs a GitHub token in the Worker, which is out of scope here.
- **Nothing here replaces B's core.** Cloudflare cannot run the Go `site.Build`. The snapshot must come from the
  API (B, A) or from the DB (C).

## Q5. B's failure mode: API down

The API sits behind Cloudflare Tunnel. When no healthy `cloudflared` is connected, Cloudflare returns error
1033 ([error 1033](https://developers.cloudflare.com/support/troubleshooting/http-status-codes/cloudflare-1xxx-errors/error-1033/)).
A systemd restart during `deploy-api` gives a short window of 502/connection errors.

Standard mitigations, in order:

1. **Retry with exponential backoff.** curl's `--retry N` retries "transient" errors (timeouts, HTTP 408, 429,
   500, 502, 503, 504). It waits 1 s, then doubles the wait up to 10 minutes. `--retry-max-time` caps the total,
   and `--retry-all-errors` widens the set, described as the "sledgehammer" (`man curl`, curl 8.x;
   [curl manpage](https://curl.se/docs/manpage.html)). A Tunnel error status is not in the transient list, so either
   use `--retry-all-errors` (safe for an idempotent GET to a file) or loop in shell. Something like
   `--retry 5 --retry-all-errors --retry-max-time 120 --max-time 30` covers a restart window inside the job's
   15-minute timeout.
2. **Fall back to the last good snapshot, by trigger.**
   - `repository_dispatch` / `schedule`: if the API stays down, **skip the deploy with a warning**. The live site
     is already the newest content anyone could publish. The next write or nightly run catches up, which is the
     same "catches up" contract the middleware already uses for failed dispatches (`republish.go`).
     Exception: the nightly run exists for date roll-overs (`publish.yml`). Rebuilding from the cached snapshot
     still moves `builtAt` and date-derived pages forward, so use the fallback for `schedule` too.
   - `push` / `workflow_dispatch` (code changes): build from the **cached last-good snapshot**, so a web-only
     fix can still ship during an API outage.
   - Cache it with `actions/cache/save` under key `site-snapshot-<hash>`, and restore with
     `restore-keys: site-snapshot-`. With several partial matches "the action returns the most recently created
     cache" ([dependency caching reference](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching)).
     Entries are immutable, evicted after 7 days unused, and the repository total is 10 GB (same source). The
     nightly run keeps the entry warm.
   - Last resort if the cache is empty: keep today's fixture path (`hash=fixture`), but **refuse to deploy** it
     to production. Deploying the fixture over real content would be worse than not deploying.
3. **Don't let the hash check mask an outage.** The current `curl ... | jq -r .hash || echo none` treats a
   missing live `/version.json` as "changed", which is the right direction. Keep that. A failed snapshot fetch must
   not produce an empty hash that compares equal to anything.

## Q6. Caching for option C, and why a fresh run says `go: downloading`

How `actions/setup-go` caches (source at `main`, commit `d8cbb80`):

- It caches the directories printed by `go env GOMODCACHE` and `go env GOCACHE`
  ([package-managers.ts](https://github.com/actions/setup-go/blob/main/src/package-managers.ts)).
- The key is `setup-go-${RUNNER_OS}-${arch}-${ImageOS}-go-${versionSpec}-${hashFiles(cache-dependency-path)}`.
  The default dependency file is `go.mod`, which the README says has been the default since v6. The restore is an
  **exact-key restore with no restore-keys**
  ([cache-restore.ts](https://github.com/actions/setup-go/blob/main/src/cache-restore.ts),
  [README](https://github.com/actions/setup-go)).
- The save runs in the post step only on `success()`
  ([action.yml](https://github.com/actions/setup-go/blob/main/action.yml)). It **does not save when the primary
  key was hit** ("Cache hit occurred on the primary key ..., not saving cache")
  ([cache-save.ts](https://github.com/actions/setup-go/blob/main/src/cache-save.ts)).
- Cache entries can't be modified, only replaced under a new key
  ([dependency caching reference](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching)).
- The Go command "will automatically download modules as needed during ordinary execution" into the module
  cache, and `go mod download` pre-fills it ([Go modules reference](https://go.dev/ref/mod#go-mod-download)).
  `go run` "compiles and runs the named main Go package" ([cmd/go](https://pkg.go.dev/cmd/go)), so a cold
  `GOCACHE` means a full compile.

**Why this repo's runs show `go: downloading` (verified with `gh cache list` and run logs):**

1. Run 41347866158 (publish, push, 2026-11-01) had no `SNAPSHOT_DATABASE_URL` yet, so it took the fixture
   branch and never ran a Go command. Its setup-go post step warned
   `Cache folder path is retrieved but doesn't exist on disk: /home/runner/go/pkg/mod`, then saved an almost
   empty entry: `setup-go-Linux-x64-ubuntu26-go-1.27.0-baa8afe3...`, **7.29 KiB**.
2. Every later run computes the same key (same Go version, same `apps/api/go.sum`). It gets an exact hit on that
   empty entry ("Cache Size: ~0 MB (7474 B)"), so it never saves again. Every run then downloads all modules and
   compiles from scratch (about 26 s in run 44619272155).
3. `deploy-api.yml` uses the same `go-version-file` and `cache-dependency-path`, so it hits the same empty
   entry.

Fixes for C, or for `deploy-api` under any option: delete that one cache entry once with `gh cache delete`.
Better, make the key depend on something that guarantees content, such as a per-workflow suffix via a different
`cache-dependency-path` glob, or a separate `actions/cache` step keyed on
`hashFiles('apps/api/**/*.go', 'apps/api/go.sum')` that caches a prebuilt `snapshot` binary. Running
`go mod download` before the fixture early-exit would also stop an empty entry being saved.

## Recommendation and outline for B

**Do B with OIDC, a retry-plus-cache fallback, and publish ordered after `deploy-api`.** Separately, delete the
empty setup-go cache entry, because it slows `deploy-api` either way.

### Endpoint

- `GET /api/v1/site/snapshot`, in the `site` module next to `/site/preview`. It returns `site.Snapshot` exactly as
  `cmd/snapshot` writes it, built by the same `site.Build(ctx, src, now)`.
- Response headers:
  - `ETag: "<version.hash>"`
  - `Cache-Control: no-store`
  - `X-Api-Commit: <build sha>`, so CI can check it serves the code it expects.
- Optional `If-None-Match` → `304`. CI could then skip the build without downloading, though the body is small
  enough that this is a nicety.
- Route option `republish.Skip()` is irrelevant because it is a GET (`ShouldPublish` ignores GETs).
- CI should call `https://api.binarysust.org` directly, not `www.binarysust.org/api`. That avoids a hop through
  the Pages Function, and the proxy forwards the `Authorization` header anyway.

### Auth

- Add a middleware/access rule, e.g. `ekman.Requires(GitHubOIDC)`, backed by `go-oidc`:
  - issuer `https://token.actions.githubusercontent.com`
  - audience `binarysust-api`
  - RS256 only
  - `repository_id` = this repo's ID
  - `ref` = `refs/heads/main`
  - `workflow_ref` = `BinarySUST/binarysust-web/.github/workflows/publish.yml@refs/heads/main`
- Config:
  - `SNAPSHOT_OIDC_AUDIENCE`, `SNAPSHOT_OIDC_REPOSITORY_ID` in `/etc/binarysust-api.env`. These are not secrets.
  - If unset, the route returns 404 or 503, so local development is unaffected.
- Log `run_id` and `event_name` from the claims for an audit trail.

### Workflow step (replaces "Snapshot the database")

```yaml
permissions:
  contents: read
  id-token: write

on:
  repository_dispatch: { types: [publish] }
  workflow_run: { workflows: [deploy-api], types: [completed] } # replaces apps/api/** push paths
  push: { branches: [main], paths: [apps/web/**, packages/**, pnpm-lock.yaml, .github/workflows/publish.yml] }
  schedule: [{ cron: "5 18 * * *" }]
  workflow_dispatch:

jobs:
  publish:
    if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'
    steps:
      - uses: actions/cache/restore@v5 # last-good snapshot, newest first
        with: { path: apps/web/.snapshot/site.json, key: site-snapshot-none, restore-keys: site-snapshot- }
      - name: Fetch the snapshot
        id: snapshot
        run: |
          jwt=$(curl -fsS -H "Authorization: Bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
                "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=binarysust-api" | jq -r .value)
          if curl -fsS --retry 5 --retry-all-errors --retry-max-time 120 --max-time 30 \
               -H "Authorization: Bearer $jwt" -o "$RUNNER_TEMP/site.json" \
               https://api.binarysust.org/api/v1/site/snapshot; then
            mv "$RUNNER_TEMP/site.json" apps/web/.snapshot/site.json
            echo "source=api" >> "$GITHUB_OUTPUT"
          elif [ -f apps/web/.snapshot/site.json ] && [ "${{ github.event_name }}" != repository_dispatch ]; then
            echo "::warning::API unreachable; building from the cached last-good snapshot."
            echo "source=cache" >> "$GITHUB_OUTPUT"
          else
            echo "::warning::API unreachable; nothing new to publish, skipping."
            echo "source=none" >> "$GITHUB_OUTPUT"; exit 0
          fi
          echo "hash=$(jq -r .version.hash apps/web/.snapshot/site.json)" >> "$GITHUB_OUTPUT"
      - if: steps.snapshot.outputs.source == 'api'
        uses: actions/cache/save@v5
        with: { path: apps/web/.snapshot/site.json, key: "site-snapshot-${{ steps.snapshot.outputs.hash }}" }
      # existing "Skip if the live site already has this content", build and deploy steps,
      # each additionally gated on steps.snapshot.outputs.source != 'none'
```

The last-resort fixture build stays for forks and local use, but never deploys to production. The setup-go step
goes away from `publish.yml`.

### Follow-ups when implementing

- `deploy-api`: either rely on `workflow_run`, or fire the `publish` dispatch after its health check. Pick one so a
  site-contract change publishes exactly once, against the new API.
- Remove the `SNAPSHOT_DATABASE_URL` secret and `DROP ROLE site_reader` in Supabase. Consider Supabase network
  restrictions limited to the VPS's IP.
- Update `docs/ARCHITECTURE.md`:
  - the CI flow line
  - point 1 ("CI produces the snapshot by reading Supabase directly")
  - the "read-only Supabase credential" tradeoff
  - the "build pulls the snapshot from the API" rejected alternative
- Also update `docs/DEPLOY.md` §1.2–1.3 and §5, and the comment in `cmd/snapshot/main.go`. `cmd/snapshot` can stay
  as a local tool.
- Optionally record the decision as an ADR in `docs/adr/`.
