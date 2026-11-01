import { createFileRoute } from "@tanstack/react-router";
import { DashboardRoot } from "~/dashboard/Layout";

// Client-only SPA: never pre-rendered, served by _shell.html (see public/_redirects).
export const Route = createFileRoute("/dashboard")({
  ssr: false,
  head: () => ({ meta: [{ title: "Dashboard | Binary SUST" }, { name: "robots", content: "noindex" }] }),
  component: DashboardRoot,
});
