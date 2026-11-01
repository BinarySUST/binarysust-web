// The club mark: eight bits reading 01000010, the letter B in ASCII. A filled cell is a 1.
const B = "01000010";

export function Mark({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 88 20" className={className} aria-hidden="true" focusable="false">
      {B.split("").map((bit, i) => (
        <rect
          key={i}
          x={2.5 + i * 11}
          y={4.5}
          width={9}
          height={11}
          rx={1}
          fill={bit === "1" ? "var(--brand)" : "none"}
          stroke={bit === "1" ? "var(--brand)" : "currentColor"}
          strokeWidth={1}
          opacity={bit === "1" ? 1 : 0.55}
        />
      ))}
    </svg>
  );
}
