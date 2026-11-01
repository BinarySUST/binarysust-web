import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Icon } from "~/components/Icon";
import { api, unwrap } from "@binarysust/api-client";
import { formatShort, formatTime } from "~/lib/format";
import { useMe } from "~/dashboard/me";
import { visibleSections } from "~/dashboard/sections";
import { ErrorBox, SectionHead } from "~/dashboard/ui";

/**
 * "Is my change live?": GET /site/status compares the API's last content change with the deployed
 * site's /version.json and answers "live" or "publishing". Any successful content write
 * invalidates ["site-status"], so this flips right after a save.
 */
function PublishStatus() {
  const status = useQuery({
    queryKey: ["site-status"],
    queryFn: async () => unwrap(await api.GET("/api/v1/site/status")),
    refetchInterval: (q) => (q.state.data?.state === "publishing" ? 15_000 : 60_000),
  });

  if (status.error) return <ErrorBox error={status.error} retry={() => void status.refetch()} />;
  if (!status.data) return <p className="text-ink-2" role="status">Checking publish status…</p>;

  const { state, contentChangedAt, live } = status.data;
  const isLive = state === "live";
  const at = (iso: string) => `${formatShort(iso)} ${formatTime(iso)}`;
  return (
    <div className="flex flex-wrap items-center gap-4" role="status" aria-live="polite">
      <span
        className={`inline-flex items-center gap-2 rounded-full px-4 py-1.5 font-semibold ${isLive ? "bg-parrot/15 text-ink" : "bg-yellow/20 text-ink"}`}
        data-state={state}
      >
        <span className={`size-2.5 rounded-full ${isLive ? "bg-parrot" : "animate-pulse bg-yellow"}`} aria-hidden="true" />
        {isLive ? "Live" : "Publishing…"}
      </span>
      <span className="text-sm text-ink-2">
        {isLive
          ? `Every change up to ${at(contentChangedAt)} is on the public site.`
          : live
            ? `Your change from ${at(contentChangedAt)} is being built; the site shows content up to ${at(live.contentChangedAt)}. It updates in 2 to 4 minutes.`
            : "Could not read the deployed site's version, so we can't confirm your latest change is live yet."}
      </span>
    </div>
  );
}

export function Overview() {
  const me = useMe();
  const sections = visibleSections(me).filter((s) => s.path !== "");
  return (
    <>
      <SectionHead title={`Hello${me.account.name ? `, ${me.account.name.split(" ")[0]}` : ""}`} />
      <section className="rounded-xl border border-line bg-panel p-6" aria-labelledby="publish">
        <h2 id="publish" className="mb-4 text-lg font-semibold">Public site</h2>
        <PublishStatus />
      </section>
      <section className="mt-8" aria-labelledby="yours">
        <h2 id="yours" className="mb-4 text-lg font-semibold">What you can do</h2>
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {sections.map((s) => (
            <li key={s.path}>
              <Link
                to="/dashboard/$"
                params={{ _splat: s.path }}
                className="flex items-center gap-3 rounded-lg border border-line bg-panel p-4 font-semibold text-ink no-underline hover:border-ink-3"
              >
                <Icon name={s.icon} /> {s.title}
              </Link>
            </li>
          ))}
        </ul>
        {me.roles.length > 0 && <p className="mt-6 text-sm text-ink-3">Your roles: {me.roles.join(", ")}.</p>}
      </section>
    </>
  );
}
