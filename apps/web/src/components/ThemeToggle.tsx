import { useSyncExternalStore } from "react";
import { Button } from "@binarysust/ui/components/button";
import { Icon } from "~/components/Icon";

type Theme = "dark" | "light";

/**
 * Inline in <head> so the theme applies before first paint: the visitor's stored choice, else the system's.
 * Until a choice is stored, the page keeps following the system if it changes.
 */
export const THEME_SCRIPT = `(function(){var d=document.documentElement,m=window.matchMedia("(prefers-color-scheme: dark)");function s(){var t=null;try{t=localStorage.getItem("theme")}catch(e){}d.dataset.theme=t==="light"||t==="dark"?t:m.matches?"dark":"light"}s();try{m.addEventListener("change",s)}catch(e){}})()`;

function subscribe(cb: () => void) {
  const mo = new MutationObserver(cb);
  mo.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  return () => mo.disconnect();
}
const read = (): Theme => (document.documentElement.dataset.theme === "dark" ? "dark" : "light");

export function ThemeToggle() {
  const theme = useSyncExternalStore(subscribe, read, (): Theme => "light");
  const next: Theme = theme === "dark" ? "light" : "dark";
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className="text-ink-2 hover:text-ink"
      aria-label={`Switch to ${next} theme`}
      title={`Switch to ${next} theme`}
      onClick={() => {
        document.documentElement.dataset.theme = next;
        try {
          localStorage.setItem("theme", next);
        } catch {
          /* private mode: the choice lasts for this page only */
        }
      }}
    >
      <Icon name={theme === "dark" ? "sun" : "moon"} />
    </Button>
  );
}
