// Named views over the generated schema (src/schema.d.ts, from apps/api/openapi.json by `pnpm gen`).
import type { components, paths } from "./schema";

export type { components, paths };
export type Schemas = components["schemas"];
export type Permission = Schemas["Permission"];
export type Account = Schemas["access.Account"];
/** Who is signed in: permissions and roles from GET /auth/me, name/handle/avatar from GET /me. */
export type Me = Schemas["ekman.Me"] & { account: Account };
