import type { Schemas } from "./types";

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
