import * as React from "react";
import { cn } from "cn";
import { ChevronDownIcon } from "lucide-react";

type NativeSelectProps = Omit<React.ComponentProps<"select">, "size"> & {
  size?: "sm" | "default";
};

function NativeSelect({ className, size = "default", ...props }: NativeSelectProps) {
  return (
    <div
      className={cn("group/native-select relative w-full has-[select:disabled]:opacity-50", className)}
      data-slot="native-select-wrapper"
      data-size={size}
    >
      <select
        data-slot="native-select"
        data-size={size}
        className="min-h-11 w-full min-w-0 rounded-md border border-line-strong bg-panel pl-3 pr-9 py-2 text-ink transition-colors placeholder:text-ink-3 hover:border-ink-3 focus-visible:border-ring focus-visible:outline-offset-0 disabled:cursor-not-allowed disabled:opacity-50 aria-invalid:border-destructive appearance-none data-[size=sm]:min-h-9 data-[size=sm]:py-1 data-[size=sm]:text-sm"
        {...props}
      />
      <ChevronDownIcon
        className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-ink-3 select-none"
        aria-hidden="true"
        data-slot="native-select-icon"
      />
    </div>
  );
}

function NativeSelectOption({ className, ...props }: React.ComponentProps<"option">) {
  return <option data-slot="native-select-option" className={cn("bg-[Canvas] text-[CanvasText]", className)} {...props} />;
}

function NativeSelectOptGroup({ className, ...props }: React.ComponentProps<"optgroup">) {
  return <optgroup data-slot="native-select-optgroup" className={cn("bg-[Canvas] text-[CanvasText]", className)} {...props} />;
}

export { NativeSelect, NativeSelectOptGroup, NativeSelectOption };
