import { useEffect, useRef } from "react";
import type { Activity } from "~/lib/slices";

// A contribution graph of the club's real output: one cell per day for the 53 weeks before the snapshot was built,
// darker the more happened that day (events, posts, talks). Sunday-first columns, like GitHub's.
const DAY = 864e5;
const TZ_OFFSET = 6 * 3600e3; // Asia/Dhaka, no DST
const dayNo = (iso: string) => Math.floor((Date.parse(iso) + TZ_OFFSET) / DAY);
const dateOf = (n: number) =>
  new Intl.DateTimeFormat("en-GB", { timeZone: "UTC", weekday: "short", day: "numeric", month: "short", year: "numeric" }).format(
    new Date(n * DAY),
  );
const monthOf = (n: number) => new Intl.DateTimeFormat("en-GB", { timeZone: "UTC", month: "short" }).format(new Date(n * DAY));

const CELL = 11;
const GAP = 3;
const STEP = CELL + GAP;
const LEFT = 28;
const TOP = 18;
const LEVEL = ["var(--graph-0)", "var(--graph-1)", "var(--graph-2)", "var(--graph-3)"] as const;
const plural = (n: number, w: string) => `${n} ${w}${n === 1 ? "" : "s"}`;

function describe(items: Activity[]): string {
  const c = { event: 0, post: 0, talk: 0 };
  for (const a of items) c[a.kind]++;
  return [c.event && plural(c.event, "event"), c.post && plural(c.post, "post"), c.talk && plural(c.talk, "talk")].filter(Boolean).join(", ");
}

export function ActivityGraph({ activity, builtAt }: { activity: Activity[]; builtAt: string }) {
  const scroller = useRef<HTMLDivElement>(null);
  // On narrow screens the graph scrolls; start at the latest weeks.
  useEffect(() => {
    const el = scroller.current;
    if (el) el.scrollLeft = el.scrollWidth;
  }, []);

  const end = dayNo(builtAt);
  const weekday = (end + 4) % 7; // 1970-01-01 was a Thursday; 0 = Sunday
  const start = end - weekday - 52 * 7;
  const byDay = new Map<number, Activity[]>();
  for (const a of activity) {
    const d = dayNo(a.date);
    if (d < start || d > end) continue;
    byDay.set(d, [...(byDay.get(d) ?? []), a]);
  }
  const weeks = 53;
  const width = LEFT + weeks * STEP;
  const height = TOP + 7 * STEP;
  const total = describe(activity);

  const cells = [];
  const months = [];
  let lastMonth = "";
  for (let w = 0; w < weeks; w++) {
    const first = start + w * 7;
    const m = monthOf(first);
    // Label each month at its first column; skip the first column's label if the next month starts too close.
    if (m !== lastMonth) {
      if ((w > 0 || monthOf(start + 21) === m) && w < weeks - 2) months.push({ x: LEFT + w * STEP, label: m });
      lastMonth = m;
    }
    for (let d = 0; d < 7; d++) {
      const n = first + d;
      if (n > end) break;
      const items = byDay.get(n) ?? [];
      const level = Math.min(items.length, 3);
      cells.push(
        <rect
          key={n}
          x={LEFT + w * STEP}
          y={TOP + d * STEP}
          width={CELL}
          height={CELL}
          rx={2}
          fill={LEVEL[level]}
          stroke={level === 0 ? "var(--line)" : "none"}
          strokeWidth={1}
        >
          <title>{items.length ? `${describe(items)} on ${dateOf(n)}` : `Nothing on ${dateOf(n)}`}</title>
        </rect>,
      );
    }
  }

  return (
    <figure>
      <div ref={scroller} className="overflow-x-auto pb-2 [scrollbar-width:thin]">
        <svg
          viewBox={`0 0 ${width} ${height}`}
          width={width}
          height={height}
          className="block max-w-none md:h-auto md:w-full md:max-w-[52rem]"
          role="img"
          aria-label={`Club activity over the last 12 months: ${total || "nothing recorded yet"}.`}
        >
          <g className="fill-ink-3" fontFamily="var(--ff-mono)" fontSize="10">
            {months.map((m) => (
              <text key={m.x} x={m.x} y={10}>
                {m.label}
              </text>
            ))}
            <text x={0} y={TOP + 1 * STEP + 9}>Mon</text>
            <text x={0} y={TOP + 3 * STEP + 9}>Wed</text>
            <text x={0} y={TOP + 5 * STEP + 9}>Fri</text>
          </g>
          {cells}
        </svg>
      </div>
      <figcaption className="mt-3 flex flex-wrap items-center justify-between gap-x-6 gap-y-2 text-sm text-ink-2">
        <span>
          {total ? `${total} in the last 12 months.` : "Nothing recorded in the last 12 months yet."}{" "}
          <span className="text-ink-3 [@media(pointer:coarse)]:hidden">Hover a day for details.</span>
        </span>
        <span className="meta flex items-center gap-1.5 text-xs text-ink-3" aria-hidden="true">
          Less
          {LEVEL.map((f, i) => (
            <span key={f} className="inline-block size-[11px] rounded-[2px]" style={{ background: f, outline: i === 0 ? "1px solid var(--line)" : undefined, outlineOffset: -1 }} />
          ))}
          More
        </span>
      </figcaption>
    </figure>
  );
}
