import { BRAND } from "@binarysust/ui/lib/brand";
import { Mark } from "@binarysust/ui/components/mark";

/**
 * The club's name as a lockup. When BRAND.logoSrc points at the official SVG it is used instead; until then the
 * name is set in type next to the bit mark. `motto` adds "technology for real life" underneath.
 */
export function Logo({ motto = false, className = "" }: { motto?: boolean; className?: string }) {
  if (BRAND.logoSrc) {
    return <img src={BRAND.logoSrc} alt={BRAND.name} width={424} height={222} className={`h-12 w-auto ${className}`} />;
  }
  return (
    <span className={`inline-flex items-center gap-2.5 ${className}`}>
      <Mark className="h-[18px] w-[79px] shrink-0 text-ink-3" />
      <span className="flex flex-col justify-center">
        <span className="display block text-[1.0625rem] leading-none tracking-[-0.01em]">
          Binary <span className="text-ink-2">SUST</span>
        </span>
        {motto && <span className="meta mt-1.5 text-xs text-ink-3">{BRAND.motto}</span>}
      </span>
    </span>
  );
}
