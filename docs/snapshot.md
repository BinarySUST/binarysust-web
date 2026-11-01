# Snapshot contract

`site.Snapshot` (Go, `api/internal/site`) is the only thing the public site reads. `cmd/snapshot` writes it
two ways from one value:

- `web/.snapshot/site.json`: the whole snapshot, read from disk at pre-render time.
- `web/public/data/**.json`: per-route slices, fetched by the browser on client-side navigation.

The Go types are the source of truth. They are also exposed through `GET /api/v1/site/preview`, the
dashboard's preview endpoint, so `make gen` emits them into `web/src/lib/api/schema.d.ts`, and the web
app imports the TS types from there. Until the API exists, `web/.snapshot/fixture.json` follows this shape.

All markdown arrives already rendered to sanitized HTML (`*Html` fields). All timestamps are RFC 3339
UTC strings. Image fields hold absolute URLs, or null.

The string unions below are real enums in `openapi.json`: a Go string type with an `Enum()` method
becomes a named enum schema (`api/internal/apidoc`), so `schema.d.ts` already narrows them. Value lists
for code that iterates them live in `web/src/lib/api/enums.ts`.

```ts
type Snapshot = {
  version: { contentChangedAt: string; builtAt: string; hash: string };
  club: { name: string; tagline: string; about: string; email: string;
          socials: { kind: string; url: string }[];
          stats: { label: string; value: number }[] };   // label agrees with value: "1 event run", "4 events run"
  currentTerm: { slug: string; label: string } | null;
  terms: { slug: string; label: string; startsOn: string }[];              // newest first
  people: PublicPerson[];                                                  // listed only
  committees: { term: string; seats: Seat[] }[];                           // every term, newest first
  events: PublicEvent[];                                                   // published, newest first
  posts: PublicPost[];                                                     // published, newest first
  talks: PublicTalk[];
  sponsors: PublicSponsor[];
  redirects: { from: string; to: string }[];                               // old handles/slugs
};

type PersonStatus = "committee" | "advisor" | "member" | "alumni" | "guest";

type PublicPerson = {
  handle: string; name: string; avatarUrl: string | null; headline: string;
  status: PersonStatus;                       // derived at build time
  lifetimeMember: boolean;                    // held a lifetime position (President, General Secretary,
                                              // Director of Training) in any term; independent of status
  department: string; batch: number | null; graduatedOn: string | null;
  workplace: string; jobTitle: string;
  bioHtml: string; skills: string[];
  socials: { kind: "github" | "linkedin" | "x" | "website" | "codeforces" | "leetcode" | "other"; url: string }[];
  cvUrl: string | null;
  projects: { title: string; summary: string; url: string | null; repoUrl: string | null; imageUrl: string | null; tags: string[] }[];
  achievements: { title: string; date: string | null; detail: string }[];
  positions: { term: string; title: string }[];   // committee history
};

type Seat = { handle: string; name: string; avatarUrl: string | null;
              position: string;                           // "Assistant Organizing Secretary"
              category: "executive" | "advisor";          // executive vs advisory committee
              group: SeatGroup;                           // how the committee page groups seats
              rank: number };                             // order within the committee (lower first)
// Seats arrive executive first, then by rank, then name. Several people can share a position in a
// term (assistants, executive members). Positions are the club constitution's
// (api/internal/people/migrations/0002_positions.sql):
//   leadership: President, Vice President, General Secretary, Director of Training, Project Co-ordinator
//   secretary:  Organizing, Event Management, Publication, IT & Design, Finance, Office, Internal Affairs
//   assistant:  the Assistant ... of each; member: Senior Executive Member, Executive Member
//   patron:     Chief Patron (the Vice-Chancellor); adviser: Chief Adviser, Adviser
type SeatGroup = "patron" | "adviser" | "leadership" | "secretary" | "assistant" | "member";

type PublicEvent = {
  slug: string; title: string;
  kind: "workshop" | "contest" | "hackathon" | "seminar" | "meetup";
  status: "published" | "cancelled";
  summary: string; bodyHtml: string; coverUrl: string | null;
  startsAt: string; endsAt: string; venue: string | null; onlineUrl: string | null;
  registration: { opensAt: string; closesAt: string; capacity: number | null } | null;
  recapHtml: string; photos: { url: string; caption: string }[];
  speakers: string[];                         // person handles
};

type PublicPost = {
  slug: string; title: string; excerpt: string; bodyHtml: string; coverUrl: string | null;
  author: string;                             // person handle
  tags: string[]; publishedAt: string; readingMinutes: number;
};

type PublicTalk = {
  slug: string; title: string; speaker: string;   // person handle
  date: string; summaryHtml: string;
  videoUrl: string | null; slidesUrl: string | null; featured: boolean;
};

type PublicSponsor = {
  name: string; tier: "platinum" | "gold" | "silver" | "partner";
  logoUrl: string; url: string; since: number; until: number | null;
};
```

Data slices under `public/data/`: `index.json` (landing: club, currentTerm, committee seats for the current
term, next 3 events, latest 3 posts, featured talk, active sponsors, and `activity`: every published event,
post and talk dated in the 53 weeks before the build, as `{ date, kind: "event" | "post" | "talk" }`, for the
activity graph; it is derived by `slices.ts`, not a snapshot field), `people.json` (directory cards),
`people/<handle>.json`, `committee/<term>.json`, `events.json`, `events/<slug>.json`, `blog.json`,
`blog/<slug>.json`, `talks.json`, `sponsors.json`. The Go side and the web `content` module both
derive slice names from this list. Adding a slice means editing this doc, `site.Slices` and `content.ts`.
