import { createFileRoute } from "@tanstack/react-router";
import { BRAND } from "@binarysust/ui/lib/brand";
import { seo } from "~/lib/seo";

// Linked from the Google sign-in consent screen; Google requires it to publish the OAuth app.
export const Route = createFileRoute("/privacy")({
  head: () =>
    seo({
      title: "Privacy",
      description: "What Binary SUST stores when a member signs in with Google, and how to have it removed.",
      path: "/privacy",
    }),
  component: PrivacyPage,
});

function PrivacyPage() {
  return (
    <article className="mx-auto max-w-3xl px-4 pt-20 pb-8 sm:px-6 sm:pt-28">
      <h1 className="display text-[clamp(2.25rem,6vw,4rem)]">Privacy</h1>
      <div className="mt-8 space-y-5 text-lg text-ink-2">
        <p>Visitors can read this site without an account. We do not use analytics, advertising or tracking cookies.</p>
        <h2 className="display pt-4 text-2xl text-ink">Member sign-in</h2>
        <p>
          Club members sign in with Google. We ask Google only for your email address, name and profile picture, and use them to check that
          the committee added your email and to show who you are in the dashboard. We do not read anything else from your Google account and
          never post on your behalf.
        </p>
        <p>
          We store your email, name, profile picture link and the time you last signed in. A cookie keeps you signed in; it is removed when
          you sign out or it expires.
        </p>
        <h2 className="display pt-4 text-2xl text-ink">Public profiles</h2>
        <p>Members, committee and alumni profiles show only what the club publishes on this site. Your email is never shown publicly.</p>
        <h2 className="display pt-4 text-2xl text-ink">Sharing and removal</h2>
        <p>
          We do not sell or share your data. It is kept on the club&apos;s own database and servers. To see, correct or delete what we hold
          about you, email{" "}
          <a href={`mailto:${BRAND.email}`} className="text-ink">
            {BRAND.email}
          </a>
          .
        </p>
      </div>
    </article>
  );
}
