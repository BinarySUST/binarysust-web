import { Link, createFileRoute } from "@tanstack/react-router";
import { Avatar } from "~/components/Avatar";
import { DateBlock, KindTag } from "~/components/events";
import { Icon } from "~/components/Icon";
import { subtitle } from "~/components/People";
import { Rsvp } from "~/components/Rsvp";
import { Badge } from "~/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "~/components/ui/card";
import { content } from "~/lib/content";
import { formatRange } from "~/lib/format";
import { highlightHtml } from "~/lib/highlight";
import { seo } from "~/lib/seo";
import { useNow } from "~/lib/useNow";

export const Route = createFileRoute("/events/$slug")({
  loader: ({ params }) => content.event(params.slug), // snapshot slice: disk at build, static JSON in the browser
  head: ({ loaderData, params }) =>
    seo({
      title: loaderData?.event.title ?? params.slug,
      description: loaderData?.event.summary ?? "A Binary SUST event.",
      path: `/events/${params.slug}`,
      type: "article",
      image: loaderData?.event.coverUrl,
    }),
  component: EventPage,
});

function EventPage() {
  const { event: e, speakers } = Route.useLoaderData();
  const now = useNow();
  const over = Date.parse(e.endsAt) < now;

  return (
    <article className="mx-auto max-w-7xl px-4 pt-10 sm:px-6 sm:pt-14">
      <Link to="/events" className="more text-sm text-ink-2">
        <Icon name="arrow" size={16} className="rotate-180" /> All events
      </Link>
      <header className="mt-8 grid gap-6 border-b border-line-strong pb-10 sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-8">
        <DateBlock iso={e.startsAt} size="lg" />
        <div className="min-w-0">
          <h1 className="display text-[clamp(2rem,5vw,3.25rem)]">{e.title}</h1>
          <p className="mt-4 flex flex-wrap items-center gap-2">
            <KindTag kind={e.kind} />
            {e.status === "cancelled" && <Badge variant="destructive">Cancelled</Badge>}
          </p>
          <p className="mt-4 max-w-[60ch] text-xl leading-snug text-ink-2">{e.summary}</p>
        </div>
      </header>

      <div className="mt-12 grid gap-x-16 gap-y-12 lg:grid-cols-[minmax(0,1fr)_22rem]">
        <div className="min-w-0 space-y-14">
          {e.coverUrl && <img src={e.coverUrl} alt="" className="aspect-[16/9] w-full rounded-[4px] border border-line object-cover" />}
          <div className="prose text-[1.0625rem]" dangerouslySetInnerHTML={{ __html: highlightHtml(e.bodyHtml) }} />
          {over && e.recapHtml && (
            <section aria-labelledby="recap" className="border-t border-line-strong pt-6">
              <h2 id="recap" className="mb-4 text-xl font-semibold">How it went</h2>
              <div className="prose" dangerouslySetInnerHTML={{ __html: e.recapHtml }} />
            </section>
          )}
          {e.photos.length > 0 && (
            <section aria-labelledby="photos">
              <h2 id="photos" className="mb-4 text-xl font-semibold">Photos</h2>
              <ul className="grid grid-cols-2 gap-3 md:grid-cols-3">
                {e.photos.map((ph) => (
                  <li key={ph.url}>
                    <figure>
                      <img src={ph.url} alt={ph.caption} loading="lazy" className="aspect-[4/3] w-full rounded-[4px] object-cover" />
                      <figcaption className="meta mt-1.5 text-ink-3">{ph.caption}</figcaption>
                    </figure>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>

        <aside className="space-y-10 lg:sticky lg:top-6 lg:self-start">
          <Card role="region" aria-labelledby="details">
            <CardHeader>
              <CardTitle id="details">Details</CardTitle>
            </CardHeader>
            <dl>
              <div className="border-b border-line px-5 py-3">
                <dt className="tlabel">When</dt>
                <dd className="font-medium">{formatRange(e.startsAt, e.endsAt)}</dd>
              </div>
              {e.venue && (
                <div className="border-b border-line px-5 py-3">
                  <dt className="tlabel">Where</dt>
                  <dd className="font-medium">{e.venue}</dd>
                </div>
              )}
              {e.onlineUrl && (
                <div className="border-b border-line px-5 py-3">
                  <dt className="tlabel">Online</dt>
                  <dd>
                    <a href={e.onlineUrl} className="link font-medium break-all">
                      {e.onlineUrl}
                    </a>
                  </dd>
                </div>
              )}
              {e.registration?.capacity != null && (
                <div className="border-b border-line px-5 py-3">
                  <dt className="tlabel">Capacity</dt>
                  <dd className="font-medium tabular">{e.registration.capacity} seats</dd>
                </div>
              )}
            </dl>
            <CardContent>
              <Rsvp slug={e.slug} registration={e.registration} endsAt={e.endsAt} status={e.status} />
            </CardContent>
          </Card>

          {speakers.length > 0 && (
            <section aria-labelledby="speakers">
              <h2 id="speakers" className="mb-3 text-[0.9375rem] font-semibold">
                {speakers.length > 1 ? "Speakers" : "Speaker"}
              </h2>
              <ul>
                {speakers.map((s) => (
                  <li key={s.handle} className="group relative flex items-center gap-4 border-t border-line py-3">
                    <Avatar handle={s.handle} name={s.name} url={s.avatarUrl} size={48} />
                    <div className="min-w-0">
                      <Link
                        to="/people/$handle"
                        params={{ handle: s.handle }}
                        className="font-semibold text-ink no-underline after:absolute after:inset-0 group-hover:underline group-hover:decoration-primary"
                      >
                        {s.name}
                      </Link>
                      <p className="text-sm text-ink-2">{subtitle(s) || s.headline}</p>
                    </div>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </aside>
      </div>
    </article>
  );
}
