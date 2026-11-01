import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { PeoplePage } from "~/features/people/components/PeoplePage";

export const Route = createFileRoute("/people/")({
  loader: () => content.people(),
  head: () =>
    seo({ title: "People", description: "Members, committee, advisors and alumni of Binary SUST, with their portfolios.", path: "/people" }),
  component: PeoplePage,
});
