import { Link } from "@tanstack/react-router";

export function NotFound() {
  return (
    <section className="mx-auto max-w-7xl px-4 pt-20 pb-8 sm:px-6 sm:pt-28">
      <h1 className="display max-w-3xl text-[clamp(2.25rem,6vw,4rem)]">This page does not exist.</h1>
      <p className="mt-5 max-w-xl text-lg text-ink-2">
        The link may be old, or the page was renamed. Everything we publish is reachable from the home page.
      </p>
      <p className="meta mt-3 text-ink-3">HTTP 404</p>
      <div className="mt-9 flex flex-wrap gap-3">
        <Link to="/" className="btn btn-primary">
          Go to the home page
        </Link>
        <Link to="/events" className="btn btn-ghost">
          See events
        </Link>
      </div>
    </section>
  );
}
