import { createFileRoute } from "@tanstack/react-router";
import { CommitteeView } from "~/features/committee/components/CommitteeView";
import { PageHead } from "~/components/Page";
import { content } from "~/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/committee/")({
  loader: async () => {
    const { currentTerm } = await content.landing();
    // A fresh database has no current term yet: show an empty page, not a 404, so the nav link and the
    // pre-render (which crawls it) both keep working.
    if (!currentTerm) return null;
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
    if (!data) {
      return <PageHead title="Committee" lede="The committee for this term has not been announced yet." />;
    }
    return (
      <>
        <PageHead
          title={`Committee ${data.term.label}`}
          lede="The students who run Binary SUST this term, and the faculty who advise them."
        />
        <CommitteeView data={data} />
      </>
    );
  },
});
