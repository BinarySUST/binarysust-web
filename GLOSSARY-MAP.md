# Glossary map

This repo has more than one bounded context. Each context keeps its own glossary and ADRs; decisions that span contexts live in `docs/adr/`.

| Context | Path       | Glossary               | ADRs                 |
| ------- | ---------- | ---------------------- | -------------------- |
| web     | `apps/web` | `apps/web/GLOSSARY.md` | `apps/web/docs/adr/` |
| api     | `apps/api` | `apps/api/GLOSSARY.md` | `apps/api/docs/adr/` |

`packages/*` are shared libraries, not contexts; their terms belong to whichever context owns the concept.

A context's glossary or ADR folder may not exist yet. `/domain-modeling` creates them lazily.
