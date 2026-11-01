import { Link, createFileRoute } from "@tanstack/react-router";
import { Avatar } from "~/components/Avatar";
import { SeatGrid, advisory, groupExecutive } from "~/components/Committee";
import { EventRow, KindTag } from "~/components/events";
import { Icon } from "~/components/Icon";
import { HeroBackdrop } from "~/components/landing/HeroBackdrop";
import { Prompt } from "~/components/landing/Prompt";
import { Stats } from "~/components/landing/Stats";
import { Section } from "~/components/Page";
import { PostList } from "~/components/PostList";
import { Rsvp } from "~/components/Rsvp";
import { buttonVariants } from "~/components/ui/button";
import { SponsorGrid } from "~/components/SponsorGrid";
import { BRAND } from "~/lib/brand";
import { content } from "~/lib/content";
import { formatFull, formatShort, formatTime } from "~/lib/format";
import { seo } from "~/lib/seo";
import type { EventCard } from "~/lib/slices";
import { useUpcoming } from "~/lib/useNow";

export const Route = createFileRoute("/")({
  loader: () => content.landing(),
  head: () =>
    seo({
      title: "Binary SUST",
      description: `${BRAND.mission} The student technology club of ${BRAND.university}.`,
      path: "/",
    }),
  component: Landing,
});

/** The next event as a small code window: plain labelled fields anyone can read, and the real Register button. */
function NextEvent({ next }: { next: EventCard | undefined }) {
  return (
    <aside aria-labelledby="next-event" className="window">
      <div className="flex items-center gap-3 border-b border-line px-4 py-2.5">
        <span className="flex gap-1.5" aria-hidden="true">
          <span className="size-2.5 rounded-full border border-line-strong" />
          <span className="size-2.5 rounded-full border border-line-strong" />
          <span className="size-2.5 rounded-full border border-line-strong" />
        </span>
        <h2 id="next-event" className="meta flex items-center gap-2 text-ink-2">
          {next && <span className="live-dot size-1.5 rounded-full bg-brand-red" aria-hidden="true" />}
          {next ? "next-event" : "events"}
        </h2>
        {next && (
          <span className="ml-auto">
            <KindTag kind={next.kind} />
          </span>
        )}
      </div>
      {next ? (
        <>
          <dl className="space-y-3 px-5 pt-5 pb-5">
            <div className="kv">
              <dt>event</dt>
              <dd className="text-[1.1875rem] leading-snug font-semibold">
                <Link to="/events/$slug" params={{ slug: next.slug }} className="text-ink no-underline hover:underline hover:decoration-primary">
                  {next.title}
                </Link>
              </dd>
            </div>
            <div className="kv">
              <dt>when</dt>
              <dd>
                {formatFull(next.startsAt)}, <span className="tabular">{formatTime(next.startsAt)}</span>
              </dd>
            </div>
            {next.venue && (
              <div className="kv">
                <dt>where</dt>
                <dd>{next.venue}</dd>
              </div>
            )}
            <div className="kv max-sm:!hidden">
              <dt>about</dt>
              <dd className="line-clamp-3 text-ink-2">{next.summary}</dd>
            </div>
          </dl>
          <div className="space-y-3 border-t border-line px-5 py-4">
            <Rsvp slug={next.slug} registration={next.registration} endsAt={next.endsAt} status={next.status} compact />
            <Link to="/events/$slug" params={{ slug: next.slug }} className="more text-sm">
              Event details <Icon name="arrow" size={16} />
            </Link>
          </div>
        </>
      ) : (
        <div className="px-5 py-5">
          <p className="font-medium">Nothing is scheduled right now.</p>
          <p className="mt-1 text-[0.9375rem] text-ink-2">New sessions are announced here and on the club's Facebook page.</p>
          <Link to="/events" className="more mt-4 text-sm">
            Past events <Icon name="arrow" size={16} />
          </Link>
        </div>
      )}
    </aside>
  );
}

