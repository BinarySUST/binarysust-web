// Node-only: reads the snapshot from disk. Used at build time by scripts/slices.ts, vite.config.ts
// (prerender page list) and the server half of content.ts. Never reachable from browser code.
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import type { Snapshot } from "./types.ts";

let cached: { file: string; snapshot: Snapshot } | undefined;

/** `.snapshot/site.json` (written by `go run ./cmd/snapshot`) wins; otherwise the committed fixture. */
export function snapshotFile(root = process.cwd()): string {
  const site = resolve(root, ".snapshot/site.json");
  return existsSync(site) ? site : resolve(root, ".snapshot/fixture.json");
}

export function loadSnapshot(root = process.cwd()): Snapshot {
  const file = snapshotFile(root);
  if (cached?.file === file) return cached.snapshot;
  const snapshot = JSON.parse(readFileSync(file, "utf8")) as Snapshot;
  // The fixture is static, so "upcoming" is judged against the time of this build, as cmd/snapshot does.
  if (file.endsWith("fixture.json")) snapshot.version = { ...snapshot.version, builtAt: new Date().toISOString() };
  cached = { file, snapshot };
  return snapshot;
}
