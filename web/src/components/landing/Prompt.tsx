import { useEffect, useState } from "react";

/**
 * A shell prompt that types its command once. The full command is in the pre-rendered HTML and stays readable
 * without JS; the typing only runs on a visible page without reduced motion, and a timer (not a paused animation
 * timeline) drives it, so the text can never get stuck half-typed.
 */
export function Prompt({ command }: { command: string }) {
  const [shown, setShown] = useState(command.length);
  useEffect(() => {
    if (document.visibilityState !== "visible" || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    let n = 0;
    setShown(0);
    let timer = window.setTimeout(function tick() {
      n += 1;
      setShown(n);
      if (n < command.length) timer = window.setTimeout(tick, 45);
    }, 350);
    return () => {
      window.clearTimeout(timer);
      setShown(command.length);
    };
  }, [command]);
  return (
    <p className="meta text-[0.875rem] text-ink-2">
      <span className="text-primary-ink" aria-hidden="true">
        ${" "}
      </span>
      <span className="sr-only">{command}</span>
      <span aria-hidden="true">{command.slice(0, shown)}</span>
      <span className="caret" aria-hidden="true" />
    </p>
  );
}
