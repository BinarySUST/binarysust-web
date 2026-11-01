import { createFileRoute } from "@tanstack/react-router";
import { SectionOutlet } from "~/dashboard/SectionOutlet";

// Every dashboard URL below /dashboard resolves through src/dashboard/sections.tsx.
export const Route = createFileRoute("/dashboard/$")({
  component: function DashboardSection() {
    return <SectionOutlet splat={Route.useParams()._splat ?? ""} />;
  },
});
