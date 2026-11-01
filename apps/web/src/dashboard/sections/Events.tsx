import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { KIND_LABEL } from "~/components/events";
import { Icon } from "~/components/Icon";
import { API_BASE, type Schemas, api, unwrap } from "@binarysust/api-client";
import { EVENT_KINDS, EVENT_STATUSES, oneOf } from "@binarysust/api-client";
import { formatShort, formatTime } from "~/lib/format";
import type { EventKind } from "~/lib/types";
import { ImageField, UploadButton } from "../MediaUpload";
import { HandleListPicker } from "../PersonLookup";
import { useMe } from "../me";
import type { SectionProps } from "../sections";
import {
  BackLink,
  DeleteButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  SaveBar,
  SectionHead,
  SubNav,
  Table,
  errorFor,
  fieldErrors,
  fromLocalInput,
  invalidate,
  slugify,
  td,
  toLocalInput,
} from "../ui";
import { Button, buttonVariants } from "@binarysust/ui/components/button";
import { Input } from "@binarysust/ui/components/input";
import { NativeSelect } from "@binarysust/ui/components/native-select";
import { Textarea } from "@binarysust/ui/components/textarea";
import { Checkbox } from "@binarysust/ui/components/checkbox";

type EventInput = Schemas["events.EventInput"];
type Event = Schemas["events.Event"];
const kindLabel = (k: string) => KIND_LABEL[k as EventKind] ?? k;

export function EventsSection({ rest }: SectionProps) {
  const [id, view] = rest;
  if (!id) return <EventList />;
  if (id !== "new" && view === "registrants") return <Registrants id={id} />;
  return <EventEditor id={id === "new" ? null : id} />;
}

function EventList() {
  const q = useQuery({ queryKey: ["events"], queryFn: async () => unwrap(await api.GET("/api/v1/events")) });
  return (
    <>
      <SectionHead title="Events">
        <Link to="/dashboard/$" params={{ _splat: "events/new" }} className={buttonVariants()}>
          <Icon name="plus" size={18} /> New event
        </Link>
      </SectionHead>
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !q.data ? (
        <Loading what="Loading events" />
      ) : q.data.length === 0 ? (
        <Empty>No events yet. Create the first one.</Empty>
      ) : (
        <Table head={["Event", "Kind", "Starts", "Status", "RSVPs", ""]}>
          {q.data.map((e) => (
            <tr key={e.id}>
              <td className={td}>
                <Link to="/dashboard/$" params={{ _splat: `events/${e.id}` }} className="link font-semibold text-ink">
                  {e.title}
                </Link>
                <span className="block text-sm text-ink-3">/{e.slug}</span>
              </td>
              <td className={td}>{kindLabel(e.kind)}</td>
              <td className={`${td} tabular`}>
                {formatShort(e.startsAt)}, {formatTime(e.startsAt)}
              </td>
              <td className={`${td} capitalize`}>{e.status}</td>
              <td className={`${td} tabular`}>
                {e.confirmed}
                {e.registration?.capacity != null && ` / ${e.registration.capacity}`}
                {e.waitlisted > 0 && <span className="block text-sm text-ink-3">+{e.waitlisted} waiting</span>}
              </td>
              <td className={td}>
                {e.registration && (
                  <Link to="/dashboard/$" params={{ _splat: `events/${e.id}/registrants` }} className="link text-sm">
                    Registrants
                  </Link>
                )}
              </td>
            </tr>
          ))}
        </Table>
      )}
    </>
  );
}

const blank: EventInput = {
  slug: "",
  title: "",
  kind: "workshop",
  status: "draft",
  summary: "",
  bodyMarkdown: "",
  recapMarkdown: "",
  startsAt: "",
  endsAt: "",
  venue: null,
  onlineUrl: null,
  coverUrl: null,
  photos: [],
  speakers: [],
  registration: null,
};

