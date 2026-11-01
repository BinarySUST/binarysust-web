// The Go API, mocked: tests run against the static build only.
import { readFileSync } from "node:fs";
import type { Me, Permission, Schemas } from "@binarysust/api-client";
import type { Page, Route } from "@playwright/test";

/** A slice the build wrote to dist/client/data (from the snapshot, or the committed fixture). */
export function slice<T>(name: string): T {
  return JSON.parse(readFileSync(new URL(`../dist/client/data/${name}.json`, import.meta.url), "utf8")) as T;
}

const json = (route: Route, status: number, body: unknown) =>
  route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });

/** Mocks /healthz and /api/v1/*. `me` signs someone in; without it the API answers as for a visitor. */
export async function mockApi(page: Page, me?: { permissions: Permission[]; roles?: string[] }) {
  await page.route("**/healthz", (r) => json(r, 200, { status: "ok" }));
  await page.route("**/api/v1/**", (route) => {
    const path = new URL(route.request().url()).pathname.replace(/^\/api\/v1/, "");
    if (path === "/auth/me") {
      const body: Schemas["ekman.Me"] = me
        ? { userId: "u-1", permissions: me.permissions, roles: me.roles ?? [] }
        : { userId: "", permissions: [], roles: [] };
      return json(route, 200, body);
    }
    if (path === "/me" && me) {
      const account: Me["account"] = {
        id: "a-1",
        email: "member@example.com",
        name: "Test Member",
        avatarUrl: null,
        personHandle: null,
        personId: null,
      };
      return json(route, 200, account);
    }
    if (path === "/site/status") {
      const status: Schemas["site.Status"] = { contentChangedAt: "2026-10-30T14:12:00Z", live: null, state: "live" };
      return json(route, 200, status);
    }
    return json(route, me ? 404 : 401, { error: { code: "not_found", message: "not mocked", requestId: "e2e" } });
  });
}
