// Badge, shaped like shadcn/ui's: a hairline mono chip for kinds, statuses and tags (.tag in styles.css).
import type { HTMLAttributes } from "react";

export type BadgeVariant = "outline" | "default" | "destructive";

const VARIANT: Record<BadgeVariant, string> = {
  outline: "tag",
  default: "tag !border-primary !text-primary-ink",
  destructive: "tag !border-destructive !text-destructive",
};

export function badgeVariants({ variant = "outline", className = "" }: { variant?: BadgeVariant; className?: string } = {}) {
  return [VARIANT[variant], className].filter(Boolean).join(" ");
}

export function Badge({ variant, className, ...props }: HTMLAttributes<HTMLSpanElement> & { variant?: BadgeVariant }) {
  return <span className={badgeVariants({ variant, className })} {...props} />;
}
