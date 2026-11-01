// Committee & roles: terms and their seats, positions and the roles each carries, explicit grants,
// and the yearly handover ("Make current"). Roles follow the committee: a seat in the CURRENT term
// whose position carries a role grants it; grants add roles on top (permanent or for one term).
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { type FormEvent, useId, useState } from "react";
import { type Schemas, api, unwrap } from "@binarysust/api-client";
import { GROUP_LABEL, POSITION_CATEGORIES, POSITION_GROUPS, type PositionGroup, type RoleName, oneOf } from "@binarysust/api-client";
import { formatShort } from "~/lib/format";
import { useMe } from "../me";
import type { SectionProps } from "../sections";
import { usePeople } from "./People";
import {
  BackLink,
  ConfirmDialog,
  Empty,
  ErrorBox,
  Field,
  Loading,
  SaveBar,
  SectionHead,
  SubNav,
  Table,
  fieldErrors,
  invalidate,
  td,
} from "../ui";
import { Button } from "@binarysust/ui/components/button";
import { Input } from "@binarysust/ui/components/input";
import { NativeSelect } from "@binarysust/ui/components/native-select";
import { Checkbox } from "@binarysust/ui/components/checkbox";

type TermRow = Schemas["people.TermRow"];
type TermIn = Schemas["people.TermIn"];
type PositionIn = Schemas["people.PositionIn"];
type PositionRoles = Schemas["access.PositionRoles"];
type Position = Schemas["people.Position"];

const VIEWS = [
  { path: "committee", label: "Terms & seats" },
  { path: "committee/positions", label: "Positions & roles" },
  { path: "committee/grants", label: "Grants" },
];

export function CommitteeSection({ rest }: SectionProps) {
  const [view, id] = rest;
  const current = view === "positions" || view === "grants" ? `committee/${view}` : "committee";
  return (
    <>
      <SectionHead title="Committee & roles" />
      <SubNav items={VIEWS} current={current} />
      {view === "positions" ? <Positions /> : view === "grants" ? <Grants /> : view === "terms" && id ? <TermDetail id={id} /> : <Terms />}
    </>
  );
}

const useTerms = () => useQuery({ queryKey: ["terms"], queryFn: async () => unwrap(await api.GET("/api/v1/terms")) });
const usePositions = () => useQuery({ queryKey: ["positions"], queryFn: async () => unwrap(await api.GET("/api/v1/positions")) });
const useRoles = () => useQuery({ queryKey: ["roles"], queryFn: async () => unwrap(await api.GET("/api/v1/roles")), staleTime: Infinity });
const roleTitle = (roles: Schemas["access.RoleInfo"][] | undefined, name: string) => roles?.find((r) => r.name === name)?.title ?? name;

// --- Terms -----------------------------------------------------------------------------------

function nextTermSlug(terms: TermRow[] | undefined): TermIn {
  const years = (terms ?? []).map((t) => Number(t.slug.slice(0, 4))).filter(Boolean);
  const y = years.length ? Math.max(...years) + 1 : new Date().getFullYear();
  const yy = String((y + 1) % 100).padStart(2, "0");
  return { slug: `${y}-${yy}`, label: `${y}–${yy}`, startsOn: `${y}-07-01` };
}

function Terms() {
  const q = useTerms();
  return (
    <div className="space-y-10">
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !q.data ? (
        <Loading what="Loading terms" />
      ) : q.data.length === 0 ? (
        <Empty>No terms yet. Create the first one below.</Empty>
      ) : (
        <Table head={["Term", "Starts", "Seats", ""]}>
          {q.data.map((t) => (
            <tr key={t.id}>
              <td className={td}>
                <Link to="/dashboard/$" params={{ _splat: `committee/terms/${t.id}` }} className="link font-semibold text-ink">
                  {t.label}
                </Link>
              </td>
              <td className={`${td} tabular`}>{formatShort(t.startsOn)}</td>
              <td className={`${td} tabular`}>{t.seats}</td>
              <td className={td}>{t.current && <span className="rounded-full bg-parrot/15 px-3 py-0.5 text-sm font-semibold">Current</span>}</td>
            </tr>
          ))}
        </Table>
      )}
      {q.data && <TermForm initial={nextTermSlug(q.data)} key={q.data.length} />}
    </div>
  );
}

