import { useEffect, useRef } from "react";

/**
 * The hero's ambient layer: a faint engineering grid, and a soft light that follows a mouse pointer across it.
 * Pure background. No pointer, or reduced motion, and the light rests in one place.
 */
export function HeroBackdrop() {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    const host = el?.parentElement;
    if (!el || !host) return;
    if (!window.matchMedia("(pointer: fine)").matches || window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
    let frame = 0;
    const move = (e: PointerEvent) => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        const r = host.getBoundingClientRect();
        el.style.setProperty("--gx", `${e.clientX - r.left}px`);
        el.style.setProperty("--gy", `${e.clientY - r.top}px`);
      });
    };
    host.addEventListener("pointermove", move);
    return () => {
      host.removeEventListener("pointermove", move);
      cancelAnimationFrame(frame);
    };
  }, []);
  return (
    <div ref={ref} className="pointer-events-none absolute inset-0" aria-hidden="true">
      <div className="blueprint absolute inset-0" />
      <div className="glow absolute inset-0" />
    </div>
  );
}
