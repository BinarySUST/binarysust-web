import type { PublicSponsor } from "~/content/types";

/** Sponsor logos in a hairline grid. Static: no marquee, nothing moves. */
export function SponsorGrid({ sponsors }: { sponsors: PublicSponsor[] }) {
  if (sponsors.length === 0) return null;
  return (
    <ul className="grid grid-cols-2 border-t border-l border-line sm:grid-cols-3 lg:grid-cols-4">
      {sponsors.map((s) => (
        <li key={s.name} className="border-r border-b border-line">
          <a
            href={s.url}
            rel="sponsored noopener"
            className="group flex h-full flex-col items-center justify-center gap-3 px-4 py-7 no-underline hover:bg-ground-2"
          >
            <img src={s.logoUrl} alt={s.name} className="h-10 w-auto max-w-full sm:h-12" loading="lazy" decoding="async" />
            <span className="text-sm text-ink-2 group-hover:text-ink">{s.name}</span>
          </a>
        </li>
      ))}
    </ul>
  );
}
