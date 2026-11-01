// People: one person row for life. Status (committee, advisor, member, alumni, guest) is derived by
// the API from affiliation, memberSince, graduatedOn and current-term seats; it is never edited here.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { Icon } from "~/components/Icon";
import { type Schemas, api, unwrap } from "~/lib/api/client";
import { type Affiliation, AFFILIATIONS as AFFILIATION_VALUES, PERSON_STATUSES, oneOf } from "~/lib/api/enums";
import { ImageField } from "../MediaUpload";
import type { SectionProps } from "../sections";
import { BackLink, DeleteButton, Empty, ErrorBox, Field, Loading, SaveBar, SectionHead, Table, fieldErrors, invalidate, orNull, td } from "../ui";

type PersonIn = Schemas["people.PersonIn"];
type Person = Schemas["people.Person"];

const AFFILIATION_LABEL: Record<Affiliation, string> = {
  student: "Student (or former student)",
  faculty: "Faculty",
  guest: "Guest (speaker, alumnus of another university)",
};

export function usePeople() {
  return useQuery({ queryKey: ["people"], queryFn: async () => unwrap(await api.GET("/api/v1/people")) });
}

export function PeopleSection({ rest }: SectionProps) {
  const [id] = rest;
  if (!id) return <PeopleList />;
  return <PersonEditor id={id === "new" ? null : id} />;
}

function PeopleList() {
  const q = usePeople();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const needle = search.trim().toLowerCase();
  const rows = q.data?.filter(
    (p) => (!status || p.status === status) && (!needle || `${p.name} ${p.handle} ${p.email ?? ""}`.toLowerCase().includes(needle)),
  );
  return (
    <>
      <SectionHead title="People">
        <Link to="/dashboard/$" params={{ _splat: "people/new" }} className="btn btn-primary">
          <Icon name="plus" size={18} /> Add person
        </Link>
      </SectionHead>
      <div className="mb-5 flex flex-wrap gap-3">
        <input className="field !w-72" type="search" placeholder="Search name, handle or email" aria-label="Search people" value={search} onChange={(e) => setSearch(e.target.value)} />
        <select className="field !w-48" aria-label="Filter by status" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Every status</option>
          {PERSON_STATUSES.map((s) => (
            <option key={s} value={s}>
              {s[0]?.toUpperCase() + s.slice(1)}
            </option>
          ))}
        </select>
      </div>
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !rows ? (
        <Loading what="Loading people" />
      ) : rows.length === 0 ? (
        <Empty>{q.data?.length ? "Nobody matches that search." : "No people yet."}</Empty>
      ) : (
        <Table head={["Name", "Email", "Affiliation", "Status", "Listed"]}>
          {rows.map((p) => (
            <tr key={p.id}>
              <td className={td}>
                <Link to="/dashboard/$" params={{ _splat: `people/${p.id}` }} className="link font-semibold text-ink">
                  {p.name}
                </Link>
                <span className="block text-sm text-ink-3">@{p.handle}</span>
              </td>
              <td className={td}>{p.email ?? "—"}</td>
              <td className={`${td} capitalize`}>{p.affiliation}</td>
              <td className={`${td} capitalize`}>{p.status}</td>
              <td className={td}>{p.listed ? "Yes" : "No"}</td>
            </tr>
          ))}
        </Table>
      )}
      <p className="mt-4 text-sm text-ink-3">{q.data && `${rows?.length ?? 0} of ${q.data.length} people.`}</p>
    </>
  );
}

const blank: PersonIn = {
  handle: "",
  name: "",
  email: null,
  affiliation: "student",
  listed: true,
  avatarUrl: null,
  headline: "",
  department: "CSE",
  batch: null,
  memberSince: null,
  graduatedOn: null,
};

const toIn = ({ id: _i, status: _s, updatedAt: _u, ...rest }: Person): PersonIn => rest;