function TermForm({ id, initial }: { id?: string; initial: TermIn }) {
  const qc = useQueryClient();
  const [f, setF] = useState(initial);
  const save = useMutation({
    mutationFn: async (body: TermIn) =>
      unwrap(id ? await api.PUT("/api/v1/terms/{id}", { params: { path: { id } }, body }) : await api.POST("/api/v1/terms", { body })),
    onSuccess: () => invalidate(qc, ["terms"]),
  });
  const fe = fieldErrors(save.error);
  const set = (k: keyof TermIn, v: string) => (setF((x) => ({ ...x, [k]: v })), save.reset());
  return (
    <form
      className="max-w-3xl rounded-xl border border-line bg-panel p-5"
      noValidate
      onSubmit={(e: FormEvent) => {
        e.preventDefault();
        save.mutate(f);
      }}
    >
      <h2 className="mb-4 text-lg font-semibold">{id ? "Edit term" : "New term"}</h2>
      <div className="grid gap-4 sm:grid-cols-3">
        <Field label="Slug" hint="Like 2026-27." error={fe.slug}>
          <Input value={f.slug} onChange={(e) => set("slug", e.target.value)} />
        </Field>
        <Field label="Label" error={fe.label}>
          <Input value={f.label} onChange={(e) => set("label", e.target.value)} />
        </Field>
        <Field label="Starts on" error={fe.startsOn}>
          <Input type="date" value={f.startsOn} onChange={(e) => set("startsOn", e.target.value)} />
        </Field>
      </div>
      <SaveBar
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        label={id ? "Save term" : "Create term"}
        successText={id ? "Saved." : "Term created. Open it to fill its seats."}
        shown={["slug", "label", "startsOn"]}
      />
    </form>
  );
}

function TermDetail({ id }: { id: string }) {
  const terms = useTerms();
  const term = terms.data?.find((t) => t.id === id);
  if (terms.error) return <ErrorBox error={terms.error} retry={() => void terms.refetch()} />;
  if (!terms.data) return <Loading what="Loading term" />;
  if (!term) return <Empty>That term does not exist.</Empty>;
  return (
    <div className="space-y-10">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <h2 className="text-2xl font-semibold">
          {term.label} {term.current && <span className="ml-2 rounded-full bg-parrot/15 px-3 py-0.5 align-middle text-sm font-semibold">Current</span>}
        </h2>
        <BackLink to="committee">All terms</BackLink>
      </div>
      <Handover term={term} current={terms.data.find((t) => t.current)} />
      <Seats term={term} />
      <TermForm id={term.id} initial={{ slug: term.slug, label: term.label, startsOn: term.startsOn }} key={term.id} />
    </div>
  );
}

function Handover({ term, current }: { term: TermRow; current: TermRow | undefined }) {
  const qc = useQueryClient();
  const me = useMe();
  const [open, setOpen] = useState(false);
  const make = useMutation({
    mutationFn: async () => unwrap(await api.POST("/api/v1/terms/{id}/make-current", { params: { path: { id: term.id } } })),
    onSuccess: () => {
      setOpen(false);
      // Roles may have changed for the signed-in admin too.
      invalidate(qc, ["terms"], ["people"], ["grants"], ["me"]);
    },
  });
  if (term.current)
    return <p className="text-ink-2">This is the current term. Its seats decide who holds committee roles right now.</p>;
  return (
    <div className="rounded-xl border border-line bg-panel p-5">
      <p className="text-ink-2">
        {current ? `${current.label} is the current term.` : "No term is current yet."} When {term.label}'s seats are filled,
        hand over.
      </p>
      <Button type="button" className="mt-4" onClick={() => setOpen(true)}>
        Make {term.label} current
      </Button>
      {make.isSuccess && <p className="mt-3 text-ink-2">Handed over. The public site republishes in a few minutes.</p>}
      <ConfirmDialog
        open={open}
        title={`Hand over to ${term.label}?`}
        confirmLabel={`Make ${term.label} current`}
        pending={make.isPending}
        error={make.error}
        onClose={() => (setOpen(false), make.reset())}
        onConfirm={() => make.mutate()}
      >
        <p>This is the yearly handover. One write, and on their next request:</p>
        <ul className="list-disc space-y-1 pl-5">
          <li>
            Everyone seated in <strong>{term.label}</strong> ({term.seats} {term.seats === 1 ? "seat" : "seats"}) gets the roles
            their positions carry.
          </li>
          <li>
            {current ? <>The {current.label} committee</> : "Anyone holding roles through a seat"} loses those roles and their
            dashboard sections. Nobody revokes anything by hand.
          </li>
          <li>Grants scoped to the old term stop applying; permanent grants stay.</li>
          <li>The public committee page and member statuses update with the next build.</li>
        </ul>
        {term.seats === 0 && <p className="font-semibold text-vermilion">{term.label} has no seats yet. Nobody will hold a seat-based role.</p>}
        {me.roles.length > 0 && (
          <p className="text-sm">
            If your own admin role comes from a {current?.label ?? "current"} seat rather than a permanent grant, you will lose
            access to this page too.
          </p>
        )}
      </ConfirmDialog>
    </div>
  );
}

