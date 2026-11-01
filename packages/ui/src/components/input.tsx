import * as React from "react"
import { Input as InputPrimitive } from "@base-ui/react/input"
import { cn } from "cn"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        // The drawing sheet's field: strong hairline, panel fill, 44px (apps/web/DESIGN.md).
        "min-h-11 w-full min-w-0 rounded-md border border-line-strong bg-panel px-3 py-2 text-ink transition-colors placeholder:text-ink-3 hover:border-ink-3 focus-visible:border-ring focus-visible:outline-offset-0 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground",
        className
      )}
      {...props}
    />
  )
}

export { Input }