function PersonEditor({ id }: { id: string | null }) {
  const q = useQuery({
    queryKey: ["people", id],
    queryFn: async () => unwrap(await api.GET("/api/v1/people/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== null,
  });
  if (id && q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (id && !q.data) return <Loading what="Loading person" />;
  return <PersonForm id={id} initial={q.data ? toIn(q.data) : blank} status={q.data?.status} key={id ?? "new"} />;
}

function PersonForm({ id, initial, status }: { id: string | null; initial: PersonIn; status?: string }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [f, setF] = useState<PersonIn>(initial);
  const save = useMutation({
    mutationFn: async (body: PersonIn) =>
      unwrap(id ? await api.PUT("/api/v1/people/{id}", { params: { path: { id } }, body }) : await api.POST("/api/v1/people", { body })),
    onSuccess: (p) => {
      qc.setQueryData(["people", p.id], p);
      invalidate(qc, ["people"]);
      if (!id) void navigate({ to: "/dashboard/$", params: { _splat: `people/${p.id}` } });
    },
  });
  const fe = fieldErrors(save.error);
  const set = <K extends keyof PersonIn>(k: K, v: PersonIn[K]) => {
    setF((x) => ({ ...x, [k]: v }));
    save.reset();
  };
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate(f);
  };

  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-8" noValidate>
      <SectionHead title={id ? initial.name : "Add person"}>
        <BackLink to="people">Back to people</BackLink>
      </SectionHead>
      {status && (
        <p className="-mt-4 text-ink-2">
          Status today: <strong className="capitalize text-ink">{status}</strong>. It follows from affiliation, dates and
          current committee seats.
        </p>
      )}
      <div className="grid gap-5 sm:grid-cols-2">
        <Field label="Name" error={fe.name}>
          <input className="field" value={f.name} onChange={(e) => set("name", e.target.value)} />
        </Field>
        <Field label="Handle" hint={id ? "Changing it keeps the old URL working as a redirect." : `/people/${f.handle || "…"}`} error={fe.handle}>
          <input className="field" value={f.handle} onChange={(e) => set("handle", e.target.value.toLowerCase())} />
        </Field>
        <Field label="Email" hint="Their Google sign-in links to this person when the emails match." error={fe.email}>
          <input className="field" type="email" value={f.email ?? ""} onChange={(e) => set("email", orNull(e.target.value))} />
        </Field>
        <Field label="Affiliation" error={fe.affiliation}>
          <select className="field" value={f.affiliation} onChange={(e) => set("affiliation", oneOf(AFFILIATION_VALUES, e.target.value))}>
            {AFFILIATION_VALUES.map((a) => (
              <option key={a} value={a}>
                {AFFILIATION_LABEL[a]}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Department" error={fe.department}>
          <input className="field" value={f.department} onChange={(e) => set("department", e.target.value)} />
        </Field>
        <Field label="Batch" hint="Admission year, e.g. 2021." error={fe.batch}>
          <input className="field" type="number" inputMode="numeric" value={f.batch ?? ""} onChange={(e) => set("batch", e.target.value ? Number(e.target.value) : null)} />
        </Field>
        <Field label="Member since" hint="Set for students who joined the club." error={fe.memberSince}>
          <input className="field" type="date" value={f.memberSince ?? ""} onChange={(e) => set("memberSince", orNull(e.target.value))} />
        </Field>
        <Field label="Graduated on" hint="On or before today makes a student an alumnus." error={fe.graduatedOn}>
          <input className="field" type="date" value={f.graduatedOn ?? ""} onChange={(e) => set("graduatedOn", orNull(e.target.value))} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Headline" error={fe.headline}>
            <input className="field" value={f.headline} onChange={(e) => set("headline", e.target.value)} maxLength={160} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <ImageField label="Photo" purpose="avatar" value={f.avatarUrl} onChange={(v) => set("avatarUrl", v)} error={fe.avatarUrl} />
        </div>
        <label className="flex items-center gap-3 sm:col-span-2">
          <input type="checkbox" className="size-5 accent-[var(--vermilion)]" checked={f.listed} onChange={(e) => set("listed", e.target.checked)} />
          Listed in the public directory
        </label>
      </div>
      <SaveBar
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        label={id ? "Save changes" : "Add person"}
        shown={["name", "handle", "email", "affiliation", "department", "batch", "memberSince", "graduatedOn", "headline", "avatarUrl"]}
      >
        {id && (
          <span className="ml-auto">
            <DeleteButton
              what="person"
              onDelete={async () => unwrap(await api.DELETE("/api/v1/people/{id}", { params: { path: { id } } }))}
              onDeleted={() => {
                invalidate(qc, ["people"]);
                void navigate({ to: "/dashboard/$", params: { _splat: "people" } });
              }}
            >
              <p>
                Only someone with no committee seats, posts or talks can be deleted; the API says what is in the way otherwise.
                To hide someone from the public site but keep their history, untick <strong>Listed</strong> instead.
              </p>
            </DeleteButton>
          </span>
        )}
      </SaveBar>
    </form>
  );
}
