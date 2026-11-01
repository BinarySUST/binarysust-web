import { createFileRoute } from "@tanstack/react-router";
import { BRAND } from "@binarysust/ui/lib/brand";
import { content } from "~/content";
import { seo } from "~/lib/seo";
import { LandingPage } from "~/components/landing/LandingPage";

export const Route = createFileRoute("/")({
  loader: () => content.landing(),
  head: () =>
    seo({
      title: "Binary SUST",
      description: `${BRAND.mission} The student technology club of ${BRAND.university}.`,
      path: "/",
    }),
  component: LandingPage,
});
