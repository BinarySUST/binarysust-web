import { createFileRoute, notFound } from "@tanstack/react-router";
import { CommitteeView } from "~/features/committee/components/CommitteeView";
import { PageHead } from "~/components/Page";
import { content } from "~/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/committee/")({
  loader: async () => {
    const { currentTerm } = await content.landing();
    if (!currentTerm) throw notFound();
    return content.committee(currentTerm.slug);
  },
  head: ({ loaderData }) =>
    seo({
      title: "Committee",
      description: `The ${loaderData?.term.label ?? "current"} committee of Binary SUST, and every committee before it.`,
      path: "/committee",
    }),
  component: function CommitteePage() {
    const data = Route.useLoaderData();
    return (
      <>
        <PageHead title={`Committee ${data.term.label}`} lede="The students who run Binary SUST this term, and the faculty who advise them." />
        <CommitteeView data={data} />
      </>
    );
  },
});
