import type { Me, Permission } from "@binarysust/api-client";
import { describe, expect, it } from "vitest";
import { allowed, findSection, sections, visibleSections } from "./sections";

const me = (permissions: Permission[], roles: string[] = []): Me => ({
  userId: "u1",
  permissions,
  roles,
  account: { id: "a1", email: "x@example.com", name: "X", avatarUrl: null, personHandle: null, personId: null },
});

const paths = (m: Me) => visibleSections(m).map((s) => s.path);
// What every signed-in account holds (club.Baseline).
const BASELINE: Permission[] = ["profile.edit_own", "blog.write"];

describe("dashboard sections", () => {
  it("shows a plain member only the signed-in and baseline sections", () => {
    expect(paths(me(BASELINE))).toEqual(["", "portfolio", "rsvps", "blog"]);
  });

  it("adds a section for each permission a role grants", () => {
    expect(paths(me([...BASELINE, "events.manage", "events.checkin"], ["event_manager"]))).toEqual([
      "",
      "portfolio",
      "rsvps",
      "events",
      "blog",
    ]);
    expect(paths(me(["blog.review"]))).toContain("blog"); // any one of a list is enough
    expect(paths(me(["sponsors.manage"]))).toEqual(["", "rsvps", "sponsors"]);
  });

  it("shows an admin every section", () => {
    const all: Permission[] = [
      "blog.review",
      "blog.write",
      "events.checkin",
      "events.manage",
      "media.upload",
      "people.manage",
      "profile.edit_own",
      "roles.manage",
      "sponsors.manage",
      "talks.manage",
    ];
    expect(paths(me(all, ["admin"]))).toEqual(sections.map((s) => s.path));
  });

  it("checks one section at a time", () => {
    const committee = sections.find((s) => s.path === "committee")!;
    expect(allowed(me(["people.manage"]), committee)).toBe(false);
    expect(allowed(me(["roles.manage"]), committee)).toBe(true);
  });

  it("finds the section for a /dashboard/* path and the rest below it", () => {
    expect(findSection("events/42/registrants")).toMatchObject({ section: { path: "events" }, rest: ["42", "registrants"] });
    expect(findSection("").section?.path).toBe("");
    expect(findSection("nope").section).toBeUndefined();
  });
});
