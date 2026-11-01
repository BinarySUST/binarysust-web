// A square portrait. With no photo, initials on a quiet tint picked from the handle: same handle, same tint, on
// every build and every device.
// One hue (the brand green); the handle picks a step of lightness and chroma, so tiles vary without new colours.
const TINTS = [
  { l: 0.95, c: 0.03, dl: 0.27, dc: 0.04 },
  { l: 0.93, c: 0.04, dl: 0.25, dc: 0.035 },
  { l: 0.96, c: 0.02, dl: 0.3, dc: 0.03 },
  { l: 0.92, c: 0.025, dl: 0.24, dc: 0.025 },
  { l: 0.94, c: 0.015, dl: 0.28, dc: 0.02 },
] as const;

function hash(s: string) {
  let h = 2166136261;
  for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
  return h >>> 0;
}

export function initials(name: string) {
  const words = name
    .replace(/^(Dr|Prof|Mr|Ms|Mrs)\.?\s+/i, "")
    .split(/\s+/)
    .filter(Boolean);
  return ((words[0]?.[0] ?? "") + (words.length > 1 ? (words[words.length - 1]?.[0] ?? "") : "")).toUpperCase();
}

export function Avatar({
  handle,
  name,
  url,
  size = 64,
  className = "",
}: {
  handle: string;
  name: string;
  url: string | null;
  size?: number;
  className?: string;
}) {
  const t = TINTS[hash(handle) % TINTS.length] ?? TINTS[0];
  const style = { width: size, height: size };
  if (url) {
    return (
      <img
        src={url}
        alt={name}
        width={size}
        height={size}
        loading="lazy"
        decoding="async"
        className={`shrink-0 rounded-[4px] border border-line object-cover ${className}`}
        style={style}
      />
    );
  }
  return (
    <span
      role="img"
      aria-label={name}
      className={`inline-grid shrink-0 place-items-center rounded-[4px] border border-line select-none ${className}`}
      style={{
        ...style,
        background: `light-dark(oklch(${t.l} ${t.c} 149), oklch(${t.dl} ${t.dc} 152))`,
        color: "light-dark(oklch(0.36 0.07 149), oklch(0.88 0.05 149))",
      }}
    >
      <span className="display" style={{ fontSize: Math.round(size * 0.36), letterSpacing: "-0.01em" }} aria-hidden="true">
        {initials(name)}
      </span>
    </span>
  );
}