function PersonPicker({ value, onChange, error, label = "Person" }: { value: string; onChange: (id: string) => void; error?: string; label?: string }) {
  const people = usePeople();
  const listId = useId();
  const [text, setText] = useState("");
  const options = people.data ?? [];
  return (
    <Field label={label} hint="Type a name or @handle and pick from the list." error={error ?? (text && !value ? "Pick someone from the list." : undefined)}>
      <Input
        
        list={listId}
        value={text}
        onChange={(e) => {
          const v = e.target.value;
          setText(v);
          const m = /@([a-z0-9-]+)\)?$/.exec(v);
          onChange(options.find((p) => p.handle === m?.[1])?.id ?? "");
        }}
 />
      <datalist id={listId}>
        {options.map((p) => (
          <option key={p.id} value={`${p.name} (@${p.handle})`} />
        ))}
      </datalist>
    </Field>
  );
}

function Seats({ term }: { term: TermRow }) {
  const qc = useQueryClient();
  const positions = usePositions();
  const seats = useQuery({
    queryKey: ["terms", term.id, "seats"],
    queryFn: async () => unwrap(await api.GET("/api/v1/terms/{id}/seats", { params: { path: { id: term.id } } })),
  });
  const [personId, setPersonId] = useState("");
  const [positionId, setPositionId] = useState("");
  const [pickerKey, setPickerKey] = useState(0);
  const add = useMutation({
    mutationFn: async () =>
      unwrap(await api.POST("/api/v1/terms/{id}/seats", { params: { path: { id: term.id } }, body: { personId, positionId } })),
    onSuccess: () => {
      setPersonId("");
      setPickerKey((k) => k + 1);
      invalidate(qc, ["terms"], ["people"]);
    },
  });
  const remove = useMutation({
    mutationFn: async (seatId: string) => unwrap(await api.DELETE("/api/v1/seats/{id}", { params: { path: { id: seatId } } })),
    onMutate: async (seatId) => {
      // Optimistic: drop the row now, put it back if the API refuses.
      const key = ["terms", term.id, "seats"];
      await qc.cancelQueries({ queryKey: key });
      const prev = qc.getQueryData<Schemas["people.SeatRow"][]>(key);
      qc.setQueryData(key, prev?.filter((s) => s.id !== seatId));
      return { prev };
    },
    onError: (_e, _id, ctx) => qc.setQueryData(["terms", term.id, "seats"], ctx?.prev),
    onSettled: () => invalidate(qc, ["terms"], ["people"]),
  });
  const fe = fieldErrors(add.error);

  return (
    <section aria-labelledby="seats-h" className="space-y-5">
      <h3 id="seats-h" className="text-lg font-semibold">
        Seats
      </h3>
      {seats.error ? (
        <ErrorBox error={seats.error} retry={() => void seats.refetch()} />
      ) : !seats.data ? (
        <Loading what="Loading seats" />
      ) : seats.data.length === 0 ? (
        <Empty>No seats in {term.label} yet.</Empty>
      ) : (
        <Table head={["Position", "Person", ""]}>
          {seats.data.map((s) => (
            <tr key={s.id}>
              <td className={td}>
                <span className="font-semibold">{s.positionTitle}</span>
                <span className="block text-sm text-ink-3">{GROUP_LABEL[s.group]}</span>
              </td>
              <td className={td}>
                {s.name} <span className="text-ink-3">@{s.handle}</span>
              </td>
              <td className={`${td} text-right`}>
                <Button type="button" variant="link" className="text-sm" onClick={() => remove.mutate(s.id)}>
                  Remove
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}
      {remove.error && <ErrorBox error={remove.error} />}
      <form
        className="grid max-w-3xl gap-4 rounded-xl border border-line bg-panel p-5 sm:grid-cols-[1fr_1fr_auto] sm:items-start"
        onSubmit={(e) => {
          e.preventDefault();
          add.mutate();
        }}
      >
        <PersonPicker key={pickerKey} value={personId} onChange={(v) => (setPersonId(v), add.reset())} error={fe.personId} />
        <Field label="Position" error={fe.positionId}>
          <NativeSelect value={positionId} onChange={(e) => (setPositionId(e.target.value), add.reset())}>
            <option value="">Choose…</option>
            {POSITION_GROUPS.map((g) => {
              const inGroup = positions.data?.filter((p) => p.group === g) ?? [];
              return inGroup.length === 0 ? null : (
                <optgroup key={g} label={GROUP_LABEL[g]}>
                  {inGroup.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.title}
                    </option>
                  ))}
                </optgroup>
              );
            })}
          </NativeSelect>
        </Field>
        <Button type="submit" className="sm:mt-7" disabled={add.isPending || !personId || !positionId}>
          {add.isPending ? "Adding…" : "Add seat"}
        </Button>
        {add.error && !fe.personId && !fe.positionId && <p className="font-semibold text-vermilion sm:col-span-3">{add.error.message}</p>}
      </form>
    </section>
  );
}

// --- Positions & roles -----------------------------------------------------------------------

function Positions() {
  const positions = usePositions();
  const roles = useRoles();
  const mapping = useQuery({ queryKey: ["position-roles"], queryFn: async () => unwrap(await api.GET("/api/v1/position-roles")) });
  const [editing, setEditing] = useState<string | null>(null);

  if (positions.error) return <ErrorBox error={positions.error} retry={() => void positions.refetch()} />;
  if (!positions.data || !roles.data || !mapping.data) return <Loading what="Loading positions" />;
  const rolesOf = (id: string) => mapping.data.find((m) => m.positionId === id)?.roles ?? [];
  return (
    <div className="space-y-10">
      <p className="max-w-3xl text-ink-2">
        Each position carries roles. Whoever holds a seat on that position in the <strong>current</strong> term holds those
        roles. Changing roles here takes effect on each holder's next request. Several people can hold the same position in
        a term (assistants, executive members). <em>Lifetime</em> positions make their holders lifetime members.
      </p>
      {positions.data.length === 0 ? (
        <Empty>No positions yet.</Empty>
      ) : (
        <ul className="space-y-4">
          {positions.data.map((p) => (
            <li key={p.id} className="rounded-xl border border-line bg-panel p-5">
              {editing === p.id ? (
                <PositionForm id={p.id} initial={p} onDone={() => setEditing(null)} />
              ) : (
                <div className="flex flex-wrap items-baseline justify-between gap-3">
                  <p>
                    <span className="text-lg font-semibold">{p.title}</span>{" "}
                    <span className="text-sm text-ink-3">
                      {GROUP_LABEL[p.group]} · {p.category} · rank {p.rank} · {p.slug}
                      {p.lifetime && " · lifetime"}
                    </span>
                  </p>
                  <Button type="button" variant="link" className="text-sm" onClick={() => setEditing(p.id)}>
                    Edit
                  </Button>
                </div>
              )}
              <PositionRolesEditor positionId={p.id} title={p.title} current={rolesOf(p.id)} roles={roles.data} />
            </li>
          ))}
        </ul>
      )}
      <div className="max-w-3xl rounded-xl border border-line p-5">
        <h2 className="mb-4 text-lg font-semibold">New position</h2>
        <PositionForm
          initial={{ title: "", slug: "", category: "executive", group: "member", lifetime: false, rank: (positions.data.filter((p) => p.category === "executive").at(-1)?.rank ?? 0) + 10 }}
          key={positions.data.length}
        />
      </div>
    </div>
  );
}

function PositionForm({ id, initial, onDone }: { id?: string; initial: PositionIn | Position; onDone?: () => void }) {
  const qc = useQueryClient();
  const [f, setF] = useState<Required<PositionIn>>({
    title: initial.title,
    slug: initial.slug,
    category: initial.category,
    group: initial.group ?? (initial.category === "advisor" ? "adviser" : "member"),
    rank: initial.rank,
    lifetime: initial.lifetime ?? false,
  });
  const save = useMutation({
    mutationFn: async (body: PositionIn) =>
      unwrap(id ? await api.PUT("/api/v1/positions/{id}", { params: { path: { id } }, body }) : await api.POST("/api/v1/positions", { body })),
    onSuccess: () => {
      invalidate(qc, ["positions"], ["position-roles"]);
      onDone?.();
    },
  });
  const fe = fieldErrors(save.error);
  const set = <K extends keyof PositionIn>(k: K, v: PositionIn[K]) => (setF((x) => ({ ...x, [k]: v })), save.reset());
  return (
    <form
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate(f);
      }}
    >
      <div className="grid gap-4 sm:grid-cols-3">
        <Field label="Title" error={fe.title}>
          <Input value={f.title} onChange={(e) => set("title", e.target.value)} />
        </Field>
        <Field label="Slug" error={fe.slug}>
          <Input value={f.slug} onChange={(e) => set("slug", e.target.value)} />
        </Field>
        <Field label="Rank" hint="Lower is listed first." error={fe.rank}>
          <Input type="number" value={f.rank} onChange={(e) => set("rank", Number(e.target.value))} />
        </Field>
        <Field label="Committee" error={fe.category}>
          <NativeSelect value={f.category} onChange={(e) => set("category", oneOf(POSITION_CATEGORIES, e.target.value))}>
            <option value="executive">Executive</option>
            <option value="advisor">Advisory</option>
          </NativeSelect>
        </Field>
        <Field label="Group" hint="How the public committee page groups seats." error={fe.group}>
          <NativeSelect value={f.group} onChange={(e) => set("group", oneOf(POSITION_GROUPS, e.target.value))}>
            {POSITION_GROUPS.map((g: PositionGroup) => (
              <option key={g} value={g}>
                {GROUP_LABEL[g]}
              </option>
            ))}
          </NativeSelect>
        </Field>
        <label className="flex items-center gap-3 sm:mt-7">
          <Checkbox className="size-5" checked={f.lifetime} onCheckedChange={(checked) => set("lifetime", checked)} />
          Lifetime: holders become lifetime members
        </label>
      </div>
      <SaveBar pending={save.isPending} error={save.error} success={save.isSuccess && !id} label={id ? "Save position" : "Create position"} shown={["title", "slug", "category", "group", "rank"]}>
        {onDone && (
          <Button type="button" variant="outline" onClick={onDone}>
            Cancel
          </Button>
        )}
      </SaveBar>
    </form>
  );
}

