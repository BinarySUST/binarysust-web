import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

// Re-skinned for the Binary SUST drawing sheet (apps/web/DESIGN.md): 44px controls, 4px radius, hairline
// outline variant, no shadows. Focus uses the global 2px :focus-visible outline from globals.css.
const buttonVariantsBase = cva(
  "group/button inline-flex shrink-0 cursor-pointer items-center justify-center gap-[0.5em] rounded-md border border-transparent font-sans text-[0.9375rem] leading-[1.2] font-[560] whitespace-nowrap no-underline transition-colors duration-150 ease-out-expo select-none active:not-disabled:not-aria-[haspopup]:translate-y-px disabled:cursor-not-allowed disabled:opacity-50 aria-disabled:cursor-not-allowed aria-disabled:opacity-50 aria-invalid:border-destructive [&_svg]:pointer-events-none [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground hover:not-disabled:bg-primary-hover",
        outline: "border-line-strong text-ink hover:not-disabled:border-ink-2 hover:not-disabled:bg-ground-2 aria-expanded:bg-ground-2",
        secondary: "bg-secondary text-secondary-foreground hover:not-disabled:bg-line aria-expanded:bg-line",
        ghost: "text-ink hover:not-disabled:bg-accent aria-expanded:bg-accent",
        destructive: "border-destructive text-destructive hover:not-disabled:bg-destructive/10",
        link: "rounded-none border-0 text-[length:inherit] leading-[inherit] [font-weight:inherit] text-inherit underline decoration-[color-mix(in_oklab,var(--primary)_55%,transparent)] decoration-1 underline-offset-[0.22em] hover:text-primary-ink hover:decoration-current",
      },
      size: {
        default: "min-h-11 px-[1.125rem] py-2",
        sm: "min-h-9 px-3 py-1 text-sm",
        lg: "min-h-12 px-6 py-2.5",
        icon: "size-11 p-0",
        "icon-sm": "size-9 p-0",
      },
    },
    compoundVariants: [{ variant: "link", className: "min-h-0 px-0 py-0" }],
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

/** The variant classes, conflict-merged so `className` overrides win when used on links (`<a className={...}>`). */
const buttonVariants = (props?: Parameters<typeof buttonVariantsBase>[0]) => cn(buttonVariantsBase(props))

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariantsBase>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
