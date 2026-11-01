import { BRAND } from "@binarysust/ui/lib/brand";
import { cn } from "@binarysust/ui/lib/utils";

/**
 * The official logo (BRAND.logoSrc). Without one, the name is set in type; `motto` adds "technology for real life"
 * underneath. Pass a height class (e.g. `h-9`) to size it; the default suits the 64px site header.
 */
export function Logo({ motto = false, className = "" }: { motto?: boolean; className?: string }) {
  if (BRAND.logoSrc) {
    return <img src={BRAND.logoSrc} alt={BRAND.name} width={424} height={222} className={cn("h-12 w-auto", className)} />;
  }
  return (
    <span className={cn("inline-flex items-center gap-2.5", className)}>
      <span className="flex flex-col justify-center">
        <span className="display block text-[1.0625rem] leading-none tracking-[-0.01em]">
          Binary <span className="text-ink-2">SUST</span>
        </span>
        {motto && <span className="meta mt-1.5 text-xs text-ink-3">{BRAND.motto}</span>}
      </span>
    </span>
  );
}
