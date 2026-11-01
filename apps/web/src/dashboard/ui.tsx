import { type QueryClient, type QueryKey, useMutation } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Children, type ReactElement, type ReactNode, cloneElement, isValidElement, useId, useState } from "react";
import { Icon } from "~/components/Icon";
import { ApiError, useWaking } from "@binarysust/api-client";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@binarysust/ui/components/alert-dialog";
import { Button } from "@binarysust/ui/components/button";
import { FieldDescription, FieldError, FieldLabel, Field as UiField } from "@binarysust/ui/components/field";
import { Input } from "@binarysust/ui/components/input";
import { NativeSelect } from "@binarysust/ui/components/native-select";
import { Table as UiTable, TableBody, TableHead, TableHeader, TableRow } from "@binarysust/ui/components/table";
import { Textarea } from "@binarysust/ui/components/textarea";

export function SectionHead({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
      <h1 className="sign text-3xl">{title}</h1>
      {children}
    </div>
  );
}

export function WakingBanner() {
  const waking = useWaking();
  if (!waking) return null;
  return (
    <div role="status" className="mb-6 flex items-center gap-3 rounded-lg border border-yellow/50 bg-yellow/10 px-4 py-3 text-ink">
      <span className="size-2.5 animate-pulse rounded-full bg-yellow" aria-hidden="true" />
      Waking the server (up to a minute)…
    </div>
  );
}

export function Loading({ what = "Loading" }: { what?: string }) {
  return <p className="py-10 text-ink-2" role="status">{what}…</p>;
}

