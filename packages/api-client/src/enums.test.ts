import { describe, expect, it } from "vitest";
import { EVENT_KINDS, POST_STATE_LABEL, POST_STATES, SPONSOR_TIERS, oneOf } from "./enums";

describe("enums", () => {
  it("lists every value once, in display order", () => {
    expect(SPONSOR_TIERS).toEqual(["platinum", "gold", "silver", "partner"]);
    expect(new Set(EVENT_KINDS).size).toBe(EVENT_KINDS.length);
    expect(Object.keys(POST_STATE_LABEL).sort()).toEqual([...POST_STATES].sort());
  });

  it("oneOf narrows a <select> value, falling back to the first value", () => {
    expect(oneOf(SPONSOR_TIERS, "gold")).toBe("gold");
    expect(oneOf(SPONSOR_TIERS, "diamond")).toBe("platinum");
  });
});
