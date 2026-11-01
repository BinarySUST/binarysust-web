import { getRouteApi } from "@tanstack/react-router";
import { Link } from "@tanstack/react-router";
import { Avatar } from "~/components/Avatar";
import { Icon } from "~/components/Icon";
import { STATUS_LABEL } from "~/features/people/components/PeopleGrid";
import { formatShort } from "~/lib/format";
import type { SocialKind } from "~/content/types";
import { buttonVariants } from "@binarysust/ui/components/button";
import { Badge, badgeVariants } from "@binarysust/ui/components/badge";

const route = getRouteApi("/people/$handle");

const SOCIAL: Record<SocialKind, string> = {
  github: "GitHub",
  linkedin: "LinkedIn",
  x: "X",
  website: "Website",
  codeforces: "Codeforces",
  leetcode: "LeetCode",
  other: "Link",
};

/** The host of a link, for showing beside its label: github.com/ada rather than the full URL. */
function host(url: string): string {
  try {
    const u = new URL(url);
    const path = u.pathname.replace(/\/$/, "");
    return u.hostname.replace(/^www\./, "") + (path && path !== "/" ? path : "");
  } catch {
    return url;
  }
}

export function PortfolioPage() {
  const p = route.useLoaderData();
  const working = p.status === "alumni" || p.status === "advisor" || p.status === "guest";
  const facts: { label: string; value: string }[] = [
    { label: "Status", value: STATUS_LABEL[p.status] },
    ...(working && p.jobTitle ? [{ label: "Role", value: p.jobTitle }] : []),
    ...(working && p.workplace ? [{ label: "Organisation", value: p.workplace }] : []),
    ...(p.department ? [{ label: "Department", value: p.department }] : []),
    ...(p.batch ? [{ label: "Batch", value: String(p.batch) }] : []),
    ...(p.graduatedOn ? [{ label: "Graduated", value: formatShort(p.graduatedOn) }] : []),
  ];

  return (
    <article className="mx-auto max-w-6xl px-4 pt-12 sm:px-6 sm:pt-16">
      <header className="grid gap-x-10 gap-y-6 sm:grid-cols-[auto_minmax(0,1fr)]">
        <Avatar handle={p.handle} name={p.name} url={p.avatarUrl} size={144} />
        <div className="min-w-0 self-end">
          <h1 className="display text-[clamp(2.25rem,5.5vw,3.5rem)]">{p.name}</h1>
          <p className="mt-2 flex flex-wrap items-center gap-3">
            <span className="meta text-primary-ink">@{p.handle}</span>
            {(p as { lifetimeMember?: boolean }).lifetimeMember === true && <Badge>Lifetime member</Badge>}
          </p>
          <p className="mt-4 max-w-[48ch] text-xl leading-snug text-ink-2">{p.headline}</p>
          <div className="no-print mt-6 flex flex-wrap items-center gap-2">
            {p.cvUrl && (
              <a href={p.cvUrl} className={buttonVariants({ size: "sm" })}>
                <Icon name="file" size={16} /> Download CV
              </a>
            )}
            {p.socials.map((s) => (
              <a key={s.url + s.kind} href={s.url} rel="me noopener" className={buttonVariants({ variant: "outline", size: "sm" })}>
                {SOCIAL[s.kind]} <Icon name="external" size={14} className="text-ink-3" />
              </a>
            ))}
          </div>
        </div>
      </header>

      <dl className="mt-10 grid grid-cols-2 border-t border-line-strong sm:grid-cols-3 lg:grid-cols-6">
        {facts.map((f) => (
          <div key={f.label} className="border-b border-line py-3 pr-4">
            <dt className="tlabel">{f.label}</dt>
            <dd className="font-medium">{f.value}</dd>
          </div>
        ))}
      </dl>

      <div className="mt-14 grid gap-x-16 gap-y-14 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <div className="min-w-0 space-y-14">
          {p.bioHtml && (
            <section aria-labelledby="bio">
              <h2 id="bio" className="mb-4 text-lg font-semibold">About</h2>
              <div className="prose text-[1.0625rem]" dangerouslySetInnerHTML={{ __html: p.bioHtml }} />
            </section>
          )}
          {p.projects.length > 0 && (
            <section aria-labelledby="projects">
              <h2 id="projects" className="mb-2 text-lg font-semibold">Projects</h2>
              <ul>
                {p.projects.map((pr) => (
                  <li key={pr.title} className="border-b border-line py-5 first:border-t">
                    <div className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1">
                      <h3 className="text-[1.1875rem] font-semibold">{pr.title}</h3>
                      <p className="flex gap-4 text-sm">
                        {pr.url && (
                          <a href={pr.url} className="link inline-flex items-center gap-1 font-medium">
                            Live <Icon name="external" size={14} />
                          </a>
                        )}
                        {pr.repoUrl && (
                          <a href={pr.repoUrl} className="link inline-flex items-center gap-1 font-medium">
                            Source <Icon name="code" size={14} />
                          </a>
                        )}
                      </p>
                    </div>
                    <p className="mt-1.5 max-w-[65ch] text-ink-2">{pr.summary}</p>
                    {pr.tags.length > 0 && (
                      <p className="meta mt-2.5 flex flex-wrap gap-x-3 text-xs text-ink-3">
                        {pr.tags.map((t) => (
                          <span key={t}>{t}</span>
                        ))}
                      </p>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}
          {p.achievements.length > 0 && (
            <section aria-labelledby="achievements">
              <h2 id="achievements" className="mb-2 text-lg font-semibold">Achievements</h2>
              <ol>
                {p.achievements.map((a) => (
                  <li key={a.title} className="grid gap-x-6 gap-y-1 border-b border-line py-4 first:border-t sm:grid-cols-[8.5rem_minmax(0,1fr)]">
                    <p className="meta pt-0.5 text-ink-3">{a.date ? formatShort(a.date) : ""}</p>
                    <div>
                      <p className="font-semibold">{a.title}</p>
                      {a.detail && <p className="mt-0.5 text-ink-2">{a.detail}</p>}
                    </div>
                  </li>
                ))}
              </ol>
            </section>
          )}
        </div>

        <aside className="space-y-10">
          {p.skills.length > 0 && (
            <section aria-labelledby="skills">
              <h2 id="skills" className="mb-3 text-lg font-semibold">Skills</h2>
              <ul className="flex flex-wrap gap-2">
                {p.skills.map((s) => (
                  <li key={s}>
                    <a href={`/people?skill=${encodeURIComponent(s)}`} className={badgeVariants({ className: "min-h-8 px-2.5 text-[0.8125rem]" })}>
                      {s}
                    </a>
                  </li>
                ))}
              </ul>
            </section>
          )}
          {p.positions.length > 0 && (
            <section aria-labelledby="positions">
              <h2 id="positions" className="mb-2 text-lg font-semibold">Binary SUST committee</h2>
              <ul>
                {p.positions.map((pos) => (
                  <li key={pos.term + pos.title} className="flex items-baseline justify-between gap-4 border-b border-line py-2.5">
                    <span className="font-medium">{pos.title}</span>
                    <Link to="/committee/$term" params={{ term: pos.term }} className="link meta text-ink-2">
                      {pos.term}
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          )}
          {p.socials.length > 0 && (
            <section aria-labelledby="links" className="hidden print:block">
              <h2 id="links" className="mb-2 text-lg font-semibold">Links</h2>
              <ul className="meta space-y-1">
                {p.socials.map((s) => (
                  <li key={s.url + s.kind}>
                    {SOCIAL[s.kind]}: {host(s.url)}
                  </li>
                ))}
              </ul>
            </section>
          )}
        </aside>
      </div>
    </article>
  );
}
