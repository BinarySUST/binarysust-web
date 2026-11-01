import { Link } from "@tanstack/react-router";
import { formatDay, formatMonth, formatShort, formatTime, formatWeekday } from "~/lib/format";
import type { EventCard } from "~/lib/slices";
import { Icon } from "./Icon";
import { Badge } from "@binarysust/ui/components/badge";

export const KIND_LABEL: Record<EventCard["kind"], string> = {
  workshop: "Workshop",
  contest: "Contest",
  hackathon: "Hackathon",
  seminar: "Seminar",
  meetup: "Meetup",
};

/** A calendar date block: weekday, day, month, in the mono metadata voice. */
export function DateBlock({ iso, size = "md" }: { iso: string; size?: "md" | "lg" }) {
  const big = size === "lg";
  return (
    <div
      className={`flex shrink-0 flex-col items-center self-start border border-line-strong bg-panel text-center ${big ? "w-[5.5rem] py-2.5" : "w-[4.25rem] py-1.5"}`}
    >
      <span className="meta text-xs text-ink-3">{formatWeekday(iso)}</span>
      <span className={`display leading-none tabular ${big ? "my-1 text-[2.5rem]" : "my-0.5 text-[1.75rem]"}`}>{formatDay(iso)}</span>
      <span className="meta text-xs text-ink-2">{formatMonth(iso)}</span>
    </div>
  );
}

export function KindTag({ kind }: { kind: EventCard["kind"] }) {
  return <Badge>{KIND_LABEL[kind]}</Badge>;
}

/** One line of the schedule: date, what and where, then the way in. */
export function EventRow({ e, past = false }: { e: EventCard; past?: boolean }) {
  return (
    <article className="group relative grid grid-cols-[auto_1fr] gap-x-5 gap-y-2 border-b border-line py-6 sm:gap-x-8 md:grid-cols-[auto_1fr_14rem]">
      <DateBlock iso={e.startsAt} />
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <KindTag kind={e.kind} />
          {e.status === "cancelled" && <Badge variant="destructive">Cancelled</Badge>}
          {!past && e.registration && e.status !== "cancelled" && <span className="meta text-ink-3">Registration required</span>}
        </div>
        <h3 className="mt-2 text-xl leading-snug font-semibold sm:text-[1.375rem]">
          <Link
            to="/events/$slug"
            params={{ slug: e.slug }}
            className="text-ink no-underline after:absolute after:inset-0 group-hover:underline group-hover:decoration-primary"
          >
            {e.title}
          </Link>
        </h3>
        <p className="mt-1.5 max-w-[62ch] text-ink-2">{e.summary}</p>
      </div>
      <dl className="meta col-start-2 space-y-1 text-ink-2 md:col-start-3 md:pt-1">
        <div className="flex items-center gap-2">
          <dt className="sr-only">Time</dt>
          <Icon name="clock" size={15} className="text-ink-3" />
          <dd>{past ? formatShort(e.startsAt) : formatTime(e.startsAt)}</dd>
        </div>
        {e.venue && (
          <div className="flex items-start gap-2">
            <dt className="sr-only">Venue</dt>
            <Icon name="pin" size={15} className="mt-0.5 shrink-0 text-ink-3" />
            <dd className="font-sans text-[0.9375rem]">{e.venue}</dd>
          </div>
        )}
      </dl>
    </article>
  );
}