function PositionRolesEditor({ positionId, title, current, roles }: { positionId: string; title: string; current: RoleName[]; roles: Schemas["access.RoleInfo"][] }) {
  const qc = useQueryClient();
  const set = useMutation({
    mutationFn: async (next: RoleName[]) =>
      unwrap(await api.PUT("/api/v1/positions/{id}/roles", { params: { path: { id: positionId } }, body: { roles: next } })),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey: ["position-roles"] });
      const prev = qc.getQueryData<PositionRoles[]>(["position-roles"]);
      qc.setQueryData<PositionRoles[]>(["position-roles"], (rows) => {
        const others = (rows ?? []).filter((r) => r.positionId !== positionId);
        return [...others, { positionId, roles: next }];
      });
      return { prev };
    },
    onError: (_e, _v, ctx) => qc.setQueryData(["position-roles"], ctx?.prev),
    onSettled: () => invalidate(qc, ["position-roles"], ["me"]),
  });
  const toggle = (name: RoleName, on: boolean) => set.mutate(on ? [...current, name] : current.filter((r) => r !== name));
  return (
    <fieldset className="mt-4">
      <legend className="mb-2 text-sm font-semibold text-ink-2">Roles this position carries</legend>
      <div className="flex flex-wrap gap-x-5 gap-y-2">
        {roles.map((r) => (
          <label key={r.name} className="flex items-center gap-2 text-sm" title={r.allPermissions ? "Every permission" : r.permissions.join(", ")}>
            <Checkbox className="size-4" checked={current.includes(r.name)} onCheckedChange={(checked) => toggle(r.name, checked)} aria-label={`${title}: ${r.title}`} />
            {r.title}
          </label>
        ))}
      </div>
      {set.error && <p className="mt-2 text-sm font-semibold text-vermilion">{set.error.message}</p>}
    </fieldset>
  );
}

