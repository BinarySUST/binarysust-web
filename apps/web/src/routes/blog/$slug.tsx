import { createFileRoute } from "@tanstack/react-router";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { PostPage } from "~/features/blog/components/PostPage";

export const Route = createFileRoute("/blog/$slug")({
  loader: ({ params }) => content.post(params.slug),
  head: ({ loaderData, params }) => {
    const base = seo({
      title: loaderData?.post.title ?? params.slug,
      description: loaderData?.post.excerpt ?? "A Binary SUST blog post.",
      path: `/blog/${params.slug}`,
      type: "article",
      image: loaderData?.post.coverUrl,
    });
    return {
      ...base,
      meta: [
        ...base.meta,
        ...(loaderData ? [{ property: "article:published_time", content: loaderData.post.publishedAt }] : []),
        ...(loaderData?.post.tags.map((t) => ({ property: "article:tag", content: t })) ?? []),
      ],
    };
  },
  component: PostPage,
});
