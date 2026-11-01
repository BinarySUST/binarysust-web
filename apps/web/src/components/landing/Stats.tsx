import type { Activity } from "~/content/slices";
import type { Club } from "~/content/types";
import { ActivityGraph } from "~/components/landing/ActivityGraph";

const compact = new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 });
/** 312 stays 312; 6500 reads 6.5K. */
const show = (n: number) => (n >= 10_000 || (n >= 1000 && n % 100 === 0) ? compact.format(n) : n.toLocaleString("en-US"));

/** "1 events run" reads wrong; singularise the label's first word when the value is 1 (the snapshot has one label). */
const label = (value: number, text: string) =>
  value === 1 && !/^\w*(ss|ses|xes|ches|shes)\b/.test(text) ? text.replace(/^(\w+)s\b/, "$1") : text;

/** The club's numbers, each read as a phrase ("312 members"), so no figure stands without its meaning. */
export function StatList({ stats }: { stats: Club["stats"] }) {
  return (
    <dl className="-mt-8 grid grid-cols-2">
      {stats.map((s, i) => (
        <div key={s.label} className={`flex flex-col-reverse justify-end gap-1.5 border-b border-line py-5 ${i % 2 === 1 ? "border-l pl-5" : "pr-4"}`}>
          <dt className="text-[0.9375rem] leading-snug text-ink-2">{label(s.value, s.label)}</dt>
          <dd className="display text-[clamp(1.75rem,3.5vw,2.25rem)] leading-none tabular">{show(s.value)}</dd>
        </div>
      ))}
    </dl>
  );
}

/** Numbers beside the activity graph: what the club is, and what it has been doing. */
export function Stats({ stats, activity, builtAt }: { stats: Club["stats"]; activity?: Activity[]; builtAt?: string }) {
  if (stats.length === 0 && !activity) return null;
  return (
    <div className="grid gap-x-14 gap-y-10 lg:grid-cols-[minmax(0,22rem)_minmax(0,1fr)]">
      {stats.length > 0 && <StatList stats={stats} />}
      {activity && builtAt && (
        <div className="min-w-0 lg:pt-1">
          <ActivityGraph activity={activity} builtAt={builtAt} />
        </div>
      )}
    </div>
  );
}
