import { Link } from "@tanstack/react-router";
import type { CommitteeSlice } from "~/content/slices";
import type { Seat } from "~/content/types";
import { Avatar } from "~/components/Avatar";
import { SectionHead } from "~/components/Page";
import { badgeVariants } from "@binarysust/ui/components/badge";

// A committee can hold thirty seats. Lead with the officers, then group the rest by kind of post so the page stays
// scannable: officers get portraits, secretaries a compact grid, executive members a dense list.
// Seats carry `group` once the snapshot provides it; until then the group is inferred from the position title.
type Group = "patron" | "adviser" | "leadership" | "secretary" | "assistant" | "member";
const LEAD = /^(president|vice[- ]?president|general secretary|director)/i;
const ASSISTANT = /\b(assistant|joint|deputy)\b/i;
const MEMBER = /\bmember\b/i;
const byRank = (a: Seat, b: Seat) => a.rank - b.rank;

function groupOf(s: Seat): Group {
  const g = (s as Seat & { group?: string }).group;
  if (g === "patron" || g === "adviser" || g === "leadership" || g === "secretary" || g === "assistant" || g === "member") return g;
  if (s.category === "advisor") return /patron/i.test(s.position) ? "patron" : "adviser";
  if (LEAD.test(s.position)) return "leadership";
  if (MEMBER.test(s.position)) return "member";
  if (ASSISTANT.test(s.position)) return "assistant";
  return "secretary";
}

export type SeatGroup = { title: string; size: "lead" | "grid" | "list"; seats: Seat[] };

const EXEC_GROUPS: { group: Group; title: string; size: SeatGroup["size"] }[] = [
  { group: "leadership", title: "Executive committee", size: "lead" },
  { group: "secretary", title: "Secretaries", size: "grid" },
  { group: "assistant", title: "Assistant secretaries", size: "grid" },
  { group: "member", title: "Executive members", size: "list" },
];

export function groupExecutive(seats: Seat[]): SeatGroup[] {
  const exec = seats.filter((s) => s.category === "executive").sort(byRank);
  // A small committee with no named officers reads best as one group.
  if (!exec.some((s) => groupOf(s) === "leadership"))
    return exec.length ? [{ title: "Executive committee", size: "lead", seats: exec }] : [];
  return EXEC_GROUPS.map((g) => ({ title: g.title, size: g.size, seats: exec.filter((s) => groupOf(s) === g.group) })).filter(
    (g) => g.seats.length > 0,
  );
}

/** Advisory committee: Chief Patron first, then the chief adviser, then advisers, each by rank. */
export function advisory(seats: Seat[]): Seat[] {
  const weight = (s: Seat) => (groupOf(s) === "patron" ? 0 : /chief/i.test(s.position) ? 1 : 2);
  return seats.filter((s) => s.category === "advisor").sort((a, b) => weight(a) - weight(b) || byRank(a, b));
}

function SeatCard({ s, size }: { s: Seat; size: SeatGroup["size"] }) {
  const link = (
    <Link
      to="/people/$handle"
      params={{ handle: s.handle }}
      className="text-ink no-underline after:absolute after:inset-0 group-hover:underline group-hover:decoration-primary"
    >
      {s.name}
    </Link>
  );
  if (size === "lead") {
    return (
      <li className="group relative">
        <Avatar handle={s.handle} name={s.name} url={s.avatarUrl} size={112} />
        <p className="mt-3 text-[1.0625rem] leading-tight font-semibold">{link}</p>
        <p className="mt-1 text-sm text-ink-2">{s.position}</p>
      </li>
    );
  }
  return (
    <li className={`group relative flex items-center gap-3 border-b border-line ${size === "grid" ? "py-3" : "py-2"}`}>
      <Avatar handle={s.handle} name={s.name} url={s.avatarUrl} size={size === "grid" ? 48 : 36} />
      <div className="min-w-0">
        <p className="truncate leading-tight font-semibold">{link}</p>
        {size === "grid" && <p className="mt-0.5 truncate text-sm text-ink-2">{s.position}</p>}
      </div>
    </li>
  );
}

export function SeatGrid({ group }: { group: SeatGroup }) {
  const cls =
    group.size === "lead"
      ? "grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-5"
      : group.size === "grid"
        ? "grid gap-x-8 sm:grid-cols-2 lg:grid-cols-3"
        : "grid gap-x-8 grid-cols-1 min-[420px]:grid-cols-2 md:grid-cols-3 lg:grid-cols-4";
  return (
    <ul className={cls}>
      {group.seats.map((s) => (
        <SeatCard key={s.handle + s.position} s={s} size={group.size} />
      ))}
    </ul>
  );
}

export function CommitteeView({ data }: { data: CommitteeSlice }) {
  const groups: SeatGroup[] = [...groupExecutive(data.seats)];
  const advisers = advisory(data.seats);
  if (advisers.length) groups.push({ title: "Advisory committee", size: advisers.length > 5 ? "grid" : "lead", seats: advisers });

  return (
    <div className="mx-auto max-w-7xl px-4 sm:px-6">
      <nav aria-label="Committee terms" className="mb-14">
        <h2 className="tlabel mb-2">Terms</h2>
        <ul className="flex flex-wrap gap-2">
          {data.terms.map((t) => {
            const active = t.slug === data.term.slug;
            return (
              <li key={t.slug}>
                <Link
                  to="/committee/$term"
                  params={{ term: t.slug }}
                  aria-current={active ? "page" : undefined}
                  className={badgeVariants({ className: "min-h-9 px-3 text-[0.8125rem]" })}
                >
                  {t.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
      {groups.map((g, i) => (
        <section key={g.title} className="mb-16" aria-label={g.title}>
          <SectionHead
            title={g.title}
            index={String(i + 1).padStart(2, "0")}
            action={
              <span className="meta text-ink-3">
                {g.seats.length} {g.seats.length === 1 ? "seat" : "seats"}
              </span>
            }
          />
          <SeatGrid group={g} />
        </section>
      ))}
    </div>
  );
}
