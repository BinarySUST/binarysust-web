import { Link, createFileRoute } from "@tanstack/react-router";
import { Avatar } from "~/components/Avatar";
import { Icon } from "~/components/Icon";
import { PageHead } from "~/components/Page";
import { content } from "~/lib/content";
import { formatFull } from "~/lib/format";
import { seo } from "~/lib/seo";

export const Route = createFileRoute("/talks")({
  loader: () => content.talks(),
  head: () => seo({ title: "Talks", description: "Advisor and guest talks hosted by Binary SUST.", path: "/talks" }),
  component: Talks,
});

function Talks() {
  const talks = Route.useLoaderData();
  return (
    <>
      <PageHead title="Talks" lede="Our advisors and guests on the things a syllabus leaves out: careers, craft and the industry." />
      <ol className="mx-auto max-w-7xl border-t border-line-strong px-4 sm:px-6">
        {talks.map((t) => (
          <li
            key={t.slug}
            id={t.slug}
            className={`grid gap-6 border-b border-line py-10 sm:grid-cols-[auto_minmax(0,1fr)] sm:gap-10 ${t.featured ? "-mx-4 bg-ground-2 px-4 sm:-mx-6 sm:px-6" : ""}`}
          >
            {t.speakerRef ? (
              <Avatar handle={t.speakerRef.handle} name={t.speakerRef.name} url={t.speakerRef.avatarUrl} size={96} />
            ) : (
              <span className="hidden w-24 sm:block" aria-hidden="true" />
            )}
            <div className="max-w-3xl">
              <h2 className="display text-[clamp(1.375rem,3vw,1.875rem)]">{t.title}</h2>
              <p className="mt-3 text-ink-2">
                {t.speakerRef ? (
                  <Link to="/people/$handle" params={{ handle: t.speakerRef.handle }} className="link font-semibold text-ink">
                    {t.speakerRef.name}
                  </Link>
                ) : (
                  <span className="font-semibold text-ink">{t.speaker}</span>
                )}
                {t.speakerRef?.headline && `, ${t.speakerRef.headline}`}
              </p>
              <p className="meta mt-1 text-ink-3">
                <time dateTime={t.date}>{formatFull(t.date)}</time>
                {t.featured && " · Featured"}
              </p>
              <div className="prose mt-4 text-ink-2" dangerouslySetInnerHTML={{ __html: t.summaryHtml }} />
              <div className="mt-5 flex flex-wrap items-center gap-3">
                {t.videoUrl && (
                  <a href={t.videoUrl} className="btn btn-primary btn-sm">
                    <Icon name="play" size={14} /> Watch
                  </a>
                )}
                {t.slidesUrl && (
                  <a href={t.slidesUrl} className="btn btn-ghost btn-sm">
                    <Icon name="file" size={14} /> Slides
                  </a>
                )}
                {!t.videoUrl && !t.slidesUrl && <p className="text-sm text-ink-3">Recording and slides not published yet.</p>}
              </div>
            </div>
          </li>
        ))}
      </ol>
    </>
  );
}
