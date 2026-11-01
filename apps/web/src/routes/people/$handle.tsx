import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { PortfolioPage } from "~/features/portfolio/components/PortfolioPage";

export const Route = createFileRoute("/people/$handle")({
  loader: ({ params }) => content.person(params.handle),
  head: ({ loaderData: p, params }) =>
    seo({
      title: p?.name ?? params.handle,
      description: p ? `${p.name}: ${p.headline}` : "A Binary SUST portfolio.",
      path: `/people/${params.handle}`,
      type: "profile",
      image: p?.avatarUrl,
    }),
  component: PortfolioPage,
});
