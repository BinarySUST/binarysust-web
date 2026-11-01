// Uploaded media lives outside this site (prod: media.binarysust.org, the binarysust-media Pages project; dev: Caddy at
// media.binarysust.localhost). Snapshot image fields are already absolute URLs; this base is only for
// URLs the dashboard builds itself (upload previews, relative keys from the media endpoint).
export const MEDIA_BASE_URL: string = (import.meta.env.VITE_MEDIA_BASE_URL ?? "https://media.binarysust.org").replace(/\/$/, "");

export function mediaUrl(keyOrUrl: string): string {
  return /^(https?:|data:)/.test(keyOrUrl) ? keyOrUrl : `${MEDIA_BASE_URL}/${keyOrUrl.replace(/^\//, "")}`;
}
