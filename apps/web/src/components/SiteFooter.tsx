import { Link } from "@tanstack/react-router";
import { BRAND } from "@binarysust/ui/lib/brand";
import { Icon } from "~/components/Icon";
import { Logo } from "@binarysust/ui/components/logo";
import { NAV } from "~/components/SiteHeader";

const MORE = [
  { to: "/alumni", label: "Alumni" },
  { to: "/advisors", label: "Advisors" },
] as const;

export function SiteFooter() {
  return (
    <footer id="site-footer" className="mt-28 border-t border-line-strong bg-ground-2">
      <div className="mx-auto grid max-w-7xl gap-x-10 gap-y-10 px-4 pt-12 pb-10 sm:px-6 md:grid-cols-[1.5fr_1fr_1fr]">
        <div>
          <Logo motto />
          <p className="mt-5 max-w-sm text-[0.9375rem] text-ink-2">The student technology club of {BRAND.university}.</p>
          <p lang="bn" className="mt-4 font-bangla text-lg font-medium text-ink-2">
            {BRAND.bangla}
          </p>
        </div>
        <nav aria-label="Footer">
          <h2 className="meta mb-3 text-ink-3">Site</h2>
          <ul className="grid grid-cols-2 gap-x-6 gap-y-2 text-[0.9375rem]">
            {[...NAV, ...MORE].map((n) => (
              <li key={n.to}>
                <Link to={n.to} className="text-ink-2 no-underline hover:text-ink hover:underline">
                  {n.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>
        <div>
          <h2 className="meta mb-3 text-ink-3">Contact</h2>
          <ul className="space-y-2 text-[0.9375rem]">
            <li>
              <a
                href={`mailto:${BRAND.email}`}
                className="inline-flex items-center gap-2 break-all text-ink-2 no-underline hover:text-ink hover:underline"
              >
                <Icon name="mail" size={16} /> {BRAND.email}
              </a>
            </li>
            {BRAND.links.map((l) => (
              <li key={l.href}>
                <a
                  href={l.href}
                  rel="me noopener"
                  className="inline-flex items-center gap-2 text-ink-2 no-underline hover:text-ink hover:underline"
                >
                  <Icon name="external" size={16} /> {l.label}
                </a>
              </li>
            ))}
            <li>
              <a href="/rss.xml" className="inline-flex items-center gap-2 text-ink-2 no-underline hover:text-ink hover:underline">
                <Icon name="rss" size={16} /> Blog RSS feed
              </a>
            </li>
            <li>
              <a href="/login" className="inline-flex items-center gap-2 text-ink-2 no-underline hover:text-ink hover:underline">
                <Icon name="user" size={16} /> Member sign-in
              </a>
            </li>
          </ul>
        </div>
      </div>
      <div className="border-t border-line">
        <p className="meta mx-auto flex max-w-7xl flex-wrap justify-between gap-x-6 gap-y-1 px-4 py-4 text-ink-3 sm:px-6">
          <span>Binary SUST · SUST, Sylhet 3114</span>
          <span>
            <a href="/sitemap.xml" className="text-ink-3 no-underline hover:text-ink hover:underline">
              sitemap.xml
            </a>
          </span>
        </p>
      </div>
    </footer>
  );
}
