import { createFileRoute } from "@tanstack/react-router";
import { SectionOutlet } from "~/dashboard/SectionOutlet";

export const Route = createFileRoute("/dashboard/")({
  component: () => <SectionOutlet splat="" />,
});
