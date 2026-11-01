// The only way browser code talks to the Go API. Same origin: /api/* is proxied by the Pages
// Function in production and by Caddy (or Vite) in development, so cookies just work.
import createClient, { type Middleware } from "openapi-fetch";
import { useSyncExternalStore } from "react";
import type { MediaPurpose } from "./enums";
import type { components, paths } from "./schema";

export type { MediaPurpose };

export type Schemas = components["schemas"];
export type Permission = Schemas["Permission"];
export type Account = Schemas["access.Account"];
/** Who is signed in: permissions and roles from GET /auth/me, name/handle/avatar from GET /me. */
export type Me = Schemas["ekman.Me"] & { account: Account };

/** Paths in the generated schema already carry this prefix; it is kept for links built by hand. */
export const API_BASE = "/api/v1";

// --- "Waking the server" ---------------------------------------------------------------------
// Render's free tier sleeps; the first request after a quiet spell can take up to a minute.
// Any request still pending after SLOW_MS flips this flag so the UI can say what is happening.
const SLOW_MS = 2500;
let pending = 0;
let waking = false;
const listeners = new Set<() => void>();
const emit = () => listeners.forEach((l) => l());
const setWaking = (v: boolean) => {
  if (waking !== v) {
    waking = v;
    emit();
  }
};
const timers = new Map<Request, ReturnType<typeof setTimeout>>();

function track(req: Request) {
  pending++;
  timers.set(req, setTimeout(() => setWaking(true), SLOW_MS));
}
function untrack(req: Request) {
  clearTimeout(timers.get(req));
  timers.delete(req);
  pending = Math.max(0, pending - 1);
  if (pending === 0) setWaking(false);
}

const wakeMiddleware: Middleware = {
  onRequest({ request }) {
    track(request);
    return request;
  },
  onResponse({ request, response }) {
    untrack(request);
    return response;
  },
  onError({ request }) {
    untrack(request);
  },
};

export function useWaking(): boolean {
  return useSyncExternalStore(
    (l) => {
      listeners.add(l);
      return () => listeners.delete(l);
    },
    () => waking,
    () => false,
  );
}

export const api = createClient<paths>({ baseUrl: "", credentials: "same-origin" });
api.use(wakeMiddleware);

/** Fire-and-forget ping so Render starts waking before the user clicks anything. */
let warmed = false;
export function warmApi() {
  if (warmed || typeof window === "undefined") return;
  warmed = true;
  const req = new Request("/healthz", { cache: "no-store" });
  track(req);
  fetch(req)
    .catch(() => undefined)
    .finally(() => untrack(req));
}

/** ekman sets a readable `ekman_hint` cookie next to its HttpOnly session cookie. */
export function hasSessionHint(): boolean {
  return typeof document !== "undefined" && /(?:^|;\s*)ekman_hint=/.test(document.cookie);
}

export function loginUrl(next: string) {
  return `/login?next=${encodeURIComponent(next)}`;
}

export function googleStartUrl(next: string) {
  return `${API_BASE}/auth/google/start?next=${encodeURIComponent(next)}`;
}

/**
 * ekman's error envelope: `{"error":{"code","message","fields","requestId"}}`. A 422 carries
 * `fields` (input name -> problem), which forms show next to the input they belong to.
 */
export type ErrorBody = Schemas["Error"]["error"];

export class ApiError extends Error {
  readonly code: string;
  readonly fields: Record<string, string>;
  readonly requestId: string | undefined;
  constructor(
    public status: number,
    body: Partial<ErrorBody> | undefined,
  ) {
    super(body?.message ? sentence(body.message) : fallbackMessage(status));
    this.code = body?.code ?? "http_" + status;
    this.fields = body?.fields ?? {};
    this.requestId = body?.requestId;
  }
}

const sentence = (s: string) => s.charAt(0).toUpperCase() + s.slice(1) + (/[.!?]$/.test(s) ? "" : ".");

function fallbackMessage(status: number): string {
  if (status === 401) return "Your session has ended. Sign in again.";
  if (status === 403) return "Your roles do not allow this.";
  if (status === 404) return "Not found. It may have been deleted.";
  if (status >= 500) return `The server could not finish that (${status}). Try again in a moment.`;
  return `Request failed (${status}).`;
}

/** Pulls ekman's envelope out of whatever openapi-fetch parsed (JSON object, or text from a proxy). */
export function toApiError(status: number, error: unknown): ApiError {
  const body = typeof error === "object" && error !== null && "error" in error ? (error as { error: Partial<ErrorBody> }).error : undefined;
  return new ApiError(status, body);
}

/** Throw on API errors so TanStack Query sees them. 204 responses resolve to undefined. */
export function unwrap<T>(r: { data?: T; error?: unknown; response: Response }): T {
  if (r.error !== undefined || !r.response.ok) throw toApiError(r.response.status, r.error);
  return r.data as T;
}

// --- Media ----------------------------------------------------------------------------------
// POST /api/v1/media takes a multipart body, which openapi-fetch does not build, so this helper
// sends it by hand. Its purpose and result are typed from the generated schema.

/** Upload purposes (api/internal/media Purposes) and the permission each needs. */
export const MEDIA_PURPOSES = {
  avatar: { label: "Avatar", requires: "profile.edit_own", accept: "image/*" },
  cv: { label: "CV (PDF)", requires: "profile.edit_own", accept: "application/pdf" },
  post: { label: "Blog image", requires: "blog.write", accept: "image/*" },
  event: { label: "Event cover or photo", requires: "events.manage", accept: "image/*" },
  talk: { label: "Talk image", requires: "talks.manage", accept: "image/*" },
  logo: { label: "Sponsor logo", requires: "sponsors.manage", accept: "image/*" },
  file: { label: "General file", requires: "media.upload", accept: "image/*,application/pdf" },
} as const satisfies Record<MediaPurpose, { label: string; requires: Permission; accept: string }>;

export type MediaObject = Schemas["media.Object"];

export async function uploadMedia(file: Blob, purpose: MediaPurpose, name = "upload"): Promise<MediaObject> {
  const form = new FormData();
  form.append("file", file, name);
  const query: NonNullable<paths["/api/v1/media"]["post"]["parameters"]["query"]> = { purpose };
  const req = new Request(`${API_BASE}/media?${new URLSearchParams(query as Record<string, string>)}`, {
    method: "POST",
    body: form,
    credentials: "same-origin",
  });
  track(req);
  try {
    const res = await fetch(req);
    const body: unknown = await res.json().catch(() => undefined);
    if (!res.ok) throw toApiError(res.status, body);
    return body as MediaObject;
  } finally {
    untrack(req);
  }
}
