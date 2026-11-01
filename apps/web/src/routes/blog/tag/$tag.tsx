import { createFileRoute, notFound } from "@tanstack/react-router";
import { PageHead } from "~/components/Page";
import { PostList } from "~/features/blog/components/PostList";
import { TagNav, allTags } from "~/features/blog/components/TagNav";
import { content } from "~/content";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/blog/tag/$tag")({
  loader: async ({ params }) => {
    const all = await content.blog();
    const posts = all.filter((p) => p.tags.includes(params.tag));
    if (!posts.length) throw notFound();
    return { posts, tags: allTags(all) };
  },
  head: ({ params }) =>
    seo({ title: `#${params.tag}`, description: `Binary SUST posts tagged ${params.tag}.`, path: `/blog/tag/${params.tag}` }),
  component: function TagPage() {
    const { posts, tags } = Route.useLoaderData();
    const { tag } = Route.useParams();
    return (
      <>
        <PageHead title={`#${tag}`} lede={`${posts.length} ${posts.length === 1 ? "post" : "posts"} tagged ${tag}.`} />
        <div className="mx-auto max-w-7xl px-4 sm:px-6">
          <TagNav tags={tags} active={tag} />
          <div className="border-t border-line-strong">
            <PostList posts={posts} />
          </div>
        </div>
      </>
    );
  },
});
