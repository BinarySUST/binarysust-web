import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { STATUS_ORDER, allPaths, allSlices, slices } from "./slices";
import type { PublicEvent, Snapshot } from "./types";

// The committed fixture has exactly the shape of site.Snapshot (docs/snapshot.md), built at 2026-10-30T14:15Z.
const fixture = JSON.parse(readFileSync(new URL("../../.snapshot/fixture.json", import.meta.url), "utf8")) as Snapshot;
const builtAt = Date.parse(fixture.version.builtAt);

describe("slices.index (the landing page)", () => {
  const index = slices.index(fixture);

  it("lists at most three published events that have not ended, soonest first", () => {
    expect(index.events.length).toBeLessThanOrEqual(3);
    expect(index.events.length).toBeGreaterThan(0);
    for (const e of index.events) {
      expect(e.status).toBe("published");
      expect(Date.parse(e.endsAt)).toBeGreaterThanOrEqual(builtAt);
      expect(e).not.toHaveProperty("bodyHtml");
    }
    const starts = index.events.map((e) => Date.parse(e.startsAt));
    expect(starts).toEqual([...starts].sort((a, b) => a - b));
  });

  it("drops cancelled and finished events", () => {
    const past: PublicEvent = { ...fixture.events[0]!, slug: "past", startsAt: "2026-01-01T10:00:00Z", endsAt: "2026-01-01T12:00:00Z" };
    const cancelled: PublicEvent = { ...fixture.events[0]!, slug: "off", status: "cancelled", endsAt: "2027-01-01T00:00:00Z" };
    const s: Snapshot = { ...fixture, events: [past, cancelled] };
    expect(slices.index(s).events).toEqual([]);
  });

  it("keeps the latest three posts, with the author as a reference", () => {
    expect(index.posts.map((p) => p.slug)).toEqual(fixture.posts.slice(0, 3).map((p) => p.slug));
    for (const p of index.posts) {
      expect(p).not.toHaveProperty("bodyHtml");
      if (p.author) expect(Object.keys(p.author).sort()).toEqual(["avatarUrl", "handle", "name"]);
    }
  });

  it("keeps sponsors active in the build year and activity from the 53 weeks before the build", () => {
    const year = new Date(builtAt).getUTCFullYear();
    for (const s of index.sponsors) {
      expect(s.since).toBeLessThanOrEqual(year);
      expect(s.until === null || s.until >= year).toBe(true);
    }
    for (const a of index.activity) {
      const t = Date.parse(a.date);
      expect(t).toBeLessThanOrEqual(builtAt);
      expect(t).toBeGreaterThan(builtAt - 53 * 7 * 864e5);
    }
    expect(index.committee).toEqual(slices.committee(fixture, fixture.currentTerm!.slug)!.seats);
  });
});

describe("slices.people", () => {
  it("orders by status (committee, advisor, member, alumni, guest), then name, as cards", () => {
    const people = slices.people(fixture);
    expect(people).toHaveLength(fixture.people.length);
    for (let i = 1; i < people.length; i++) {
      const [a, b] = [people[i - 1]!, people[i]!];
      const byStatus = STATUS_ORDER.indexOf(a.status) - STATUS_ORDER.indexOf(b.status);
      expect(byStatus < 0 || (byStatus === 0 && a.name.localeCompare(b.name) <= 0)).toBe(true);
    }
    expect(people[0]).not.toHaveProperty("bioHtml");
  });
});

describe("slices.committee", () => {
  it("puts the executive committee first, then orders by rank", () => {
    const c = slices.committee(fixture, fixture.currentTerm!.slug)!;
    expect(c.isCurrent).toBe(true);
    const firstAdvisor = c.seats.findIndex((s) => s.category === "advisor");
    if (firstAdvisor >= 0) expect(c.seats.slice(firstAdvisor).every((s) => s.category === "advisor")).toBe(true);
    for (let i = 1; i < c.seats.length; i++) {
      const [a, b] = [c.seats[i - 1]!, c.seats[i]!];
      if (a.category === b.category) expect(a.rank).toBeLessThanOrEqual(b.rank);
    }
  });

  it("is null for a term without a committee", () => {
    expect(slices.committee(fixture, "1999-00")).toBeNull();
  });
});

describe("slices.event and slices.post", () => {
  it("resolves event speakers to people cards and drops unknown handles", () => {
    const e = fixture.events.find((x) => x.speakers.length > 0)!;
    const s: Snapshot = { ...fixture, events: [{ ...e, speakers: [...e.speakers, "nobody-by-this-handle"] }] };
    const slice = slices.event(s, e.slug)!;
    expect(slice.speakers.map((p) => p.handle)).toEqual(e.speakers);
    expect(slices.event(fixture, "no-such-event")).toBeNull();
  });

  it("suggests up to two other posts that share a tag", () => {
    for (const p of fixture.posts) {
      const { more } = slices.post(fixture, p.slug)!;
      expect(more.length).toBeLessThanOrEqual(2);
      for (const m of more) {
        expect(m.slug).not.toBe(p.slug);
        expect(m.tags.some((t) => p.tags.includes(t))).toBe(true);
      }
    }
  });

  it("orders talks newest first", () => {
    const dates = slices.talks(fixture).map((t) => Date.parse(t.date));
    expect(dates).toEqual([...dates].sort((a, b) => b - a));
  });
});

describe("allPaths and allSlices", () => {
  it("has a slice for every pre-rendered detail page, and no duplicate URLs", () => {
    const paths = allPaths(fixture);
    expect(new Set(paths).size).toBe(paths.length);
    const files = allSlices(fixture);
    for (const p of paths.filter((x) => /^\/(people|events|blog|committee)\/[^/]+$/.test(x))) {
      expect(files.has(p.slice(1)), p).toBe(true);
    }
    expect(files.has("index")).toBe(true);
  });
});
