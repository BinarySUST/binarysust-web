// Dates render identically at pre-render and in the browser: fixed locale, fixed zone (Sylhet).
const TZ = "Asia/Dhaka";
const fmt = (o: Intl.DateTimeFormatOptions) => new Intl.DateTimeFormat("en-GB", { timeZone: TZ, ...o });

const day = fmt({ day: "numeric" });
const mon = fmt({ month: "short" });
const wk = fmt({ weekday: "short" });
const full = fmt({ weekday: "long", day: "numeric", month: "long", year: "numeric" });
const short = fmt({ day: "numeric", month: "short", year: "numeric" });
const time = fmt({ hour: "numeric", minute: "2-digit", hour12: true });
const parts = fmt({ day: "numeric", month: "numeric", year: "numeric", hour: "numeric", hourCycle: "h23" });

export const formatDay = (iso: string) => day.format(new Date(iso));
export const formatMonth = (iso: string) => mon.format(new Date(iso));
export const formatWeekday = (iso: string) => wk.format(new Date(iso));
export const formatFull = (iso: string) => full.format(new Date(iso));
export const formatShort = (iso: string) => short.format(new Date(iso));
export const formatTime = (iso: string) => time.format(new Date(iso)).replace(" ", " ");

/** Calendar parts of an instant in Sylhet, as numbers. */
export function dhakaDate(iso: string): { day: number; month: number; year: number; hour: number } {
  const p = Object.fromEntries(parts.formatToParts(new Date(iso)).map((x) => [x.type, x.value]));
  return { day: Number(p.day), month: Number(p.month), year: Number(p.year), hour: Number(p.hour) };
}

export function formatRange(startsAt: string, endsAt: string): string {
  const sameDay = formatShort(startsAt) === formatShort(endsAt);
  return sameDay
    ? `${formatFull(startsAt)}, ${formatTime(startsAt)} to ${formatTime(endsAt)}`
    : `${formatShort(startsAt)} to ${formatShort(endsAt)}`;
}

export const isPast = (endsAt: string, now = Date.now()) => Date.parse(endsAt) < now;
