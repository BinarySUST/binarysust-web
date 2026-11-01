import { createFileRoute } from "@tanstack/react-router";
import { PageHead } from "~/components/Page";
import { PostList } from "~/features/blog/components/PostList";
import { TagNav, allTags } from "~/features/blog/components/TagNav";
import { content } from "~/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/blog/")({
  loader: () => content.blog(),
  head: () => seo({ title: "Blog", description: "Notes, tutorials and event write-ups from Binary SUST members.", path: "/blog" }),
  component: function BlogPage() {
    const posts = Route.useLoaderData();
    return (
      <>
        <PageHead
          title="Blog"
          lede={
            <>
              Tutorials, contest editorials and event write-ups by members. Also as an{" "}
              <a href="/rss.xml" className="link">RSS feed</a>.
            </>
          }
        />
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          <TagNav tags={allTags(posts)} />
          {posts.length ? <div className="border-t border-line-strong"><PostList posts={posts} /></div> : <p className="text-ink-2">No posts yet.</p>}
        </div>
      </>
    );
  },
});
