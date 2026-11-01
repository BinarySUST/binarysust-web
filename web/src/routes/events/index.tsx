import { createFileRoute } from "@tanstack/react-router";
import { EventRow } from "~/components/events";
import { PageHead, SectionHead } from "~/components/Page";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";
import { useNow } from "~/lib/useNow";

export const Route = createFileRoute("/events/")({
  loader: () => content.events(),
  head: () =>
    seo({ title: "Events", description: "Workshops, contests, hackathons and meetups run by Binary SUST.", path: "/events" }),
  component: Events,
});

function Events() {
  const events = Route.useLoaderData();
  const now = useNow();
  // The page may be days old; the split is recomputed against the visitor's clock after hydration.
  const upcoming = events.filter((e) => Date.parse(e.endsAt) >= now).sort((a, b) => Date.parse(a.startsAt) - Date.parse(b.startsAt));
  const past = events.filter((e) => Date.parse(e.endsAt) < now);

  return (
    <>
      <PageHead
        title="Events"
        lede="Workshops, contests, hackathons, talks and meetups. Open to every SUST student unless the page says otherwise."
      />
      <div className="mx-auto max-w-7xl space-y-20 px-4 sm:px-6">
        <section aria-labelledby="upcoming">
          <SectionHead id="upcoming" index="01" title="Upcoming" />
          {upcoming.length ? (
            <div className="-mt-8">
              {upcoming.map((e) => (
                <EventRow key={e.slug} e={e} />
              ))}
            </div>
          ) : (
            <p className="border border-dashed border-line-strong p-8 text-ink-2">
              Nothing is scheduled yet. New events are announced here and on the blog.
            </p>
          )}
        </section>
        {past.length > 0 && (
          <section aria-labelledby="past">
            <SectionHead id="past" index={upcoming.length ? "02" : "01"} title="Past events" />
            <div className="-mt-8">
              {past.map((e) => (
                <EventRow key={e.slug} e={e} past />
              ))}
            </div>
          </section>
        )}
      </div>
    </>
  );
}
