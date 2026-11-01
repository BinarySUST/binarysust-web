// Card, shaped like shadcn/ui's (Card, CardHeader, CardTitle, CardContent, CardFooter): a hairline-framed panel
// whose parts are separated by hairlines.
import type { HTMLAttributes } from "react";

type P = HTMLAttributes<HTMLDivElement>;
const cx = (...c: (string | undefined)[]) => c.filter(Boolean).join(" ");

export const Card = ({ className, ...p }: P) => <div className={cx("overflow-hidden rounded-md border border-input bg-card text-card-foreground", className)} {...p} />;
export const CardHeader = ({ className, ...p }: P) => <div className={cx("flex items-center gap-3 border-b border-border px-4 py-2.5", className)} {...p} />;
export const CardTitle = ({ className, ...p }: HTMLAttributes<HTMLHeadingElement>) => <h2 className={cx("text-[0.9375rem] font-semibold", className)} {...p} />;
export const CardContent = ({ className, ...p }: P) => <div className={cx("px-5 py-5", className)} {...p} />;
export const CardFooter = ({ className, ...p }: P) => <div className={cx("border-t border-border px-5 py-4", className)} {...p} />;
