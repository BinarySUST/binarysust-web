import { createFileRoute } from "@tanstack/react-router";
import { PageHead } from "~/components/Page";
import { PeopleGrid } from "~/components/People";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/alumni")({
  loader: async () => (await content.people()).filter((p) => p.status === "alumni"),
  head: () => seo({ title: "Alumni", description: "Where Binary SUST alumni work now.", path: "/alumni" }),
  component: function AlumniPage() {
    const people = Route.useLoaderData();
    return (
      <>
        <PageHead title="Alumni" lede="Former members and committee, and where they ended up." />
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          {people.length ? <PeopleGrid people={people} /> : <p className="text-ink-2">No alumni are listed yet.</p>}
        </div>
      </>
    );
  },
});
