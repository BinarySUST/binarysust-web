import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { SponsorsPage } from "~/features/sponsors/components/SponsorsPage";

export const Route = createFileRoute("/sponsors")({
  loader: () => content.sponsors(),
  head: () => seo({ title: "Sponsors", description: "The companies and partners that back Binary SUST.", path: "/sponsors" }),
  component: SponsorsPage,
});
