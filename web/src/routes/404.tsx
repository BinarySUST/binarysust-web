import { createFileRoute } from "@tanstack/react-router";
import { NotFound } from "~/components/NotFound";

// Pre-rendered to /404.html, which Cloudflare Pages serves for any unknown URL.
export const Route = createFileRoute("/404")({
  head: () => ({ meta: [{ title: "Page not found | Binary SUST" }, { name: "robots", content: "noindex" }] }),
  component: NotFound,
});
