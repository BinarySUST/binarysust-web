## What and why

<!-- One or two sentences. Link the issue: "Closes #123". -->

## How to check it

<!-- Steps a reviewer can follow, e.g. "pnpm dev, open /dashboard/events, create an event". -->

## Checklist

- [ ] The PR title follows Conventional Commits (`feat(events): add waitlist emails`); it becomes the squash commit.
- [ ] `pnpm lint`, `pnpm typecheck` and `pnpm test` pass locally.
- [ ] API changes: ran `pnpm gen` and committed `apps/api/openapi.json` and `packages/api-client/src/schema.d.ts`.
- [ ] New migrations are new files (`NNNN_*.sql`); no applied migration was edited.
- [ ] UI changes: checked light and dark theme and a 360px-wide screen.
- [ ] Docs updated where behaviour changed (`docs/`, READMEs, `CONTRIBUTING.md`).
