import { createFileRoute } from "@tanstack/react-router";
import { CommitteeView } from "~/features/committee/components/CommitteeView";
import { PageHead } from "~/components/Page";
import { content } from "~/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/committee/$term")({
  loader: ({ params }) => content.committee(params.term),
  head: ({ loaderData, params }) =>
    seo({
      title: `Committee ${loaderData?.term.label ?? params.term}`,
      description: `Binary SUST committee for the ${loaderData?.term.label ?? params.term} term.`,
      path: `/committee/${params.term}`,
    }),
  component: function TermPage() {
    const data = Route.useLoaderData();
    return (
      <>
        <PageHead
          title={`Committee ${data.term.label}`}
          lede={data.isCurrent ? "The committee serving this term." : `The committee that served in ${data.term.label}.`}
        />
        <CommitteeView data={data} />
      </>
    );
  },
});
