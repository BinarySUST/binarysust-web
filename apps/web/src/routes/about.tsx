import { Link, createFileRoute } from "@tanstack/react-router";
import { Icon } from "~/components/Icon";
import { PageHead, Section } from "~/components/Page";
import { Stats } from "~/components/landing/Stats";
import { BRAND } from "@binarysust/ui/lib/brand";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";
import { buttonVariants } from "@binarysust/ui/components/button";

export const Route = createFileRoute("/about")({
  loader: () => content.landing(),
  head: ({ loaderData }) =>
    seo({ title: "About", description: loaderData?.club.about.slice(0, 160) ?? "About Binary SUST.", path: "/about" }),
  component: About,
});

function About() {
  const { club, currentTerm } = Route.useLoaderData();
  return (
    <>
      <PageHead title={`About ${club.name}`} lede={`The student technology club of ${BRAND.university}.`} />
      <div className="mx-auto grid max-w-7xl gap-x-16 gap-y-12 border-t border-line-strong px-4 pt-10 sm:px-6 lg:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="prose text-[1.125rem]">
          <p>{club.about}</p>
          <h2>What the club is for</h2>
          <p>The club's constitution sets out its objectives:</p>
          <ul>
            {BRAND.objectives.map((o) => (
              <li key={o}>{o}.</li>
            ))}
          </ul>
          <h2>How the club runs</h2>
          <p>
            An executive committee elected for each term runs the club: a president, a general secretary, secretaries for training, events, publication, IT and design, and executive members. An advisory committee of faculty, with the Vice-Chancellor as Chief Patron, guides it.
            {currentTerm && (
              <>
                {" "}
                The current committee is the <Link to="/committee">{currentTerm.label} committee</Link>; every earlier
                term stays on record.
              </>
            )}
          </p>
          <p>
            Members keep public portfolios on this site. Alumni stay listed after they graduate, so first-years can see
            where a SUST degree and a few late-night contests lead.
          </p>
        </div>
        <aside className="space-y-6">
          <section aria-labelledby="contact" className="border border-line-strong bg-panel">
            <h2 id="contact" className="border-b border-line px-5 py-3 text-[0.9375rem] font-semibold">
              Get in touch
            </h2>
            <ul className="space-y-2.5 px-5 py-4">
              <li>
                <a href={`mailto:${BRAND.email}`} className="link inline-flex items-center gap-2 break-all">
                  <Icon name="mail" size={16} /> {BRAND.email}
                </a>
              </li>
              {BRAND.links.map((l) => (
                <li key={l.href}>
                  <a href={l.href} className="link inline-flex items-center gap-2" rel="me noopener">
                    <Icon name="external" size={16} /> {l.label}
                  </a>
                </li>
              ))}
            </ul>
            <p className="border-t border-line px-5 py-3 text-sm text-ink-2">{BRAND.university}</p>
          </section>
          <div className="flex flex-wrap gap-3">
            <Link to="/advisors" className={buttonVariants({ variant: "outline", size: "sm" })}>
              Our advisors
            </Link>
            <Link to="/alumni" className={buttonVariants({ variant: "outline", size: "sm" })}>
              Alumni
            </Link>
          </div>
        </aside>
      </div>
      <Section title="The club in numbers" id="numbers">
        <div className="max-w-xl">
          <Stats stats={club.stats} />
        </div>
      </Section>
    </>
  );
}
