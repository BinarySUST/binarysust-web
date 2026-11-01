import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

// The drawing sheet's tag: a hairline mono chip for kinds, statuses and post tags (apps/web/DESIGN.md).
// As a link, aria-current="page" inverts it to ink.
const badgeVariantsBase = cva(
  "group/badge inline-flex min-h-7 w-fit shrink-0 items-center gap-[0.375em] rounded-sm border px-2 py-0.5 font-mono text-xs leading-[1.3] whitespace-nowrap no-underline [&>svg]:pointer-events-none [&>svg]:size-3!",
  {
    variants: {
      variant: {
        outline:
          "border-line-strong text-ink-2 [a]:hover:border-ink-2 [a]:hover:text-ink aria-[current=page]:border-ink aria-[current=page]:bg-ink aria-[current=page]:text-ground",
        default: "border-primary text-primary-ink",
        secondary: "border-line bg-secondary text-secondary-foreground",
        destructive: "border-destructive text-destructive",
      },
    },
    defaultVariants: {
      variant: "outline",
    },
  }
)

/** The variant classes, conflict-merged so `className` overrides win when used on links (`<a className={...}>`). */
const badgeVariants = (props?: Parameters<typeof badgeVariantsBase>[0]) => cn(badgeVariantsBase(props))

function Badge({
  className,
  variant = "outline",
  render,
  ...props
}: useRender.ComponentProps<"span"> & VariantProps<typeof badgeVariantsBase>) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ variant }), className),
      },
      props
    ),
    render,
    state: {
      slot: "badge",
      variant,
    },
  })
}

export { Badge, badgeVariants }
