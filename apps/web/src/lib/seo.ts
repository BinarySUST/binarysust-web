import { SITE_URL } from "./site";

type Seo = { title: string; description: string; path: string; type?: "website" | "article" | "profile"; image?: string | null };

const SITE = "Binary SUST";

/** Per-route <head>: title, description, canonical and OpenGraph/Twitter cards. */
export function seo({ title, description, path, type = "website", image }: Seo) {
  const full = title === SITE ? `${SITE}: the tech club of SUST` : `${title} | ${SITE}`;
  const url = SITE_URL + path;
  return {
    meta: [
      { title: full },
      { name: "description", content: description },
      { property: "og:site_name", content: SITE },
      { property: "og:title", content: title },
      { property: "og:description", content: description },
      { property: "og:type", content: type },
      { property: "og:url", content: url },
      ...(image ? [{ property: "og:image", content: image }] : []),
      { name: "twitter:card", content: image ? "summary_large_image" : "summary" },
      { name: "twitter:title", content: title },
      { name: "twitter:description", content: description },
    ],
    links: [{ rel: "canonical", href: url }],
  };
}
