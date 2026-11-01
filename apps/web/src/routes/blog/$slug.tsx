import { Link, createFileRoute } from "@tanstack/react-router";
import { Avatar } from "~/components/Avatar";
import { Icon } from "~/components/Icon";
import { SectionHead } from "~/components/Page";
import { PostList } from "~/components/PostList";
import { content } from "~/lib/content";
import { formatFull } from "~/lib/format";
import { highlightHtml } from "~/lib/highlight";
import { seo } from "~/lib/seo";
import { badgeVariants } from "@binarysust/ui/components/badge";

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
  component: Post,
});

function Post() {
  const { post, author, more } = Route.useLoaderData();
  return (
    <>
      <article className="mx-auto max-w-[46rem] px-4 pt-10 sm:px-6 sm:pt-14">
        <Link to="/blog" className="more text-sm text-ink-2">
          <Icon name="arrow" size={16} className="rotate-180" /> The blog
        </Link>
        <header className="mt-8 border-b border-line-strong pb-8">
          <h1 className="display text-[clamp(2rem,5vw,3rem)] leading-[1.08]">{post.title}</h1>
          <p className="mt-5 text-xl leading-snug text-ink-2">{post.excerpt}</p>
          <div className="mt-7 flex flex-wrap items-center gap-x-5 gap-y-3">
            {author && (
              <Link to="/people/$handle" params={{ handle: author.handle }} className="group flex items-center gap-3 text-ink no-underline">
                <Avatar handle={author.handle} name={author.name} url={author.avatarUrl} size={40} />
                <span className="font-semibold group-hover:underline">{author.name}</span>
              </Link>
            )}
            <span className="meta text-ink-3">
              <time dateTime={post.publishedAt}>{formatFull(post.publishedAt)}</time> · {post.readingMinutes} min read
            </span>
          </div>
        </header>
        {post.coverUrl && <img src={post.coverUrl} alt="" className="mt-10 aspect-[16/9] w-full rounded-[4px] border border-line object-cover" />}
        <div className="prose mt-10 text-[1.125rem]" dangerouslySetInnerHTML={{ __html: highlightHtml(post.bodyHtml) }} />
        {post.tags.length > 0 && (
          <ul className="mt-12 flex flex-wrap gap-2 border-t border-line pt-6" aria-label="Tags">
            {post.tags.map((t) => (
              <li key={t}>
                <Link to="/blog/tag/$tag" params={{ tag: t }} className={badgeVariants({ className: "min-h-9 px-3" })}>
                  #{t}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </article>
      {more.length > 0 && (
        <section className="mx-auto max-w-[46rem] px-4 pt-20 sm:px-6" aria-labelledby="more">
          <SectionHead id="more" title="Read next" />
          <div className="-mt-8">
            <PostList posts={more} />
          </div>
        </section>
      )}
    </>
  );
}
