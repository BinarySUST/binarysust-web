// Button, shaped like shadcn/ui's (variant + size, `buttonVariants` for links styled as buttons) so it can be
// swapped for the shadcn component one-for-one. The look lives in .btn / .btn-* in styles.css.
import type { ButtonHTMLAttributes } from "react";

export type ButtonVariant = "default" | "outline" | "ghost" | "link";
export type ButtonSize = "default" | "sm";

const VARIANT: Record<ButtonVariant, string> = {
  default: "btn btn-primary",
  outline: "btn btn-ghost",
  ghost: "btn border-transparent text-ink hover:bg-accent",
  link: "more",
};

export function buttonVariants({ variant = "default", size = "default", className = "" }: { variant?: ButtonVariant; size?: ButtonSize; className?: string } = {}) {
  return [VARIANT[variant], size === "sm" && variant !== "link" ? "btn-sm" : "", className].filter(Boolean).join(" ");
}

export function Button({
  variant,
  size,
  className,
  type = "button",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant; size?: ButtonSize }) {
  return <button type={type} className={buttonVariants({ variant, size, className })} {...props} />;
}
