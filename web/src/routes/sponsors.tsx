import { createFileRoute } from "@tanstack/react-router";
import { PageHead, SectionHead } from "~/components/Page";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";
import type { PublicSponsor } from "~/lib/types";

export const Route = createFileRoute("/sponsors")({
  loader: () => content.sponsors(),
  head: () => seo({ title: "Sponsors", description: "The companies and partners that back Binary SUST.", path: "/sponsors" }),
  component: Sponsors,
});

const TIERS: { tier: PublicSponsor["tier"]; title: string; logo: string }[] = [
  { tier: "platinum", title: "Platinum", logo: "h-16 sm:h-20" },
  { tier: "gold", title: "Gold", logo: "h-14" },
  { tier: "silver", title: "Silver", logo: "h-11" },
  { tier: "partner", title: "Partners", logo: "h-11" },
];

function Sponsors() {
  const sponsors = Route.useLoaderData();
  const year = new Date(__BUILD_TIME__).getUTCFullYear();
  return (
    <>
      <PageHead title="Sponsors" lede="The organisations that fund our prizes, venues and sessions. Thank you." />
      <div className="mx-auto max-w-7xl space-y-16 px-4 sm:px-6">
        {TIERS.filter(({ tier }) => sponsors.some((s) => s.tier === tier)).map(({ tier, title, logo }, i) => {
          const list = sponsors.filter((s) => s.tier === tier);
          return (
            <section key={tier} aria-labelledby={`tier-${tier}`}>
              <SectionHead id={`tier-${tier}`} index={String(i + 1).padStart(2, "0")} title={title} />
              <ul className="grid border-t border-l border-line sm:grid-cols-2 lg:grid-cols-3">
                {list.map((s) => (
                  <li key={s.name} className="border-r border-b border-line">
                    <a href={s.url} rel="sponsored noopener" className="group flex h-full flex-col gap-5 p-6 no-underline hover:bg-ground-2">
                      <span className="flex min-h-24 items-center">
                        <img src={s.logoUrl} alt={s.name} className={`${logo} w-auto max-w-full`} loading="lazy" />
                      </span>
                      <span className="border-t border-line pt-3">
                        <span className="block font-semibold text-ink group-hover:underline">{s.name}</span>
                        <span className="meta text-ink-3">
                          Since {s.since}
                          {s.until !== null && s.until < year ? ` (until ${s.until})` : ""}
                        </span>
                      </span>
                    </a>
                  </li>
                ))}
              </ul>
            </section>
          );
        })}
        <p className="max-w-2xl text-ink-2">
          Want to support the club? Write to the committee from the <a href="/about" className="link">about page</a>.
        </p>
      </div>
    </>
  );
}
