// THE list of dashboard sections. The sidebar, the routes (/dashboard/<path>) and permission filtering
// all derive from it. Adding a section = one entry here. A typo in `requires` fails `tsc`, because
// Permission comes from the API's generated types.
import type { ComponentType } from "react";
import type { IconName } from "~/components/Icon";
import type { Me, Permission } from "@binarysust/api-client";
import { BlogSection } from "~/features/blog/dashboard/BlogSection";
import { CommitteeSection } from "~/features/committee/dashboard/CommitteeSection";
import { EventsSection } from "~/features/events/dashboard/EventsSection";
import { MediaSection } from "~/features/media/dashboard/MediaSection";
import { Overview } from "~/dashboard/Overview";
import { PeopleSection } from "~/features/people/dashboard/PeopleSection";
import { PortfolioSection } from "~/features/portfolio/dashboard/PortfolioSection";
import { RsvpsSection } from "~/features/events/dashboard/RsvpsSection";
import { SponsorsSection } from "~/features/sponsors/dashboard/SponsorsSection";
import { TalksSection } from "~/features/talks/dashboard/TalksSection";

/** `rest` is the path below the section, e.g. ["42"] for /dashboard/events/42. */
export type SectionProps = { rest: string[] };

export type Section = {
  path: string; // "" is /dashboard itself
  title: string;
  icon: IconName;
  requires: Permission | Permission[] | "signed_in"; // an array means any one of them
  component: ComponentType<SectionProps>;
};

export const sections: Section[] = [
  { path: "", title: "Overview", icon: "home", requires: "signed_in", component: Overview },
  { path: "portfolio", title: "My portfolio", icon: "user", requires: "profile.edit_own", component: PortfolioSection },
  { path: "rsvps", title: "My RSVPs", icon: "ticket", requires: "signed_in", component: RsvpsSection },
  { path: "events", title: "Events", icon: "calendar", requires: "events.manage", component: EventsSection },
  { path: "blog", title: "Blog", icon: "pen", requires: ["blog.write", "blog.review"], component: BlogSection },
  { path: "people", title: "People", icon: "users", requires: "people.manage", component: PeopleSection },
  { path: "committee", title: "Committee & roles", icon: "shield", requires: "roles.manage", component: CommitteeSection },
  { path: "talks", title: "Talks", icon: "mic", requires: "talks.manage", component: TalksSection },
  { path: "sponsors", title: "Sponsors", icon: "handshake", requires: "sponsors.manage", component: SponsorsSection },
  { path: "media", title: "Media", icon: "image", requires: "media.upload", component: MediaSection },
];

export function allowed(me: Me, s: Section): boolean {
  if (s.requires === "signed_in") return true;
  const need = Array.isArray(s.requires) ? s.requires : [s.requires];
  return need.some((p) => me.permissions.includes(p));
}

export const visibleSections = (me: Me) => sections.filter((s) => allowed(me, s));

export function findSection(splat: string): { section: Section | undefined; rest: string[] } {
  const [head = "", ...rest] = splat.split("/").filter(Boolean);
  return { section: sections.find((s) => s.path === head), rest };
}
