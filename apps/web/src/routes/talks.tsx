import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { TalksPage } from "~/features/talks/components/TalksPage";

export const Route = createFileRoute("/talks")({
  loader: () => content.talks(),
  head: () => seo({ title: "Talks", description: "Advisor and guest talks hosted by Binary SUST.", path: "/talks" }),
  component: TalksPage,
});
