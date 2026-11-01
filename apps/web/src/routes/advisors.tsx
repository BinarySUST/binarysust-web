import { createFileRoute } from "@tanstack/react-router";
import { PageHead } from "~/components/Page";
import { PeopleGrid } from "~/components/People";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/advisors")({
  loader: async () => (await content.people()).filter((p) => p.status === "advisor"),
  head: () => seo({ title: "Advisors", description: "Faculty advisors of Binary SUST.", path: "/advisors" }),
  component: function AdvisorsPage() {
    const people = Route.useLoaderData();
    return (
      <>
        <PageHead title="Advisors" lede="Faculty members who advise the club and its committee." />
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {people.length ? <PeopleGrid people={people} /> : <p className="text-ink-2">No advisors are listed yet.</p>}
        </div>
      </>
    );
  },
});
