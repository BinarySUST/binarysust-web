import { Link } from "@tanstack/react-router";
import { formatShort } from "~/lib/format";
import type { PostCard } from "~/content/slices";

export function PostList({ posts }: { posts: PostCard[] }) {
  return (
    <ol>
      {posts.map((p) => (
        <li key={p.slug} className="group relative grid gap-2 border-b border-line py-7 md:grid-cols-[11rem_minmax(0,1fr)] md:gap-10">
          <p className="meta text-ink-3 md:pt-1.5">
            <time dateTime={p.publishedAt}>{formatShort(p.publishedAt)}</time>
            <span className="md:mt-1 md:block">
              <span className="md:hidden"> · </span>
              {p.readingMinutes} min read
            </span>
          </p>
          <div>
            <h3 className="text-xl leading-snug font-semibold sm:text-[1.5rem]">
              <Link
                to="/blog/$slug"
                params={{ slug: p.slug }}
                className="text-ink no-underline after:absolute after:inset-0 group-hover:underline group-hover:decoration-primary"
              >
                {p.title}
              </Link>
            </h3>
            <p className="mt-2 max-w-[65ch] text-ink-2">{p.excerpt}</p>
            {(p.author || p.tags.length > 0) && (
              <p className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-ink-3">
                {p.author && <span>{p.author.name}</span>}
                {p.tags.map((t) => (
                  <span key={t} className="meta">
                    #{t}
                  </span>
                ))}
              </p>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}