// --- Grants ----------------------------------------------------------------------------------

function Grants() {
  const qc = useQueryClient();
  const grants = useQuery({ queryKey: ["grants"], queryFn: async () => unwrap(await api.GET("/api/v1/grants")) });
  const roles = useRoles();
  const terms = useTerms();
  const [personId, setPersonId] = useState("");
  const [role, setRole] = useState<RoleName | "">("");
  const [termId, setTermId] = useState("");
  const [pickerKey, setPickerKey] = useState(0);
  const create = useMutation({
    mutationFn: async () => {
      if (!role) throw new Error("Choose a role.");
      return unwrap(await api.POST("/api/v1/grants", { body: { personId, role, termId: termId || null } }));
    },
    onSuccess: () => {
      setPersonId("");
      setPickerKey((k) => k + 1);
      invalidate(qc, ["grants"], ["me"]);
    },
  });
  const remove = useMutation({
    mutationFn: async (id: string) => unwrap(await api.DELETE("/api/v1/grants/{id}", { params: { path: { id } } })),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: ["grants"] });
      const prev = qc.getQueryData<Schemas["access.Grant"][]>(["grants"]);
      qc.setQueryData(["grants"], prev?.filter((g) => g.id !== id));
      return { prev };
    },
    onError: (_e, _id, ctx) => qc.setQueryData(["grants"], ctx?.prev),
    onSettled: () => invalidate(qc, ["grants"], ["me"]),
  });
  const fe = fieldErrors(create.error);

  return (
    <div className="space-y-10">
      <p className="max-w-3xl text-ink-2">
        Grants give a role to someone without a seat: a check-in volunteer for one term, or a permanent administrator. Prefer
        seats where you can, so roles hand over with the committee.
      </p>
      {grants.error ? (
        <ErrorBox error={grants.error} retry={() => void grants.refetch()} />
      ) : !grants.data ? (
        <Loading what="Loading grants" />
      ) : grants.data.length === 0 ? (
        <Empty>No grants.</Empty>
      ) : (
        <Table head={["Person", "Role", "Applies", "Granted", ""]}>
          {grants.data.map((g) => (
            <tr key={g.id}>
              <td className={td}>
                {g.personName} <span className="text-ink-3">@{g.personHandle}</span>
              </td>
              <td className={td}>{roleTitle(roles.data, g.role)}</td>
              <td className={td}>{g.termSlug ? `During ${g.termSlug}` : "Permanent"}</td>
              <td className={`${td} tabular`}>{formatShort(g.createdAt)}</td>
              <td className={`${td} text-right`}>
                <Button type="button" variant="link" className="text-sm" onClick={() => remove.mutate(g.id)}>
                  Revoke
                </Button>
              </td>
            </tr>
          ))}
        </Table>
      )}
      {remove.error && <ErrorBox error={remove.error} />}
      <form
        className="max-w-3xl rounded-xl border border-line bg-panel p-5"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
      >
        <h2 className="mb-4 text-lg font-semibold">New grant</h2>
        <div className="grid gap-4 sm:grid-cols-3">
          <PersonPicker key={pickerKey} value={personId} onChange={(v) => (setPersonId(v), create.reset())} error={fe.personId} />
          <Field label="Role" error={fe.role}>
            <NativeSelect value={role} onChange={(e) => (setRole(roles.data?.find((r) => r.name === e.target.value)?.name ?? ""), create.reset())}>
              <option value="">Choose…</option>
              {roles.data?.map((r) => (
                <option key={r.name} value={r.name}>
                  {r.title}
                </option>
              ))}
            </NativeSelect>
          </Field>
          <Field label="Applies" error={fe.termId}>
            <NativeSelect value={termId} onChange={(e) => (setTermId(e.target.value), create.reset())}>
              <option value="">Permanently</option>
              {terms.data?.map((t) => (
                <option key={t.id} value={t.id}>
                  During {t.label}
                </option>
              ))}
            </NativeSelect>
          </Field>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-4">
          <Button type="submit" disabled={create.isPending || !personId || !role}>
            {create.isPending ? "Granting…" : "Grant role"}
          </Button>
          {create.error && !fe.personId && !fe.role && !fe.termId && <span className="font-semibold text-vermilion">{create.error.message}</span>}
        </div>
      </form>
    </div>
  );
}
