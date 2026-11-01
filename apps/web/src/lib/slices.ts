// One slicing implementation, two callers:
//   - scripts/slices.ts writes every slice to public/data/<name>.json at build time;
//   - content.ts (server half) slices the in-memory snapshot during pre-render.
// Slice names follow docs/snapshot.md. Adding a slice means editing that doc, site.Slices (Go) and this file.
import type {
  Club,
  PersonStatus,
  PublicEvent,
  PublicPerson,
  PublicPost,
  PublicSponsor,
  PublicTalk,
  Seat,
  Snapshot,
  Term,
} from "./types.ts";

export type PersonRef = Pick<PublicPerson, "handle" | "name" | "avatarUrl">;

export type PersonCard = PersonRef &
  Pick<PublicPerson, "headline" | "status" | "department" | "batch" | "workplace" | "jobTitle" | "skills">;

export type EventCard = Omit<PublicEvent, "bodyHtml" | "recapHtml" | "photos" | "speakers">;

export type PostCard = Omit<PublicPost, "bodyHtml" | "author"> & { author: PersonRef | null };

export type TalkWithSpeaker = PublicTalk & { speakerRef: PersonCard | null };

export type IndexSlice = {
  club: Club;
  currentTerm: Snapshot["currentTerm"];
  committee: Seat[];
  events: EventCard[]; // next 3 by startsAt, as of build time
  posts: PostCard[]; // latest 3
  talk: TalkWithSpeaker | null; // featured
  sponsors: PublicSponsor[]; // active this year
  activity: Activity[]; // every event, post and talk in the 53 weeks before the build
  version: Snapshot["version"];
};

/** One dated thing the club did, for the landing page's activity graph. */
export type Activity = { date: string; kind: "event" | "post" | "talk" };

export type CommitteeSlice = { term: Term; isCurrent: boolean; terms: Term[]; seats: Seat[] };
export type EventSlice = { event: PublicEvent; speakers: PersonCard[] };
export type PostSlice = { post: PublicPost; author: PersonCard | null; more: PostCard[] };

const ref = (p: PublicPerson): PersonRef => ({ handle: p.handle, name: p.name, avatarUrl: p.avatarUrl });

const card = (p: PublicPerson): PersonCard => ({
  ...ref(p),
  headline: p.headline,
  status: p.status,
  department: p.department,
  batch: p.batch,
  workplace: p.workplace,
  jobTitle: p.jobTitle,
  skills: p.skills,
});

const eventCard = ({ bodyHtml: _b, recapHtml: _r, photos: _p, speakers: _s, ...rest }: PublicEvent): EventCard => rest;

function personIndex(s: Snapshot) {
  return new Map(s.people.map((p) => [p.handle, p]));
}

function postCard(s: Snapshot, { bodyHtml: _b, author, ...rest }: PublicPost): PostCard {
  const a = personIndex(s).get(author);
  return { ...rest, author: a ? ref(a) : null };
}

const bySeat = (a: Seat, b: Seat) => (a.category === b.category ? a.rank - b.rank : a.category === "executive" ? -1 : 1);

export const STATUS_ORDER: PersonStatus[] = ["committee", "advisor", "member", "alumni", "guest"];