export function ErrorBox({ error, retry }: { error: unknown; retry?: () => void }) {
  const msg = error instanceof Error ? error.message : "Something went wrong.";
  return (
    <div role="alert" className="rounded-lg border border-vermilion/60 bg-vermilion/10 p-4">
      <p className="flex items-center gap-2 font-semibold">
        <Icon name="alert" size={18} /> {msg}
      </p>
      {error instanceof ApiError && error.requestId && <p className="mt-1 text-sm text-ink-3">Request {error.requestId}</p>}
      {retry && (
        <Button type="button" onClick={retry} variant="outline" size="sm" className="mt-3">
          Try again
        </Button>
      )}
    </div>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <p className="rounded-lg border border-dashed border-line p-8 text-center text-ink-2">{children}</p>;
}

export function Table({ head, children }: { head: string[]; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-lg border border-line">
      <UiTable className="min-w-[36rem] text-left text-[0.95rem]">
        <TableHeader className="bg-ground-2 text-sm">
          <TableRow className="hover:bg-transparent">
            {head.map((h, i) => (
              <TableHead key={h || i} scope="col" className="h-auto px-4 py-3 font-semibold text-ink-2">
                {h}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody className="divide-y divide-line">{children}</TableBody>
      </UiTable>
    </div>
  );
}

export const td = "px-4 py-3";

/** The controls Field wires to its label: @binarysust/ui inputs, or native ones for custom widgets. */
const CONTROLS = new Set<unknown>(["input", "select", "textarea", Input, NativeSelect, Textarea]);
type ControlProps = { id?: string; "aria-describedby"?: string; "aria-invalid"?: boolean | "true" | "false" };

/**
 * A labelled input. `error` is the API's message for this field (422 `fields`), shown under it.
 *
 * The label is a real `<label htmlFor>`: Field gives its first control (Input, NativeSelect or Textarea
 * among its direct children) an id, and points `aria-describedby` at the hint or error. Anything else
 * (a custom widget) is wrapped in a group named by the label.
 */
export function Field({ label, hint, error, children }: { label: string; hint?: string; error?: string; children: ReactNode }) {
  const base = useId();
  const labelId = `${base}-label`;
  const noteId = `${base}-note`;
  const note = error ?? hint;
  const items = Children.toArray(children);
  const at = items.findIndex((c) => isValidElement(c) && CONTROLS.has(c.type));
  const control = at >= 0 ? (items[at] as ReactElement<ControlProps>) : undefined;
  const controlId = control ? (control.props.id ?? `${base}-control`) : undefined;
  const describedBy = [control?.props["aria-describedby"], note ? noteId : undefined].filter(Boolean).join(" ") || undefined;
  const wired = control
    ? items.map((c, i) =>
        i === at ? cloneElement(control, { id: controlId, "aria-describedby": describedBy, "aria-invalid": error ? true : control.props["aria-invalid"] }) : c,
      )
    : items;
  const labelClass = "text-sm font-semibold text-ink-2";
  return (
    <UiField
      data-invalid={error ? true : undefined}
      aria-labelledby={controlId ? undefined : labelId}
      aria-describedby={!controlId && note ? noteId : undefined}
      className="gap-1.5"
    >
      {controlId ? (
        <FieldLabel htmlFor={controlId} id={labelId} className={labelClass}>
          {label}
        </FieldLabel>
      ) : (
        <span id={labelId} className={labelClass}>
          {label}
        </span>
      )}
      {wired}
      {error ? (
        <FieldError id={noteId} className="flex items-center gap-1.5 font-semibold">
          <Icon name="alert" size={14} /> {error}
        </FieldError>
      ) : (
        hint && (
          <FieldDescription id={noteId} className="text-ink-3">
            {hint}
          </FieldDescription>
        )
      )}
    </UiField>
  );
}

/** Field problems from a failed save, keyed by the API's input name ("registration.capacity"). */
export function fieldErrors(error: unknown): Record<string, string> {
  return error instanceof ApiError ? error.fields : {};
}

/** Message for one field; `prefix` matches nested names too ("socials" matches "socials[0].url"). */
export function errorFor(fields: Record<string, string>, name: string, prefix = false): string | undefined {
  if (fields[name]) return fields[name];
  if (!prefix) return undefined;
  const k = Object.keys(fields).find((f) => f.startsWith(name + ".") || f.startsWith(name + "["));
  return k ? `${k.slice(name.length).replace(/^\./, "")}: ${fields[k]}` : undefined;
}

/**
 * The save row under a form: button, then whatever happened. Field errors the form shows inline
 * (`shown`) are not repeated; any others are listed so a 422 never fails silently.
 */
export function SaveBar({
  pending,
  error,
  success,
  label,
  shown = [],
  successText = "Saved. The public site republishes in a few minutes.",
  problem,
  children,
  sticky,
}: {
  pending: boolean;
  error: unknown;
  success: boolean;
  label: string;
  shown?: string[];
  successText?: string;
  problem?: string | null;
  children?: ReactNode;
  sticky?: boolean;
}) {
  const fields = fieldErrors(error);
  const isShown = (k: string) => shown.some((s) => k === s || k.startsWith(s + ".") || k.startsWith(s + "["));
  const others = Object.entries(fields).filter(([k]) => !isShown(k));
  const hasFields = Object.keys(fields).length > 0;
  return (
    <div className={`flex flex-wrap items-center gap-4 border-t border-line pt-6 ${sticky ? "sticky bottom-0 -mx-4 bg-ground px-4 pb-4" : ""}`}>
      <Button type="submit" disabled={pending}>
        {pending ? "Saving…" : label}
      </Button>
      {children}
      <span aria-live="polite" className="text-sm">
        {problem ? (
          <span className="font-semibold text-vermilion">{problem}</span>
        ) : error ? (
          <span className="font-semibold text-vermilion">
            {hasFields ? "Some fields need fixing." : error instanceof Error ? error.message : "Could not save."}
            {others.map(([k, v]) => (
              <span key={k} className="block font-normal">
                {k}: {v}
              </span>
            ))}
          </span>
        ) : (
          success && <span className="text-ink-2">{successText}</span>
        )}
      </span>
    </div>
  );
}

/** Refetch these lists and the publish status (every content write republishes). */
export function invalidate(qc: QueryClient, ...keys: QueryKey[]) {
  for (const k of [...keys, ["site-status"]]) void qc.invalidateQueries({ queryKey: k });
}

/** Tabs inside a section, as links so each view has its own URL. */
export function SubNav({ items, current }: { items: { path: string; label: string }[]; current: string }) {
  return (
    <nav aria-label="Section views" className="mb-8 flex flex-wrap gap-2 border-b border-line pb-3">
      {items.map((it) => (
        <Link
          key={it.path}
          to="/dashboard/$"
          params={{ _splat: it.path }}
          aria-current={it.path === current ? "page" : undefined}
          className={`rounded-lg px-3 py-1.5 text-sm font-semibold no-underline ${it.path === current ? "bg-ink text-ground" : "text-ink-2 hover:bg-panel hover:text-ink"}`}
        >
          {it.label}
        </Link>
      ))}
    </nav>
  );
}

export function BackLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to="/dashboard/$" params={{ _splat: to }} className="link text-sm font-semibold">
      {children}
    </Link>
  );
}

/** A modal confirmation (Base UI alert dialog: focus trap, Esc to close). */
export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  pending,
  error,
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  confirmLabel: string;
  pending?: boolean;
  error?: unknown;
  onConfirm: () => void;
  onClose: () => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={(next) => !next && onClose()}>
      <AlertDialogContent className="gap-0 p-6 data-[size=default]:max-w-[min(34rem,calc(100vw-2rem))] data-[size=default]:sm:max-w-[34rem]">
        <AlertDialogHeader className="place-items-start text-left">
          <AlertDialogTitle className="text-xl font-semibold">{title}</AlertDialogTitle>
          <AlertDialogDescription render={<div />} className="mt-3 space-y-3 text-base text-ink-2">
            {children}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error ? (
          <div className="mt-4">
            <ErrorBox error={error} />
          </div>
        ) : null}
        <AlertDialogFooter className="-mx-6 -mb-6 mt-6 px-6">
          <AlertDialogCancel disabled={pending}>Cancel</AlertDialogCancel>
          <Button type="button" onClick={onConfirm} disabled={pending}>
            {pending ? "Working…" : confirmLabel}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/**
 * "Delete" with a confirmation dialog. The API refuses deletes that would lose history (a 409
 * with a message saying what to do instead); the dialog shows that message.
 */
export function DeleteButton({
  what,
  onDelete,
  onDeleted,
  children,
}: {
  what: string;
  onDelete: () => Promise<unknown>;
  onDeleted: () => void;
  children?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const del = useMutation({
    mutationFn: onDelete,
    onSuccess: () => {
      setOpen(false);
      onDeleted();
    },
  });
  return (
    <>
      <Button type="button" variant="outline" size="sm" onClick={() => setOpen(true)}>
        Delete {what}
      </Button>
      <ConfirmDialog
        open={open}
        title={`Delete this ${what}?`}
        confirmLabel={`Delete ${what}`}
        pending={del.isPending}
        error={del.error}
        onClose={() => (setOpen(false), del.reset())}
        onConfirm={() => del.mutate()}
      >
        {children ?? <p>This can't be undone.</p>}
      </ConfirmDialog>
    </>
  );
}

/** ISO (UTC) <-> <input type="datetime-local"> value in the browser's zone. */
export function toLocalInput(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  const off = d.getTimezoneOffset() * 60_000;
  return new Date(d.getTime() - off).toISOString().slice(0, 16);
}
export function fromLocalInput(v: string): string {
  return new Date(v).toISOString();
}

export const orNull = (s: string) => (s.trim() ? s.trim() : null);
export const splitList = (s: string) =>
  s
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);

export function slugify(s: string) {
  return s
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
}
