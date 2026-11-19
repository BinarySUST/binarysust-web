# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Layout: multi-context

This repo is multi-context. `GLOSSARY-MAP.md` at the root lists the contexts:

```
/
├── GLOSSARY-MAP.md
├── docs/adr/                 ← system-wide decisions
└── apps/
    ├── web/
    │   ├── GLOSSARY.md
    │   └── docs/adr/         ← web-specific decisions
    └── api/
        ├── GLOSSARY.md
        └── docs/adr/         ← api-specific decisions
```

`packages/*` are shared libraries, not contexts.

## Before exploring, read these

- **`GLOSSARY-MAP.md`** at the repo root, then the `GLOSSARY.md` of each context relevant to the topic.
- **`docs/adr/`**: system-wide ADRs that touch the area you're about to work in.
- **`apps/<context>/docs/adr/`**: context-scoped ADRs for the contexts involved.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in the relevant `GLOSSARY.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0001 (monorepo structure), but worth reopening because…_
