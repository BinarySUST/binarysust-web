import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { EventsPage } from "~/features/events/components/EventsPage";

export const Route = createFileRoute("/events/")({
  loader: () => content.events(),
  head: () =>
    seo({ title: "Events", description: "Workshops, contests, hackathons and meetups run by Binary SUST.", path: "/events" }),
  component: EventsPage,
});