export const slices = {
  index(s: Snapshot): IndexSlice {
    const now = Date.parse(s.version.builtAt);
    const upcoming = s.events
      .filter((e) => e.status === "published" && Date.parse(e.endsAt) >= now)
      .sort((a, b) => Date.parse(a.startsAt) - Date.parse(b.startsAt))
      .slice(0, 3)
      .map(eventCard);
    const year = new Date(now).getUTCFullYear();
    const featured = s.talks.find((t) => t.featured) ?? s.talks[0] ?? null;
    return {
      club: s.club,
      currentTerm: s.currentTerm,
      committee: s.currentTerm ? (slices.committee(s, s.currentTerm.slug)?.seats ?? []) : [],
      events: upcoming,
      posts: s.posts.slice(0, 3).map((p) => postCard(s, p)),
      talk: featured ? slices.talkWithSpeaker(s, featured) : null,
      sponsors: s.sponsors.filter((sp) => sp.since <= year && (sp.until === null || sp.until >= year)),
      activity: [
        ...s.events.filter((e) => e.status === "published").map((e) => ({ date: e.startsAt, kind: "event" as const })),
        ...s.posts.map((p) => ({ date: p.publishedAt, kind: "post" as const })),
        ...s.talks.map((t) => ({ date: t.date, kind: "talk" as const })),
      ].filter((a) => {
        const t = Date.parse(a.date);
        return t <= now && t > now - 53 * 7 * 864e5;
      }),
      version: s.version,
    };
  },

  people(s: Snapshot): PersonCard[] {
    return [...s.people]
      .sort(
        (a, b) =>
          STATUS_ORDER.indexOf(a.status) - STATUS_ORDER.indexOf(b.status) || a.name.localeCompare(b.name),
      )
      .map(card);
  },

  person(s: Snapshot, handle: string): PublicPerson | null {
    return personIndex(s).get(handle) ?? null;
  },

  committee(s: Snapshot, term: string): CommitteeSlice | null {
    const t = s.terms.find((x) => x.slug === term);
    const c = s.committees.find((x) => x.term === term);
    if (!t || !c) return null;
    return { term: t, isCurrent: s.currentTerm?.slug === term, terms: s.terms, seats: [...c.seats].sort(bySeat) };
  },

  events(s: Snapshot): EventCard[] {
    return s.events.map(eventCard);
  },

  event(s: Snapshot, slug: string): EventSlice | null {
    const e = s.events.find((x) => x.slug === slug);
    if (!e) return null;
    const people = personIndex(s);
    const speakers = e.speakers.flatMap((h) => {
      const p = people.get(h);
      return p ? [card(p)] : [];
    });
    return { event: e, speakers };
  },

  blog(s: Snapshot): PostCard[] {
    return s.posts.map((p) => postCard(s, p));
  },

  post(s: Snapshot, slug: string): PostSlice | null {
    const p = s.posts.find((x) => x.slug === slug);
    if (!p) return null;
    const a = personIndex(s).get(p.author);
    const more = s.posts
      .filter((x) => x.slug !== slug && x.tags.some((t) => p.tags.includes(t)))
      .slice(0, 2)
      .map((x) => postCard(s, x));
    return { post: p, author: a ? card(a) : null, more };
  },

  talkWithSpeaker(s: Snapshot, t: PublicTalk): TalkWithSpeaker {
    const p = personIndex(s).get(t.speaker);
    return { ...t, speakerRef: p ? card(p) : null };
  },

  talks(s: Snapshot): TalkWithSpeaker[] {
    return [...s.talks]
      .sort((a, b) => Date.parse(b.date) - Date.parse(a.date))
      .map((t) => slices.talkWithSpeaker(s, t));
  },

  sponsors(s: Snapshot): PublicSponsor[] {
    return s.sponsors;
  },
};

/** Every slice file, keyed by its path under public/data (without ".json"). */
export function allSlices(s: Snapshot): Map<string, unknown> {
  const out = new Map<string, unknown>([
    ["index", slices.index(s)],
    ["people", slices.people(s)],
    ["events", slices.events(s)],
    ["blog", slices.blog(s)],
    ["talks", slices.talks(s)],
    ["sponsors", slices.sponsors(s)],
  ]);
  for (const p of s.people) out.set(`people/${p.handle}`, slices.person(s, p.handle));
  for (const t of s.terms) {
    const c = slices.committee(s, t.slug);
    if (c) out.set(`committee/${t.slug}`, c);
  }
  for (const e of s.events) out.set(`events/${e.slug}`, slices.event(s, e.slug));
  for (const p of s.posts) out.set(`blog/${p.slug}`, slices.post(s, p.slug));
  return out;
}

/** Every public URL the site has, used for the prerender page list and the sitemap. */
export function allPaths(s: Snapshot): string[] {
  const tags = new Set(s.posts.flatMap((p) => p.tags));
  return [
    "/",
    "/about",
    "/committee",
    "/people",
    "/alumni",
    "/advisors",
    "/events",
    "/blog",
    "/talks",
    "/sponsors",
    ...s.terms.filter((t) => s.committees.some((c) => c.term === t.slug)).map((t) => `/committee/${t.slug}`),
    ...s.people.map((p) => `/people/${p.handle}`),
    ...s.events.map((e) => `/events/${e.slug}`),
    ...s.posts.map((p) => `/blog/${p.slug}`),
    ...[...tags].map((t) => `/blog/tag/${encodeURIComponent(t)}`),
  ];
}