function Landing() {
  const data = Route.useLoaderData();
  const upcoming = useUpcoming(data.events);
  const next = upcoming[0];
  const executives = data.committee.filter((s) => s.category === "executive");
  const advisors = advisory(data.committee);
  // The landing shows the officers and the first few secretaries; the committee page has everyone.
  const preview = groupExecutive(data.committee)
    .slice(0, 2)
    .map((g, i) => ({ ...g, seats: g.seats.slice(0, i === 0 ? 5 : 6) }));
  const shown = preview.reduce((n, g) => n + g.seats.length, 0);
  const talk = data.talk;

  return (
    <>
      <section aria-labelledby="hero-title" className="relative overflow-hidden border-b border-line">
        <HeroBackdrop />
        {/* On phones the next-event window follows the mission directly, so Reserve is in the first viewport. */}
        <div className="relative mx-auto grid max-w-7xl gap-8 px-4 pt-10 pb-14 sm:px-6 sm:pt-20 lg:grid-cols-[minmax(0,1fr)_25rem] lg:gap-x-16 lg:gap-y-0 lg:pt-24 lg:pb-24">
          <div className="lg:col-start-1 lg:row-start-1 lg:self-end">
            <h1 id="hero-title" className="display text-[clamp(3rem,9vw,6rem)] leading-[0.96]">
              Binary SUST
            </h1>
            <p className="mt-4 flex flex-wrap items-baseline gap-x-3 gap-y-1">
              <span lang="bn" className="font-bangla text-[1.625rem] leading-none font-medium text-ink-2">
                {BRAND.bangla}
              </span>
              <span className="text-ink-3">{BRAND.motto}</span>
            </p>
            <div className="mt-7 max-w-[38rem] border-l border-line-strong pl-5 sm:mt-9">
<Prompt command="binary --mission" />
              <p className="mt-2.5 text-[clamp(1.25rem,2.2vw,1.5rem)] leading-snug text-ink">{BRAND.mission}</p>
            </div>
          </div>
          <div className="lg:col-start-1 lg:row-start-2 lg:self-start">
            <p className="max-w-[52ch] text-ink-2 lg:mt-6">The student technology club of {BRAND.university}.</p>
            <div className="mt-7 flex flex-wrap items-center gap-3 sm:mt-9">
              <a href="#join" className={buttonVariants()}>
                Join the club
              </a>
              <Link to="/events" className={buttonVariants({ variant: "outline" })}>
                All events
              </Link>
            </div>
          </div>
          <div className="row-start-2 lg:col-start-2 lg:row-span-2 lg:row-start-1 lg:self-center">
            <NextEvent next={next} />
          </div>
        </div>
      </section>

      <Section title="The club in numbers" index="01" id="numbers">
        <Stats stats={data.club.stats} activity={data.activity} builtAt={data.version.builtAt} />
      </Section>

      {upcoming.length > 0 && (
        <Section
          title="Coming up"
          index="02"
          id="coming-up"
          action={
            <Link to="/events" className="more">
              All events <Icon name="arrow" size={18} />
            </Link>
          }
        >
          <div className="-mt-6">
            {upcoming.map((e) => (
              <EventRow key={e.slug} e={e} />
            ))}
          </div>
        </Section>
      )}

      {preview.length > 0 && data.currentTerm && (
        <Section
          title={`The ${data.currentTerm.label} committee`}
          index="03"
          id="committee"
          action={
            <Link to="/committee" className="more">
              Committee history <Icon name="arrow" size={18} />
            </Link>
          }
        >
          {preview.map((g, i) => (
            <div key={g.title} className={i > 0 ? "mt-10" : ""}>
              <SeatGrid group={g} />
            </div>
          ))}
          {executives.length > shown && (
            <p className="mt-6">
              <Link to="/committee" className="more text-sm">
                All {executives.length} members of the committee <Icon name="arrow" size={16} />
              </Link>
            </p>
          )}
          {advisors.length > 0 && (
            <p className="mt-12 max-w-3xl border-t border-line pt-5 text-ink-2">
              Advised by{" "}
              {advisors.map((a, i) => (
                <span key={a.handle}>
                  {i > 0 && (i === advisors.length - 1 ? " and " : ", ")}
                  <Link to="/people/$handle" params={{ handle: a.handle }} className="link font-semibold text-ink">
                    {a.name}
                  </Link>{" "}
                  ({a.position.toLowerCase()})
                </span>
              ))}
              .
            </p>
          )}
        </Section>
      )}

      {talk && (
        <section className="mt-24 border-y border-line bg-ground-2" aria-labelledby="talk">
          <div className="mx-auto grid max-w-7xl gap-8 px-4 py-14 sm:px-6 md:grid-cols-[auto_minmax(0,1fr)] md:gap-12 md:py-20">
            {talk.speakerRef && (
              <Avatar handle={talk.speakerRef.handle} name={talk.speakerRef.name} url={talk.speakerRef.avatarUrl} size={136} />
            )}
            <div className="max-w-3xl">
              <h2 id="talk" className="display text-[clamp(1.75rem,3.5vw,2.5rem)]">
                {talk.title}
              </h2>
              <p className="meta mt-3 text-ink-3">
                Advisor talk · <time dateTime={talk.date}>{formatShort(talk.date)}</time>
              </p>
              <div className="mt-5 text-lg text-ink-2" dangerouslySetInnerHTML={{ __html: talk.summaryHtml }} />
              {talk.speakerRef && (
                <p className="mt-5">
                  <Link to="/people/$handle" params={{ handle: talk.speakerRef.handle }} className="link font-semibold text-ink">
                    {talk.speakerRef.name}
                  </Link>
                  <span className="text-ink-2">, {talk.speakerRef.headline}</span>
                </p>
              )}
              <Link to="/talks" className="more mt-7">
                All talks <Icon name="arrow" size={18} />
              </Link>
            </div>
          </div>
        </section>
      )}

      {data.posts.length > 0 && (
        <Section
          title="Latest writing"
          index="04"
          id="writing"
          action={
            <Link to="/blog" className="more">
              The blog <Icon name="arrow" size={18} />
            </Link>
          }
        >
          <div className="-mt-8">
            <PostList posts={data.posts} />
          </div>
        </Section>
      )}

      {data.sponsors.length > 0 && (
        <Section
          title="Supported by"
          index="05"
          id="sponsors"
          action={
            <Link to="/sponsors" className="more">
              Our sponsors <Icon name="arrow" size={18} />
            </Link>
          }
        >
          <SponsorGrid sponsors={data.sponsors} />
        </Section>
      )}

      <section id="join" className="mx-auto max-w-7xl px-4 pt-24 sm:px-6" aria-labelledby="join-title">
        <div className="grid gap-8 border-t-2 border-ink pt-10 md:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] md:gap-16">
          <div>
            <h2 id="join-title" className="display text-[clamp(2rem,5vw,3.25rem)]">
              Turn up to one session.
            </h2>
            <p className="mt-5 max-w-[56ch] text-lg text-ink-2">
              That is how most members started. Any SUST student can come, from any department: sign in with Google to
              reserve a seat, and write to the committee if you want to help run things.
            </p>
          </div>
          <div className="flex flex-col items-start gap-3 md:pt-2">
            <Link to="/events" className="btn btn-primary">
              See upcoming events <Icon name="arrow" size={18} />
            </Link>
            <a href={`mailto:${BRAND.email}`} className="btn btn-ghost max-w-full">
              <Icon name="mail" size={18} /> <span className="truncate">{BRAND.email}</span>
            </a>
            <p className="mt-2 text-sm text-ink-3">
              Or message us on{" "}
              {BRAND.links.map((l, i) => (
                <span key={l.href}>
                  {i > 0 && " or "}
                  <a href={l.href} rel="noopener" className="link">
                    {l.label}
                  </a>
                </span>
              ))}
              .
            </p>
          </div>
        </div>
      </section>
    </>
  );
}
