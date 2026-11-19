# ADR 0002: The API publishes the snapshot to Workers KV

- Status: accepted
- Date: 2026-11-20
- Spec: BinarySUST/binarysust-web#7. Research: `docs/research/publish-snapshot-source.md`

## Context

The public site is built from one `site.Snapshot` (ARCHITECTURE.md, decision 1). Until now, the publish workflow
built it itself: it installed Go, compiled `cmd/snapshot` and connected to Supabase with a read-only role whose
connection string lived in GitHub secrets. That meant:

- **A database credential in CI.** GitHub's hosted-runner IPs can't usefully be allowlisted, so the credential is effectively reachable from the whole internet.
- **The IPv4 pooler.** GitHub runners are IPv4-only, and Supabase's free direct connection is IPv6, so CI had to go through the pooler.
- **Slow runs.** About half a minute of every run went to downloading Go modules and compiling.
- **Repeated database reads.** Every run, including the nightly one and code-only pushes, queried the database again.

The project rule is "$0, no card anywhere" on free plans.

## Decision

The API is the only producer of the snapshot, and Cloudflare Workers KV holds it between the API and CI.

- **`republish.Publisher` (in the API) produces it.** It implements the existing `republish.Notifier`, so the middleware and `republish.Skip()` are unchanged. The middleware only signals it, so an editor's write is never delayed.
- **One background goroutine builds it.** Each run calls `site.Build` and compares the hash with the last published one. If the hash changed, it stores `snapshot:<hash>` (30-day TTL), then `latest` (`{hash, builtAt}`), then fires `repository_dispatch` with `client_payload.hash`. Signals that arrive during a run merge into one next run.
- **Runs are triggered by** content writes, API boot (so a deploy that changes `site.Build` publishes at once), 00:05 Asia/Dhaka (the date rollover, replacing the workflow cron), and `POST /site/republish` (permission `site.publish`, forced).
- **A failure is retried on the next trigger.** A failure is logged and never returned to the editor. The hash only counts as published after the dispatch succeeds.
- **The publish workflow reads the snapshot from KV.** It fetches `snapshot:<hash>`, or `latest` for push and manual runs, with a read-only KV token. It retries for about two minutes, because KV is eventually consistent, and builds as before. Without KV secrets (forks) it builds the fixture and never deploys.
- **`SNAPSHOT_DATABASE_URL` and the `site_reader` role are removed.** `cmd/snapshot` stays for local builds.

## Alternatives

- **R2.** Strongly consistent and free at this size, but Cloudflare requires a payment method before R2 can be enabled.
- **CI pulls from an API endpoint (OIDC-authenticated).** Simpler, but a deploy fails whenever the API is down, and code-only pushes would depend on the API.
- **Keep CI on Supabase and fix the Go cache.** Fixes the speed, keeps the credential.
- **Render at request time.** Reverses decision 1.

## Consequences

- No database credential in GitHub, and publish runs no longer need Go.
- The site is built from what the deployed API produces. A change to `site.Snapshot` must stay readable by the current web code, or the API must ship first so its boot publish refreshes KV.
- The free plan allows 1,000 KV writes a day. Each changed publish uses two.
- One process publishes: running more than one API instance would need coordination.
