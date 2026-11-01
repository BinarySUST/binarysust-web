// The one way a public page gets content. Every loader calls `content.*`.
//   Pre-render (server half): slice the snapshot read from disk, with the same code that writes public/data.
//   Browser (client half):    fetch that slice as static JSON from /data/<name>.json.
// The full snapshot never reaches client JavaScript, and no public page ever calls the API.
import { createIsomorphicFn } from "@tanstack/react-start";
import { notFound } from "@tanstack/react-router";
import { slices } from "./slices";
import type { Snapshot } from "./types";

async function fetchSlice<T>(name: string): Promise<T | null> {
  const res = await fetch(`/data/${name}.json`);
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`Could not load /data/${name}.json (${res.status})`);
  return (await res.json()) as T;
}

function slice<A extends string[], T>(pick: (s: Snapshot, ...args: A) => T | null, name: (...args: A) => string) {
  const get = createIsomorphicFn()
    .server(async (...args: A): Promise<T | null> => {
      const { loadSnapshot } = await import("./snapshot.server");
      return pick(loadSnapshot(), ...args);
    })
    .client((...args: A): Promise<T | null> => fetchSlice<T>(name(...args)));

  return async (...args: A): Promise<T> => {
    const value = await get(...args);
    if (value == null) throw notFound();
    return value;
  };
}

const safe = (s: string) => encodeURIComponent(s);

export const content = {
  landing: slice(slices.index, () => "index"),
  people: slice(slices.people, () => "people"),
  person: slice(slices.person, (handle: string) => `people/${safe(handle)}`),
  committee: slice(slices.committee, (term: string) => `committee/${safe(term)}`),
  events: slice(slices.events, () => "events"),
  event: slice(slices.event, (slug: string) => `events/${safe(slug)}`),
  blog: slice(slices.blog, () => "blog"),
  post: slice(slices.post, (slug: string) => `blog/${safe(slug)}`),
  talks: slice(slices.talks, () => "talks"),
  sponsors: slice(slices.sponsors, () => "sponsors"),
};
