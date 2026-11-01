import { useSyncExternalStore } from "react";

const noop = () => () => {};
let clientNow: number | undefined;
const getClientNow = () => (clientNow ??= Date.now());

/** The build's clock during pre-render and hydration, then the browser's. Pages pre-rendered days ago
 *  use it to move events that have since ended out of "upcoming". */
export function useNow(): number {
  return useSyncExternalStore(noop, getClientNow, () => __BUILD_TIME__);
}

/** The browser's clock, or null while pre-rendering (for UI that must not guess at build time). */
export function useClientNow(): number | null {
  return useSyncExternalStore<number | null>(noop, getClientNow, () => null);
}

/** A browser-only value, `fallback` during pre-render and hydration. */
export function useClientValue<T>(read: () => T, fallback: T): T {
  return useSyncExternalStore(noop, read, () => fallback);
}

export function useUpcoming<T extends { endsAt: string }>(events: T[]): T[] {
  const now = useNow();
  return events.filter((e) => Date.parse(e.endsAt) >= now);
}
