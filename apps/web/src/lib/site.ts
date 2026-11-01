// Canonical origin for sitemap, RSS and OpenGraph URLs.
export const SITE_URL = ((typeof process !== "undefined" ? process.env.SITE_URL : undefined) ?? "https://binarysust.org").replace(
  /\/$/,
  "",
);
