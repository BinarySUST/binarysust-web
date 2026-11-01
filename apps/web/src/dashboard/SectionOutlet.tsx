import { Link } from "@tanstack/react-router";
import { useMe } from "~/dashboard/me";
import { allowed, findSection } from "~/dashboard/sections";
import { SectionHead } from "~/dashboard/ui";

/** Renders the section a /dashboard/<path> URL names, if the caller may see it. */
export function SectionOutlet({ splat }: { splat: string }) {
  const me = useMe();
  const { section, rest } = findSection(splat);
  if (!section)
    return (
      <>
        <SectionHead title="Not found" />
        <p className="text-ink-2">
          There is no dashboard section here. <Link to="/dashboard" className="link">Back to the overview</Link>.
        </p>
      </>
    );
  if (!allowed(me, section))
    return (
      <>
        <SectionHead title={section.title} />
        <p className="text-ink-2">
          Your roles do not include this section. If you think they should, ask whoever holds Committee &amp; roles.
        </p>
      </>
    );
  const C = section.component;
  return <C rest={rest} />;
}
