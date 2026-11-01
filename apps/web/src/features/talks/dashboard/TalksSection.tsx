import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { Icon } from "~/components/Icon";
import { type Schemas, api, unwrap } from "@binarysust/api-client";
import { formatShort } from "~/lib/format";
import { HandlePicker } from "~/features/people/components/PersonLookup";
import type { SectionProps } from "~/dashboard/sections";
import { BackLink, DeleteButton, Empty, ErrorBox, Field, Loading, SaveBar, SectionHead, Table, fieldErrors, invalidate, orNull, slugify, td } from "~/dashboard/ui";
import { buttonVariants } from "@binarysust/ui/components/button";
import { Input } from "@binarysust/ui/components/input";
import { Textarea } from "@binarysust/ui/components/textarea";
import { Checkbox } from "@binarysust/ui/components/checkbox";

type TalkIn = Schemas["talks.In"];
type Talk = Schemas["talks.Talk"];

export function TalksSection({ rest }: SectionProps) {
  const [id] = rest;
  if (!id) return <TalkList />;
  return <TalkEditor id={id === "new" ? null : id} />;
}

function TalkList() {
  const q = useQuery({ queryKey: ["talks"], queryFn: async () => unwrap(await api.GET("/api/v1/talks")) });
  return (
    <>
      <SectionHead title="Talks">
        <Link to="/dashboard/$" params={{ _splat: "talks/new" }} className={buttonVariants()}>
          <Icon name="plus" size={18} /> New talk
        </Link>
      </SectionHead>
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !q.data ? (
        <Loading what="Loading talks" />
      ) : q.data.length === 0 ? (
        <Empty>No talks yet.</Empty>
      ) : (
        <Table head={["Talk", "Speaker", "Date", "Featured"]}>
          {q.data.map((t) => (
            <tr key={t.id}>
              <td className={td}>
                <Link to="/dashboard/$" params={{ _splat: `talks/${t.id}` }} className="link font-semibold text-ink">
                  {t.title}
                </Link>
              </td>
              <td className={td}>
                {t.speakerName} <span className="text-ink-3">@{t.speaker}</span>
              </td>
              <td className={`${td} tabular`}>{formatShort(t.date)}</td>
              <td className={td}>{t.featured ? "Featured" : "—"}</td>
            </tr>
          ))}
        </Table>
      )}
    </>
  );
}

const blank: TalkIn = { slug: "", title: "", speaker: "", date: "", summaryMarkdown: "", videoUrl: null, slidesUrl: null, featured: false };
const toIn = ({ id: _i, speakerName: _n, updatedAt: _u, ...rest }: Talk): TalkIn => rest;

function TalkEditor({ id }: { id: string | null }) {
  const q = useQuery({
    queryKey: ["talks", id],
    queryFn: async () => unwrap(await api.GET("/api/v1/talks/{id}", { params: { path: { id: id ?? "" } } })),
    enabled: id !== null,
  });
  if (id && q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (id && !q.data) return <Loading what="Loading talk" />;
  return <TalkForm id={id} initial={q.data ? toIn(q.data) : blank} key={id ?? "new"} />;
}

function TalkForm({ id, initial }: { id: string | null; initial: TalkIn }) {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [f, setF] = useState<TalkIn>(initial);
  const save = useMutation({
    mutationFn: async (body: TalkIn) =>
      unwrap(id ? await api.PUT("/api/v1/talks/{id}", { params: { path: { id } }, body }) : await api.POST("/api/v1/talks", { body })),
    onSuccess: (t) => {
      qc.setQueryData(["talks", t.id], t);
      invalidate(qc, ["talks"]);
      if (!id) void navigate({ to: "/dashboard/$", params: { _splat: `talks/${t.id}` } });
    },
  });
  const fe = fieldErrors(save.error);
  const set = <K extends keyof TalkIn>(k: K, v: TalkIn[K]) => (setF((x) => ({ ...x, [k]: v })), save.reset());
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate(f);
  };

  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-8" noValidate>
      <SectionHead title={id ? "Edit talk" : "New talk"}>
        <BackLink to="talks">Back to talks</BackLink>
      </SectionHead>
      <div className="grid gap-5 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Title" error={fe.title}>
            <Input
              
              value={f.title}
              onChange={(e) => {
                const title = e.target.value;
                setF((x) => ({ ...x, title, slug: id || x.slug !== slugify(x.title) ? x.slug : slugify(title) }));
                save.reset();
              }}
 />
          </Field>
        </div>
        <Field label="URL slug" error={fe.slug}>
          <Input value={f.slug} onChange={(e) => set("slug", e.target.value)} />
        </Field>
        <HandlePicker label="Speaker" hint="Add them under People first if they are new." error={fe.speaker} value={f.speaker} onChange={(v) => set("speaker", v)} />
        <Field label="Date" error={fe.date}>
          <Input type="date" value={f.date.slice(0, 10)} onChange={(e) => set("date", e.target.value ? `${e.target.value}T00:00:00Z` : "")} />
        </Field>
        <label className="flex items-center gap-3 sm:mt-7">
          <Checkbox className="size-5" checked={f.featured} onCheckedChange={(checked) => set("featured", checked)} />
          Feature on the home page
        </label>
        <Field label="Video link" error={fe.videoUrl}>
          <Input type="url" value={f.videoUrl ?? ""} onChange={(e) => set("videoUrl", orNull(e.target.value))} />
        </Field>
        <Field label="Slides link" error={fe.slidesUrl}>
          <Input type="url" value={f.slidesUrl ?? ""} onChange={(e) => set("slidesUrl", orNull(e.target.value))} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Summary" hint="Markdown." error={fe.summaryMarkdown}>
            <Textarea className="min-h-40 text-sm" value={f.summaryMarkdown} onChange={(e) => set("summaryMarkdown", e.target.value)} />
          </Field>
        </div>
      </div>
      <SaveBar
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        label={id ? "Save changes" : "Create talk"}
        shown={["title", "slug", "speaker", "date", "videoUrl", "slidesUrl", "summaryMarkdown"]}
      >
        {id && (
          <span className="ml-auto">
            <DeleteButton
              what="talk"
              onDelete={async () => unwrap(await api.DELETE("/api/v1/talks/{id}", { params: { path: { id } } }))}
              onDeleted={() => {
                invalidate(qc, ["talks"]);
                void navigate({ to: "/dashboard/$", params: { _splat: "talks" } });
              }}
            >
              <p>The talk disappears from the public site after the next build. The speaker's profile stays. This can't be undone.</p>
            </DeleteButton>
          </span>
        )}
      </SaveBar>
    </form>
  );
}
