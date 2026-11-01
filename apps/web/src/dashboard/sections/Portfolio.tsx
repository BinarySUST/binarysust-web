import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { Icon } from "~/components/Icon";
import { type Schemas, api, unwrap } from "~/lib/api/client";
import { SOCIAL_KINDS, oneOf } from "~/lib/api/enums";
import { ImageField, UploadButton } from "../MediaUpload";
import { ErrorBox, Field, Loading, SaveBar, SectionHead, errorFor, fieldErrors, invalidate, orNull, splitList } from "../ui";

type Portfolio = Schemas["people.Portfolio"];

const emptyProject = (): Portfolio["projects"][number] => ({ title: "", summary: "", url: null, repoUrl: null, imageUrl: null, tags: [] });
const emptyAchievement = (): Portfolio["achievements"][number] => ({ title: "", date: null, detail: "" });

export function PortfolioSection() {
  const q = useQuery({ queryKey: ["me", "portfolio"], queryFn: async () => unwrap(await api.GET("/api/v1/me/portfolio")) });
  if (q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (!q.data) return <Loading what="Loading your portfolio" />;
  return <PortfolioForm initial={q.data} />;
}

function PortfolioForm({ initial }: { initial: Portfolio }) {
  const qc = useQueryClient();
  const [form, setForm] = useState<Portfolio>(initial);
  const [skills, setSkills] = useState(initial.skills.join(", "));

  const save = useMutation({
    mutationFn: async (body: Portfolio) => unwrap(await api.PUT("/api/v1/me/portfolio", { body })),
    onSuccess: (data) => {
      qc.setQueryData(["me", "portfolio"], data);
      invalidate(qc, ["me"]);
    },
  });
  const fe = fieldErrors(save.error);

  const set = <K extends keyof Portfolio>(k: K, v: Portfolio[K]) => {
    setForm((f) => ({ ...f, [k]: v }));
    save.reset();
  };
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate({ ...form, skills: splitList(skills) });
  };

  return (
    <form onSubmit={onSubmit} className="max-w-3xl space-y-10" noValidate>
      <SectionHead title="My portfolio" />

      <fieldset className="space-y-5">
        <legend className="mb-2 text-lg font-semibold">Basics</legend>
        <ImageField
          label="Photo"
          purpose="avatar"
          value={form.avatarUrl ?? null}
          onChange={(v) => set("avatarUrl", v)}
          error={fe.avatarUrl}
          hint="A square photo works best. Resized to WebP before upload."
        />
        <Field label="Headline" hint="One line under your name, e.g. “Backend in Go, competitive programmer”." error={fe.headline}>
          <input className="field" value={form.headline} onChange={(e) => set("headline", e.target.value)} maxLength={120} required />
        </Field>
        <div className="grid gap-5 sm:grid-cols-2">
          <Field label="Job title" error={fe.jobTitle}>
            <input className="field" value={form.jobTitle} onChange={(e) => set("jobTitle", e.target.value)} />
          </Field>
          <Field label="Workplace" error={fe.workplace}>
            <input className="field" value={form.workplace} onChange={(e) => set("workplace", e.target.value)} />
          </Field>
        </div>
        <Field label="Bio" hint="Markdown. Rendered the same way as the blog." error={fe.bioMarkdown}>
          <textarea className="field min-h-40 font-[var(--font-code)] text-sm" value={form.bioMarkdown} onChange={(e) => set("bioMarkdown", e.target.value)} />
        </Field>
        <Field label="Skills" hint="Comma-separated." error={errorFor(fe, "skills", true)}>
          <input className="field" value={skills} onChange={(e) => (setSkills(e.target.value), save.reset())} />
        </Field>
        <Field label="CV link" error={fe.cvUrl}>
          <input className="field" type="url" value={form.cvUrl ?? ""} onChange={(e) => set("cvUrl", orNull(e.target.value))} placeholder="https://" />
        </Field>
        <UploadButton purpose="cv" label="Upload CV (PDF)" onUploaded={(o) => set("cvUrl", o.url)} />
      </fieldset>

      <fieldset className="space-y-3">
        <legend className="mb-2 text-lg font-semibold">Links</legend>
        {errorFor(fe, "socials", true) && <p className="text-sm font-semibold text-vermilion">{errorFor(fe, "socials", true)}</p>}
        {form.socials.map((s, i) => (
          <div key={i} className="grid grid-cols-[8.5rem_1fr_auto] gap-2">
            <select
              className="field"
              aria-label={`Link ${i + 1} kind`}
              value={s.kind}
              onChange={(e) => set("socials", form.socials.map((x, j) => (j === i ? { ...x, kind: oneOf(SOCIAL_KINDS, e.target.value) } : x)))}
            >
              {SOCIAL_KINDS.map((k) => (
                <option key={k} value={k}>
                  {k}
                </option>
              ))}
            </select>
            <input
              className="field"
              type="url"
              aria-label={`Link ${i + 1} URL`}
              value={s.url}
              onChange={(e) => set("socials", form.socials.map((x, j) => (j === i ? { ...x, url: e.target.value } : x)))}
            />
            <button type="button" className="btn btn-ghost !px-3" aria-label={`Remove link ${i + 1}`} onClick={() => set("socials", form.socials.filter((_, j) => j !== i))}>
              <Icon name="close" size={18} />
            </button>
          </div>
        ))}
        <button type="button" className="btn btn-ghost !min-h-10 !text-sm" onClick={() => set("socials", [...form.socials, { kind: "github", url: "" }])}>
          <Icon name="plus" size={16} /> Add link
        </button>
      </fieldset>

      <fieldset className="space-y-4">
        <legend className="mb-2 text-lg font-semibold">Projects</legend>
        {errorFor(fe, "projects", true) && <p className="text-sm font-semibold text-vermilion">{errorFor(fe, "projects", true)}</p>}
        {form.projects.map((p, i) => {
          const up = (patch: Partial<typeof p>) => set("projects", form.projects.map((x, j) => (j === i ? { ...x, ...patch } : x)));
          return (
            <div key={i} className="space-y-4 rounded-lg border border-line bg-panel p-4">
              <div className="flex items-center justify-between">
                <p className="font-semibold">Project {i + 1}</p>
                <button type="button" className="link text-sm" onClick={() => set("projects", form.projects.filter((_, j) => j !== i))}>
                  Remove
                </button>
              </div>
              <Field label="Title">
                <input className="field" value={p.title} onChange={(e) => up({ title: e.target.value })} />
              </Field>
              <Field label="Summary">
                <textarea className="field min-h-20" value={p.summary} onChange={(e) => up({ summary: e.target.value })} />
              </Field>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Live URL">
                  <input className="field" type="url" value={p.url ?? ""} onChange={(e) => up({ url: orNull(e.target.value) })} />
                </Field>
                <Field label="Source URL">
                  <input className="field" type="url" value={p.repoUrl ?? ""} onChange={(e) => up({ repoUrl: orNull(e.target.value) })} />
                </Field>
              </div>
              <Field label="Tags" hint="Comma-separated.">
                <input className="field" defaultValue={p.tags.join(", ")} onBlur={(e) => up({ tags: splitList(e.target.value) })} />
              </Field>
            </div>
          );
        })}
        <button type="button" className="btn btn-ghost !min-h-10 !text-sm" onClick={() => set("projects", [...form.projects, emptyProject()])}>
          <Icon name="plus" size={16} /> Add project
        </button>
      </fieldset>

      <fieldset className="space-y-4">
        <legend className="mb-2 text-lg font-semibold">Achievements</legend>
        {errorFor(fe, "achievements", true) && <p className="text-sm font-semibold text-vermilion">{errorFor(fe, "achievements", true)}</p>}
        {form.achievements.map((a, i) => {
          const up = (patch: Partial<typeof a>) => set("achievements", form.achievements.map((x, j) => (j === i ? { ...x, ...patch } : x)));
          return (
            <div key={i} className="grid gap-4 rounded-lg border border-line bg-panel p-4 sm:grid-cols-[1fr_11rem]">
              <Field label="Title">
                <input className="field" value={a.title} onChange={(e) => up({ title: e.target.value })} />
              </Field>
              <Field label="Date">
                <input className="field" type="date" value={a.date?.slice(0, 10) ?? ""} onChange={(e) => up({ date: orNull(e.target.value) })} />
              </Field>
              <div className="sm:col-span-2">
                <Field label="Detail">
                  <input className="field" value={a.detail} onChange={(e) => up({ detail: e.target.value })} />
                </Field>
              </div>
              <button type="button" className="link justify-self-start text-sm" onClick={() => set("achievements", form.achievements.filter((_, j) => j !== i))}>
                Remove
              </button>
            </div>
          );
        })}
        <button type="button" className="btn btn-ghost !min-h-10 !text-sm" onClick={() => set("achievements", [...form.achievements, emptyAchievement()])}>
          <Icon name="plus" size={16} /> Add achievement
        </button>
      </fieldset>

      <SaveBar
        sticky
        pending={save.isPending}
        error={save.error}
        success={save.isSuccess}
        label="Save portfolio"
        successText="Saved. Your public page updates in a few minutes."
        shown={["avatarUrl", "headline", "jobTitle", "workplace", "bioMarkdown", "skills", "cvUrl", "socials", "projects", "achievements"]}
      />
    </form>
  );
}
