import { useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { Icon } from "~/components/Icon";
import { SPONSOR_TIERS, oneOf } from "@binarysust/api-client";
import { ImageField } from "~/features/media/components/MediaUpload";
import type { SectionProps } from "~/dashboard/sections";
import {
  BackLink,
  DeleteButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  SaveBar,
  SectionHead,
  Table,
  fieldErrors,
  invalidate,
  td,
} from "~/dashboard/ui";
import { buttonVariants } from "@binarysust/ui/components/button";
import { Input } from "@binarysust/ui/components/input";
import { NativeSelect } from "@binarysust/ui/components/native-select";
import { type Sponsor, type SponsorIn, deleteSponsor, sponsorKeys, useSaveSponsor, useSponsor, useSponsors } from "../queries";

export function SponsorsSection({ rest }: SectionProps) {
  const [id] = rest;
  if (!id) return <SponsorList />;
  return <SponsorEditor id={id === "new" ? null : id} />;
}

function SponsorList() {
  const q = useSponsors();
  const year = new Date().getFullYear();
  return (
    <>
      <SectionHead title="Sponsors">
        <Link to="/dashboard/$" params={{ _splat: "sponsors/new" }} className={buttonVariants()}>
          <Icon name="plus" size={18} /> New sponsor
        </Link>
      </SectionHead>
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !q.data ? (
        <Loading what="Loading sponsors" />
      ) : q.data.length === 0 ? (
        <Empty>No sponsors yet.</Empty>
      ) : (
        <Table head={["Sponsor", "Tier", "Years", ""]}>
          {q.data.map((s) => (
            <tr key={s.id}>
              <td className={td}>
                <span className="flex items-center gap-3">
                  <img src={s.logoUrl} alt="" className="h-8 w-16 shrink-0 object-contain" />
                  <Link to="/dashboard/$" params={{ _splat: `sponsors/${s.id}` }} className="link font-semibold text-ink">
                    {s.name}
                  </Link>
                </span>
              </td>
              <td className={`${td} capitalize`}>{s.tier}</td>
              <td className={`${td} tabular`}>
                {s.since} to {s.until ?? "now"}
              </td>
              <td className={`${td} text-sm text-ink-3`}>{s.until !== null && s.until < year ? "Past" : "Active"}</td>
            </tr>
          ))}
        </Table>
      )}
    </>
  );
}

const blank = (): SponsorIn => ({ name: "", tier: "partner", logoUrl: "", url: "", since: new Date().getFullYear(), until: null });
const toIn = ({ id: _i, updatedAt: _u, ...rest }: Sponsor): SponsorIn => rest;

function SponsorEditor({ id }: { id: string | null }) {
  const q = useSponsor(id);
  if (id && q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (id && !q.data) return <Loading what="Loading sponsor" />;
  return <SponsorForm id={id} initial={q.data ? toIn(q.data) : blank()} key={id ?? "new"} />;
}

function SponsorForm({ id, initial }: { id: string | null; initial: SponsorIn }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [f, setF] = useState<SponsorIn>(initial);
  const save = useSaveSponsor(id, (s) => void navigate({ to: "/dashboard/$", params: { _splat: `sponsors/${s.id}` } }));
  const fe = fieldErrors(save.error);
  const set = <K extends keyof SponsorIn>(k: K, v: SponsorIn[K]) => (setF((x) => ({ ...x, [k]: v })), save.reset());
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate(f);
  };
  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-8" noValidate>
      <SectionHead title={id ? initial.name : "New sponsor"}>
        <BackLink to="sponsors">Back to sponsors</BackLink>
      </SectionHead>
      <div className="grid gap-5 sm:grid-cols-2">
        <Field label="Name" error={fe.name}>
          <Input value={f.name} onChange={(e) => set("name", e.target.value)} />
        </Field>
        <Field label="Tier" error={fe.tier}>
          <NativeSelect value={f.tier} onChange={(e) => set("tier", oneOf(SPONSOR_TIERS, e.target.value))}>
            {SPONSOR_TIERS.map((t) => (
              <option key={t} value={t}>
                {t[0]?.toUpperCase() + t.slice(1)}
              </option>
            ))}
          </NativeSelect>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Website" error={fe.url}>
            <Input type="url" value={f.url} onChange={(e) => set("url", e.target.value)} placeholder="https://" />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <ImageField
            label="Logo"
            purpose="logo"
            value={f.logoUrl || null}
            onChange={(v) => set("logoUrl", v ?? "")}
            error={fe.logoUrl}
            hint="A wide logo on a transparent background works best."
          />
        </div>
        <Field label="Since (year)" error={fe.since}>
          <Input type="number" inputMode="numeric" value={f.since} onChange={(e) => set("since", Number(e.target.value))} />
        </Field>
        <Field label="Until (year)" hint="Empty while the sponsorship is active." error={fe.until}>
          <Input
            type="number"
            inputMode="numeric"
            value={f.until ?? ""}
            onChange={(e) => set("until", e.target.value ? Number(e.target.value) : null)}
          />
        </Field>
      </div>
      <SaveBar
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        label={id ? "Save changes" : "Add sponsor"}
        shown={["name", "tier", "url", "logoUrl", "since", "until"]}
      >
        {id && (
          <span className="ml-auto">
            <DeleteButton
              what="sponsor"
              onDelete={() => deleteSponsor(id)}
              onDeleted={() => {
                invalidate(qc, sponsorKeys.all);
                void navigate({ to: "/dashboard/$", params: { _splat: "sponsors" } });
              }}
            >
              <p>
                {initial.name} disappears from the public site after the next build. To show a past sponsor instead, set{" "}
                <strong>Until</strong> to the last year.
              </p>
            </DeleteButton>
          </span>
        )}
      </SaveBar>
    </form>
  );
}
