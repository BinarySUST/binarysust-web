import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { EventPage } from "~/features/events/components/EventPage";

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
