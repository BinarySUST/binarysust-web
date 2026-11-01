import { createFileRoute } from "@tanstack/react-router";
import { useMemo, useState, useSyncExternalStore } from "react";
import { PageHead } from "~/components/Page";
import { PeopleGrid, STATUS_LABEL } from "~/components/People";
import { content } from "~/lib/content";
import { seo } from "~/lib/seo";
import { STATUS_ORDER } from "~/lib/slices";
import type { PersonStatus } from "~/lib/types";
import { Button } from "@binarysust/ui/components/button";
import { Input } from "@binarysust/ui/components/input";
import { NativeSelect } from "@binarysust/ui/components/native-select";

export const Route = createFileRoute("/people/")({
  loader: () => content.people(),
  head: () =>
    seo({ title: "People", description: "Members, committee, advisors and alumni of Binary SUST, with their portfolios.", path: "/people" }),
  component: People,
});

// Filters live in the query string (?status=alumni&skill=Go) so they can be shared. The pre-rendered page
// lists everyone; the browser applies the filters after hydration.
const SEARCH_EVENT = "people-search";
function subscribeSearch(cb: () => void) {
  window.addEventListener("popstate", cb);
  window.addEventListener(SEARCH_EVENT, cb);
  return () => {
    window.removeEventListener("popstate", cb);
    window.removeEventListener(SEARCH_EVENT, cb);
  };
}
function writeSearch(patch: Record<string, string>) {
  const sp = new URLSearchParams(window.location.search);
  for (const [k, v] of Object.entries(patch)) {
    if (v) sp.set(k, v);
    else sp.delete(k);
  }
  const qs = sp.toString();
  window.history.replaceState(window.history.state, "", qs ? `?${qs}` : window.location.pathname);
  window.dispatchEvent(new Event(SEARCH_EVENT));
}

function People() {
  const people = Route.useLoaderData();
  const search = useSyncExternalStore(subscribeSearch, () => window.location.search, () => "");
  const sp = new URLSearchParams(search);
  const rawStatus = sp.get("status") ?? "";
  const status: PersonStatus | "all" = (STATUS_ORDER as string[]).includes(rawStatus) ? (rawStatus as PersonStatus) : "all";
  const skill = sp.get("skill") ?? "";
  const setStatus = (s: PersonStatus | "all") => writeSearch({ status: s === "all" ? "" : s });
  const setSkill = (s: string) => writeSearch({ skill: s });
  const [q, setQ] = useState("");

  const skills = useMemo(() => [...new Set(people.flatMap((p) => p.skills))].sort((a, b) => a.localeCompare(b)), [people]);
  const statuses = STATUS_ORDER.filter((s) => people.some((p) => p.status === s));
  const needle = q.trim().toLowerCase();
  const shown = people.filter(
    (p) =>
      (status === "all" || p.status === status) &&
      (!skill || p.skills.includes(skill)) &&
      (!needle || `${p.name} ${p.headline} ${p.workplace}`.toLowerCase().includes(needle)),
  );

  return (
    <>
      <PageHead title="People" lede="Everyone with a public page: the committee, members, advisors and alumni." />
      <div className="mx-auto max-w-7xl px-4 sm:px-6">
        <form className="mb-10 grid gap-6 border-y border-line py-6 md:grid-cols-[1fr_auto]" role="search" onSubmit={(e) => e.preventDefault()}>
          <fieldset>
            <legend className="tlabel mb-2">Status</legend>
            <div className="flex flex-wrap gap-2">
              {(["all", ...statuses] as const).map((s) => (
                <label key={s} className="cursor-pointer">
                  <input type="radio" name="status" value={s} checked={status === s} onChange={() => setStatus(s)} className="peer sr-only" />
                  <span className="inline-flex min-h-10 items-center rounded-[4px] border border-line-strong px-3.5 text-sm font-medium text-ink-2 hover:border-ink-2 hover:text-ink peer-checked:border-ink peer-checked:bg-ink peer-checked:text-ground peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-[var(--focus)]">
                    {s === "all" ? "Everyone" : STATUS_LABEL[s]}
                  </span>
                </label>
              ))}
            </div>
          </fieldset>
          <div className="grid gap-4 sm:grid-cols-2 md:w-[28rem]">
            <label className="block">
              <span className="tlabel mb-2">Skill</span>
              <NativeSelect value={skill} onChange={(e) => setSkill(e.target.value)}>
                <option value="">Any skill</option>
                {skills.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </NativeSelect>
            </label>
            <label className="block">
              <span className="tlabel mb-2">Name or company</span>
              <Input type="search" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search people" />
            </label>
          </div>
        </form>
        <p className="meta mb-3 text-ink-3" aria-live="polite">
          Showing {shown.length} of {people.length}
        </p>
        {shown.length > 0 ? (
          <PeopleGrid people={shown} />
        ) : (
          <div className="border border-dashed border-line-strong p-10 text-center">
            <p className="font-semibold">Nobody matches those filters.</p>
            <Button type="button" variant="outline" className="mt-4" onClick={() => (writeSearch({ status: "", skill: "" }), setQ(""))}>
              Clear filters
            </Button>
          </div>
        )}
      </div>
    </>
  );
}
