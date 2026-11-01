import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { AboutPage } from "~/components/AboutPage";

export const Route = createFileRoute("/about")({
  loader: () => content.landing(),
  head: ({ loaderData }) =>
    seo({ title: "About", description: loaderData?.club.about.slice(0, 160) ?? "About Binary SUST.", path: "/about" }),
  component: AboutPage,
});
