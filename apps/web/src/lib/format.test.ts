import { describe, expect, it } from "vitest";
import { dhakaDate, formatDay, formatFull, formatMonth, formatRange, formatShort, formatTime, formatWeekday, isPast } from "./format";

// 19:30 UTC on 11 Dec is 01:30 on 12 Dec in Sylhet (UTC+6): every formatter must say the 12th.
const LATE_UTC = "2026-12-11T19:30:00Z";

describe("format (Asia/Dhaka, en-GB)", () => {
  it("formats the calendar day in Sylhet, not UTC", () => {
    expect(formatDay(LATE_UTC)).toBe("12");
    expect(formatMonth(LATE_UTC)).toBe("Dec");
    expect(formatWeekday(LATE_UTC)).toBe("Sat");
    expect(formatShort(LATE_UTC)).toBe("12 Dec 2026");
    expect(formatFull(LATE_UTC)).toMatch(/^Saturday,? 12 December 2026$/);
    expect(dhakaDate(LATE_UTC)).toEqual({ day: 12, month: 12, year: 2026, hour: 1 });
  });

  it("formats times on a 12-hour clock", () => {
    expect(formatTime(LATE_UTC)).toMatch(/^1:30\s?am$/);
    expect(formatTime("2026-12-12T09:00:00Z")).toMatch(/^3:00\s?pm$/);
  });

  it("shows one day with times when start and end share a Sylhet date", () => {
    // 00:30 and 16:00 on 12 Dec in Sylhet, although the UTC dates differ.
    const r = formatRange("2026-12-11T18:30:00Z", "2026-12-12T10:00:00Z");
    expect(r).toMatch(/^Saturday,? 12 December 2026, 12:30\s?am to 4:00\s?pm$/);
  });

  it("shows a date span when the event runs over several days", () => {
    expect(formatRange("2026-12-12T03:00:00Z", "2026-12-13T12:00:00Z")).toBe("12 Dec 2026 to 13 Dec 2026");
  });

  it("isPast compares the end instant with now", () => {
    const now = Date.parse("2026-12-12T00:00:00Z");
    expect(isPast("2026-12-11T23:59:59Z", now)).toBe(true);
    expect(isPast("2026-12-12T00:00:01Z", now)).toBe(false);
  });
});
