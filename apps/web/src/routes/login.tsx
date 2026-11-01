import { Link, createFileRoute } from "@tanstack/react-router";
import { useEffect } from "react";
import { Icon } from "~/components/Icon";
import { Logo } from "~/components/Logo";
import { API_BASE, googleStartUrl, warmApi } from "@binarysust/api-client";

/** Only same-site paths survive as ?next=, so the sign-in flow can't bounce visitors elsewhere. */
function safeNext(v: unknown): string {
  return typeof v === "string" && v.startsWith("/") && !v.startsWith("//") ? v : "/dashboard";
}

export const Route = createFileRoute("/login")({
  ssr: false, // client-only: served by _shell.html
  validateSearch: (s: Record<string, unknown>) => ({ next: safeNext(s.next) }),
  head: () => ({ meta: [{ title: "Sign in | Binary SUST" }, { name: "robots", content: "noindex" }] }),
  component: Login,
});

function Login() {
  const { next } = Route.useSearch();
  // Client-only route, so window is always there.
  const devHost = /(^|\.)localhost$/.test(window.location.hostname);
  useEffect(() => warmApi(), []);

  return (
    <main id="main" className="relative grid min-h-dvh place-items-center bg-ground px-4 py-12">
      <div className="blueprint pointer-events-none absolute inset-0" aria-hidden="true" />
      <div className="sheet relative w-full max-w-md">
        <div className="border-b border-line px-8 py-5 sm:px-10">
          <Link to="/" className="text-ink no-underline" aria-label="Binary SUST, home">
            <Logo />
          </Link>
        </div>
        <div className="px-8 pt-8 pb-9 sm:px-10">
          <h1 className="display text-[2rem]">Sign in</h1>
          <p className="mt-3 text-ink-2">
            Use your Google account. Any account can sign in to reserve seats; committee members also get the dashboard.
          </p>
          <a href={googleStartUrl(next)} className="btn btn-ghost mt-8 w-full bg-panel">
            <svg width="18" height="18" viewBox="0 0 48 48" aria-hidden="true">
              <path fill="#EA4335" d="M24 9.5c3.5 0 6.6 1.2 9.1 3.6l6.8-6.8C35.8 2.4 30.3 0 24 0 14.6 0 6.6 5.4 2.7 13.3l7.9 6.1C12.5 13.6 17.8 9.5 24 9.5z" />
              <path fill="#4285F4" d="M46.5 24.5c0-1.6-.1-3.1-.4-4.5H24v9h12.7c-.6 3-2.3 5.5-4.8 7.2l7.7 6c4.5-4.2 6.9-10.3 6.9-17.7z" />
              <path fill="#FBBC05" d="M10.6 28.6c-.5-1.4-.8-3-.8-4.6s.3-3.2.8-4.6l-7.9-6.1C1 16.6 0 20.2 0 24s1 7.4 2.7 10.7l7.9-6.1z" />
              <path fill="#34A853" d="M24 48c6.5 0 11.9-2.1 15.9-5.8l-7.7-6c-2.2 1.5-5 2.3-8.2 2.3-6.2 0-11.5-4.1-13.4-9.9l-7.9 6.1C6.6 42.6 14.6 48 24 48z" />
            </svg>
            Continue with Google
          </a>
          {devHost && (
            <a href={`${API_BASE}/auth/dev/start?next=${encodeURIComponent(next)}`} className="btn btn-ghost mt-3 w-full">
              <Icon name="code" size={18} /> Dev sign-in (local only)
            </a>
          )}
          <p className="mt-6 border-t border-line pt-5 text-sm text-ink-3">
            The server may take up to a minute to wake after a quiet spell. If Google's page takes a while, that is why.
          </p>
        </div>
      </div>
    </main>
  );
}
