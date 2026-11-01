import { Link } from "@tanstack/react-router";
import type { PersonCard } from "~/lib/slices";
import type { PersonStatus } from "~/lib/types";
import { Avatar } from "./Avatar";

export const STATUS_LABEL: Record<PersonStatus, string> = {
  committee: "Committee",
  advisor: "Advisor",
  member: "Member",
  alumni: "Alumni",
  guest: "Guest",
};

export function subtitle(p: PersonCard): string {
  if (p.status === "alumni" && (p.jobTitle || p.workplace)) return [p.jobTitle, p.workplace].filter(Boolean).join(", ");
  if (p.status === "advisor" || p.status === "guest") return p.headline;
  return [p.department, p.batch ? `batch ${p.batch}` : ""].filter(Boolean).join(", ");
}

/** A directory in hairline cells: portrait, name, where they are, what they do. */
export function PeopleGrid({ people }: { people: PersonCard[] }) {
  return (
    <ul className="grid border-t border-line sm:grid-cols-2 sm:border-l lg:grid-cols-3">
      {people.map((p) => (
        <li key={p.handle} className="group relative flex gap-4 border-b border-line py-5 sm:border-r sm:px-5 hover:bg-ground-2">
          <Avatar handle={p.handle} name={p.name} url={p.avatarUrl} size={64} />
          <div className="min-w-0">
            <p className="text-[1.0625rem] leading-tight font-semibold">
              <Link
                to="/people/$handle"
                params={{ handle: p.handle }}
                className="text-ink no-underline after:absolute after:inset-0 group-hover:underline group-hover:decoration-primary"
              >
                {p.name}
              </Link>
            </p>
            <p className="meta mt-0.5 text-xs text-ink-3">@{p.handle}</p>
            <p className="mt-1.5 text-sm text-ink-2">{subtitle(p)}</p>
            {p.status !== "advisor" && p.status !== "guest" && p.headline && (
              <p className="mt-2 line-clamp-2 text-[0.9375rem] text-ink-2">{p.headline}</p>
            )}
            {p.skills.length > 0 && (
              <p className="meta mt-2.5 flex flex-wrap gap-x-3 gap-y-1 text-xs text-ink-3">
                {p.skills.slice(0, 4).map((s) => (
                  <span key={s}>{s}</span>
                ))}
              </p>
            )}
          </div>
        </li>
      ))}
    </ul>
  );
}
