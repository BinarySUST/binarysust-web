// Cloudflare Pages Function: same-origin proxy for /api/* to the Go API on Render.
// The browser only ever talks to the site's own origin, so session cookies are first-party and there is
// no CORS. Method, headers, body and cookies go upstream; status, headers and every Set-Cookie come back.

interface Env {
  API_ORIGIN: string; // e.g. https://binarysust-api.onrender.com (no trailing slash)
}

const HOP_BY_HOP = ["connection", "keep-alive", "proxy-connection", "transfer-encoding", "upgrade", "te", "trailer", "host"];

export const onRequest: PagesFunction<Env> = async ({ request, env }) => {
  if (!env.API_ORIGIN) return new Response("API_ORIGIN is not configured", { status: 500 });

  const url = new URL(request.url);
  const upstream = new URL(url.pathname + url.search, env.API_ORIGIN);

  const headers = new Headers(request.headers);
  for (const h of HOP_BY_HOP) headers.delete(h);
  headers.set("X-Forwarded-Host", url.host);
  headers.set("X-Forwarded-Proto", url.protocol.replace(":", ""));
  const ip = request.headers.get("CF-Connecting-IP");
  if (ip) headers.set("X-Forwarded-For", ip);

  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  let res: Response;
  try {
    res = await fetch(upstream, {
      method: request.method,
      headers,
      body: hasBody ? request.body : undefined,
      redirect: "manual", // OAuth redirects (Location to Google, back to /dashboard) must reach the browser
    });
  } catch {
    return Response.json(
      { title: "API unreachable", status: 502, detail: "The server did not answer. It may be waking up; try again in a minute." },
      { status: 502 },
    );
  }

  const out = new Headers();
  res.headers.forEach((v, k) => {
    if (k !== "set-cookie" && !HOP_BY_HOP.includes(k)) out.set(k, v);
  });
  // Headers.forEach folds multiple Set-Cookie headers into one; getSetCookie keeps them separate.
  for (const c of res.headers.getSetCookie()) out.append("set-cookie", stripDomain(c));
  out.set("cache-control", res.headers.get("cache-control") ?? "no-store");

  return new Response(res.body, { status: res.status, statusText: res.statusText, headers: out });
};

/** Cookies set by the API for its own host must land on the site's host instead. */
function stripDomain(cookie: string): string {
  return cookie.replace(/;\s*domain=[^;]*/i, "");
}
