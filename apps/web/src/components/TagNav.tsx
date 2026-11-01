import { Link } from "@tanstack/react-router";

export function TagNav({ tags, active }: { tags: string[]; active?: string }) {
  return (
    <nav aria-label="Tags" className="mb-6">
      <ul className="flex flex-wrap gap-2">
        <li>
          <Link to="/blog" className="tag min-h-9 px-3" aria-current={active ? undefined : "page"}>
            All posts
          </Link>
        </li>
        {tags.map((t) => (
          <li key={t}>
            <Link to="/blog/tag/$tag" params={{ tag: t }} aria-current={t === active ? "page" : undefined} className="tag min-h-9 px-3">
              #{t}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}

export const allTags = (posts: { tags: string[] }[]) => [...new Set(posts.flatMap((p) => p.tags))].sort();