const toInput = (e: Event): EventInput => ({
  slug: e.slug,
  title: e.title,
  kind: e.kind,
  status: e.status,
  summary: e.summary,
  bodyMarkdown: e.bodyMarkdown,
  recapMarkdown: e.recapMarkdown,
  startsAt: e.startsAt,
  endsAt: e.endsAt,
  venue: e.venue,
  onlineUrl: e.onlineUrl,
  coverUrl: e.coverUrl,
  photos: e.photos,
  speakers: e.speakers,
  registration: e.registration,
});

function useEvent(id: string | null) {
  return useQuery({
    queryKey: ["events", id],
    queryFn: async () => unwrap(await api.GET("/api/v1/events/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== null,
  });
}

function EventEditor({ id }: { id: string | null }) {
  const q = useEvent(id);
  if (id !== null && q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (id !== null && !q.data) return <Loading what="Loading event" />;
  return <EventForm id={id} initial={q.data ? toInput(q.data) : blank} event={q.data} key={id ?? "new"} />;
}

/** Checks the browser can make before a round trip; the API re-checks everything. */
function validate(e: EventInput): string | null {
  if (!e.startsAt || !e.endsAt) return "Set when the event starts and ends.";
  if (e.registration && (!e.registration.opensAt || !e.registration.closesAt))
    return "Registration needs both an opening and a closing time.";
  return null;
}

function EventForm({ id, initial, event }: { id: string | null; initial: EventInput; event?: Event }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [f, setF] = useState<EventInput>(initial);
  const [problem, setProblem] = useState<string | null>(null);
  const hasRegistration = !!event?.registration;

  const save = useMutation({
    mutationFn: async (body: EventInput) =>
      unwrap(id ? await api.PUT("/api/v1/events/{id}", { params: { path: { id } }, body }) : await api.POST("/api/v1/events", { body })),
    onSuccess: (saved) => {
      qc.setQueryData(["events", saved.id], saved);
      invalidate(qc, ["events"]);
      if (!id) void navigate({ to: "/dashboard/$", params: { _splat: `events/${saved.id}` } });
    },
  });
  const fe = fieldErrors(save.error);
  const set = <K extends keyof EventInput>(k: K, v: EventInput[K]) => {
    setF((x) => ({ ...x, [k]: v }));
    save.reset();
  };
  const setReg = (patch: Partial<NonNullable<EventInput["registration"]>>) =>
    set("registration", { opensAt: "", closesAt: "", capacity: null, ...f.registration, ...patch });
  const photos = f.photos ?? [];

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const err = validate(f);
    setProblem(err);
    if (!err) save.mutate(f);
  };

  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-8" noValidate>
      <SectionHead title={id ? "Edit event" : "New event"}>
        <BackLink to="events">Back to events</BackLink>
      </SectionHead>
      {id && hasRegistration && (
        <SubNav
          current={`events/${id}`}
          items={[
            { path: `events/${id}`, label: "Details" },
            { path: `events/${id}/registrants`, label: "Registrants & check-in" },
          ]}
        />
      )}

      <div className="grid gap-5 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Title" error={fe.title}>
            <Input
              
              value={f.title}
              onChange={(e) => {
                const title = e.target.value;
                setF((x) => ({ ...x, title, slug: id || x.slug !== slugify(x.title) ? x.slug : slugify(title) }));
              }}
 />
          </Field>
        </div>
        <Field label="URL slug" hint={`/events/${f.slug || "…"}`} error={fe.slug}>
          <Input value={f.slug} onChange={(e) => set("slug", e.target.value)} />
        </Field>
        <Field label="Kind" error={fe.kind}>
          <NativeSelect value={f.kind} onChange={(e) => set("kind", oneOf(EVENT_KINDS, e.target.value))}>
            {Object.entries(KIND_LABEL).map(([k, label]) => (
              <option key={k} value={k}>
                {label}
              </option>
            ))}
          </NativeSelect>
        </Field>
        <Field label="Starts" error={fe.startsAt}>
          <Input type="datetime-local" value={toLocalInput(f.startsAt)} onChange={(e) => set("startsAt", e.target.value ? fromLocalInput(e.target.value) : "")} />
        </Field>
        <Field label="Ends" error={fe.endsAt}>
          <Input type="datetime-local" value={toLocalInput(f.endsAt)} onChange={(e) => set("endsAt", e.target.value ? fromLocalInput(e.target.value) : "")} />
        </Field>
        <Field label="Venue" error={fe.venue}>
          <Input value={f.venue ?? ""} onChange={(e) => set("venue", e.target.value || null)} />
        </Field>
        <Field label="Online link" error={fe.onlineUrl}>
          <Input type="url" value={f.onlineUrl ?? ""} onChange={(e) => set("onlineUrl", e.target.value || null)} />
        </Field>
        <div className="sm:col-span-2">
          <ImageField label="Cover image" purpose="event" value={f.coverUrl ?? null} onChange={(v) => set("coverUrl", v)} error={fe.coverUrl} hint="Resized to WebP, at most 1600 px." />
        </div>
        <div className="sm:col-span-2">
          <Field label="Summary" hint="One or two sentences, shown on lists and link previews." error={fe.summary}>
            <Textarea className="min-h-20" value={f.summary} onChange={(e) => set("summary", e.target.value)} maxLength={280} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Details" hint="Markdown." error={fe.bodyMarkdown}>
            <Textarea className="min-h-48 text-sm" value={f.bodyMarkdown} onChange={(e) => set("bodyMarkdown", e.target.value)} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <HandleListPicker label="Speakers" error={errorFor(fe, "speakers", true)} value={f.speakers ?? []} onChange={(v) => set("speakers", v)} />
        </div>
      </div>

      <fieldset className="rounded-lg border border-line p-5">
        <legend className="px-1 font-semibold">Registration</legend>
        <label className="flex items-center gap-3">
          <Checkbox
            className="size-5"
            checked={f.registration !== null}
            onCheckedChange={(checked) => set("registration", checked ? { opensAt: "", closesAt: "", capacity: null } : null)}
          />
          Require RSVP (window and capacity are set together)
        </label>
        {fe.registration && <p className="mt-2 text-sm font-semibold text-vermilion">{fe.registration}</p>}
        {f.registration && (
          <div className="mt-5 grid gap-5 sm:grid-cols-3">
            <Field label="Opens" error={fe["registration.opensAt"]}>
              <Input type="datetime-local" value={toLocalInput(f.registration.opensAt)} onChange={(e) => setReg({ opensAt: e.target.value ? fromLocalInput(e.target.value) : "" })} />
            </Field>
            <Field label="Closes" error={fe["registration.closesAt"]}>
              <Input type="datetime-local" value={toLocalInput(f.registration.closesAt)} onChange={(e) => setReg({ closesAt: e.target.value ? fromLocalInput(e.target.value) : "" })} />
            </Field>
            <Field label="Capacity" hint="Empty means no limit." error={fe["registration.capacity"]}>
              <Input
                
                type="number"
                min={1}
                inputMode="numeric"
                value={f.registration.capacity ?? ""}
                onChange={(e) => setReg({ capacity: e.target.value ? Number(e.target.value) : null })}
 />
            </Field>
          </div>
        )}
      </fieldset>

      <fieldset className="space-y-4 rounded-lg border border-line p-5">
        <legend className="px-1 font-semibold">After the event</legend>
        <Field label="Recap" hint="Markdown, shown on the event page once it has happened." error={fe.recapMarkdown}>
          <Textarea className="min-h-32 text-sm" value={f.recapMarkdown ?? ""} onChange={(e) => set("recapMarkdown", e.target.value)} />
        </Field>
        <div>
          <p className="mb-2 text-sm font-semibold text-ink-2">Photos</p>
          {(fe.photos || errorFor(fe, "photos", true)) && <p className="mb-2 text-sm font-semibold text-vermilion">{errorFor(fe, "photos", true)}</p>}
          <ul className="space-y-3">
            {photos.map((p, i) => (
              <li key={p.url + i} className="flex items-center gap-3">
                <img src={p.url} alt="" className="h-14 w-20 shrink-0 rounded border border-line object-cover" />
                <Input
                  
                  aria-label={`Caption for photo ${i + 1}`}
                  placeholder="Caption"
                  value={p.caption}
                  onChange={(e) => set("photos", photos.map((x, j) => (j === i ? { ...x, caption: e.target.value } : x)))}
 />
                <Button type="button" variant="outline" className="px-3" aria-label={`Remove photo ${i + 1}`} onClick={() => set("photos", photos.filter((_, j) => j !== i))}>
                  <Icon name="close" size={18} />
                </Button>
              </li>
            ))}
          </ul>
          <div className="mt-3">
            <UploadButton purpose="event" label="Add photo" onUploaded={(o) => set("photos", [...photos, { url: o.url, caption: "" }])} />
          </div>
        </div>
      </fieldset>

      <Field label="Status" error={fe.status}>
        <NativeSelect className="max-w-xs" value={f.status} onChange={(e) => set("status", oneOf(EVENT_STATUSES, e.target.value))}>
          <option value="draft">Draft (not public)</option>
          <option value="published">Published</option>
          <option value="cancelled">Cancelled (stays public, marked cancelled)</option>
        </NativeSelect>
      </Field>

      <SaveBar
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        problem={problem}
        label={id ? "Save changes" : "Create event"}
        shown={["title", "slug", "kind", "startsAt", "endsAt", "venue", "onlineUrl", "coverUrl", "summary", "bodyMarkdown", "speakers", "registration", "recapMarkdown", "photos", "status"]}
      >
        {id && event && (
          <span className="ml-auto">
            <DeleteButton
              what="event"
              onDelete={async () => unwrap(await api.DELETE("/api/v1/events/{id}", { params: { path: { id } } }))}
              onDeleted={() => {
                invalidate(qc, ["events"]);
                void navigate({ to: "/dashboard/$", params: { _splat: "events" } });
              }}
            >
              {event.status !== "draft" && event.rsvpCount > 0 ? (
                <p>
                  {event.rsvpCount} {event.rsvpCount === 1 ? "person is" : "people are"} registered, so the API will refuse. Set
                  the status to <strong>Cancelled</strong> instead: the page stays up, marked cancelled.
                </p>
              ) : (
                <p>It disappears from the dashboard{event.status === "draft" ? "" : " and, after the next build, from the public site"}. This can't be undone.</p>
              )}
            </DeleteButton>
          </span>
        )}
      </SaveBar>
    </form>
  );
}

// --- Registrants & check-in ------------------------------------------------------------------

const csvUrl = (slug: string) => `${API_BASE}/events/${encodeURIComponent(slug)}/registrants.csv`;

function Registrants({ id }: { id: string }) {
  const ev = useEvent(id);
  const list = useQuery({
    queryKey: ["events", id, "registrants"],
    queryFn: async () => unwrap(await api.GET("/api/v1/events/{id}/registrants", { params: { path: { id } } })),
  });
  const [filter, setFilter] = useState("");

  if (ev.error) return <ErrorBox error={ev.error} retry={() => void ev.refetch()} />;
  if (!ev.data) return <Loading what="Loading event" />;
  const e = ev.data;
  const needle = filter.trim().toLowerCase();
  const rows = (list.data ?? []).filter((r) => !needle || `${r.name} ${r.email} ${r.handle ?? ""} ${r.ticket}`.toLowerCase().includes(needle));

  return (
    <>
      <SectionHead title={e.title}>
        <BackLink to="events">Back to events</BackLink>
      </SectionHead>
      <SubNav
        current={`events/${id}/registrants`}
        items={[
          { path: `events/${id}`, label: "Details" },
          { path: `events/${id}/registrants`, label: "Registrants & check-in" },
        ]}
      />
      <p className="mb-6 text-ink-2 tabular">
        {e.confirmed} confirmed{e.registration?.capacity != null && ` of ${e.registration.capacity}`} · {e.waitlisted} on the waitlist · {e.checkedIn} checked in
      </p>

      <CheckIn slug={e.slug} onDone={() => (void list.refetch(), void ev.refetch())} />

      <div className="mt-10 mb-4 flex flex-wrap items-end justify-between gap-4">
        <h2 className="text-lg font-semibold">Registrants</h2>
        <div className="flex flex-wrap items-center gap-3">
          <Input className="w-64" type="search" placeholder="Filter by name, email or ticket" aria-label="Filter registrants" value={filter} onChange={(x) => setFilter(x.target.value)} />
          <a href={csvUrl(e.slug)} download className={buttonVariants({ variant: "outline", size: "sm" })}>
            <Icon name="file" size={16} /> Download CSV
          </a>
        </div>
      </div>
      {list.error ? (
        <ErrorBox error={list.error} retry={() => void list.refetch()} />
      ) : !list.data ? (
        <Loading what="Loading registrants" />
      ) : rows.length === 0 ? (
        <Empty>{list.data.length === 0 ? "Nobody has registered yet." : "No registrant matches that filter."}</Empty>
      ) : (
        <Table head={["Name", "Status", "Ticket", "Registered", "Checked in"]}>
          {rows.map((r) => (
            <tr key={r.ticket || r.email}>
              <td className={td}>
                <span className="font-semibold">{r.name || r.email}</span>
                <span className="block text-sm text-ink-3">
                  {r.email}
                  {r.handle && ` · @${r.handle}`}
                </span>
              </td>
              <td className={`${td} capitalize`}>{r.status}</td>
              <td className={`${td} font-[var(--font-code)] text-sm`}>{r.ticket || "—"}</td>
              <td className={`${td} tabular text-sm`}>{`${formatShort(r.registeredAt)} ${formatTime(r.registeredAt)}`}</td>
              <td className={`${td} tabular text-sm`}>{r.checkedInAt ? formatTime(r.checkedInAt) : "—"}</td>
            </tr>
          ))}
        </Table>
      )}
    </>
  );
}

function CheckIn({ slug, onDone }: { slug: string; onDone: () => void }) {
  const me = useMe();
  const [ticket, setTicket] = useState("");
  const check = useMutation({
    mutationFn: async (t: string) => unwrap(await api.POST("/api/v1/events/{slug}/checkin", { params: { path: { slug } }, body: { ticket: t } })),
    onSuccess: () => {
      setTicket("");
      onDone();
    },
  });
  if (!me.permissions.includes("events.checkin")) return null;
  const fe = fieldErrors(check.error);
  return (
    <form
      className="rounded-xl border border-line bg-panel p-5"
      onSubmit={(e) => {
        e.preventDefault();
        if (ticket.trim()) check.mutate(ticket.trim());
      }}
    >
      <h2 className="mb-3 text-lg font-semibold">Check in</h2>
      <div className="flex flex-wrap items-end gap-3">
        <div className="min-w-0 flex-1">
          <Field label="Ticket code" hint="Type or scan the code from the attendee's ticket." error={fe.ticket}>
            <Input className="font-[var(--font-code)] uppercase" autoComplete="off" value={ticket} onChange={(e) => (setTicket(e.target.value), check.reset())} />
          </Field>
        </div>
        <Button type="submit" disabled={check.isPending || !ticket.trim()}>
          {check.isPending ? "Checking…" : "Check in"}
        </Button>
      </div>
      <div aria-live="polite" className="mt-3">
        {check.data && (
          <p className="flex items-center gap-2 font-semibold">
            <Icon name={check.data.alreadyCheckedIn ? "alert" : "check"} size={16} /> {check.data.name} ({check.data.email}){" "}
            {check.data.alreadyCheckedIn ? `was already checked in at ${formatTime(check.data.checkedInAt)}.` : "is checked in."}
          </p>
        )}
        {check.error && !fe.ticket && <p className="font-semibold text-vermilion">{check.error.message}</p>}
      </div>
    </form>
  );
}
