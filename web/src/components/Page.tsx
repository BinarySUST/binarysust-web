import type { ReactNode } from "react";

/** Standard public page header: title, optional lede, then whatever the page adds (filters, meta). */
export function PageHead({ title, lede, children }: { title: ReactNode; lede?: ReactNode; children?: ReactNode }) {
  return (
    <header className="mx-auto max-w-7xl px-4 pt-14 pb-10 sm:px-6 sm:pt-20">
      <h1 className="display text-[clamp(2.25rem,5.5vw,3.75rem)]">{title}</h1>
      {lede && <p className="mt-4 max-w-2xl text-lg text-ink-2 sm:text-xl">{lede}</p>}
      {children}
    </header>
  );
}

/**
 * A page section. Every heading hangs on the same left axis, with a hairline running out to the optional action.
 */
export function Section({
  title,
  index,
  action,
  children,
  id,
  className = "",
}: {
  title: string;
  index?: string;
  action?: ReactNode;
  children: ReactNode;
  id?: string;
  className?: string;
}) {
  return (
    <section className={`mx-auto max-w-7xl px-4 pt-24 sm:px-6 ${className}`} aria-labelledby={id}>
      <SectionHead id={id} title={title} index={index} action={action} />
      {children}
    </section>
  );
}

/** `index` is the section's place on the page ("01"), set in mono on the heading's own line. */
export function SectionHead({
  id,
  title,
  index,
  action,
  as: H = "h2",
}: {
  id?: string;
  title: string;
  index?: string;
  action?: ReactNode;
  as?: "h2" | "h3";
}) {
  return (
    <div className="mb-8 flex flex-wrap items-baseline gap-x-6 gap-y-3 border-b border-line-strong pb-4">
      <H id={id} className="display flex items-baseline gap-4 text-[clamp(1.5rem,3vw,2rem)]">
        {index && (
          <span className="meta shrink-0 text-[0.8125rem] font-normal tracking-normal text-primary-ink" aria-hidden="true">
            {index}
          </span>
        )}
        <span>{title}</span>
      </H>
      {action && <div className="ml-auto">{action}</div>}
    </div>
  );
}
