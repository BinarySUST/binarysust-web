// Blog: your posts, the editor with a live preview, and for reviewers the review queue. The
// preview is rendered by POST /posts/preview, the same Go renderer the public site uses; it saves
// nothing and never republishes. Saving is explicit (the Save button, or before a transition).
//
// Workflow (api/internal/blog/workflow.go): draft -> in_review -> published <-> unpublished.
//   author:   submit (draft|unpublished), withdraw (in_review)
//   reviewer: request_changes (in_review, with a note), publish (in_review|unpublished), unpublish (published)
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useEffect, useState } from "react";
import { Icon } from "~/components/Icon";
import { type Me, type Schemas, api, unwrap } from "~/lib/api/client";
import { POST_STATE_LABEL, type PostAction, type PostState } from "~/lib/api/enums";
import { formatShort, formatTime } from "~/lib/format";
import { ImageField } from "../MediaUpload";
import { useMe } from "../me";
import type { SectionProps } from "../sections";
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
  errorFor,
  fieldErrors,
  invalidate,
  slugify,
  splitList,
  td,
} from "../ui";

type Post = Schemas["blog.Post"];
type PostIn = Schemas["blog.PostIn"];
type Action = PostAction;
const STATE_LABEL = POST_STATE_LABEL;

export function BlogSection({ rest }: SectionProps) {
  const [id] = rest;
  if (!id || id === "review" || id === "all") return <PostList view={(id as "review" | "all" | undefined) ?? "mine"} />;
  if (id === "new") return <NewPost />;
  return <PostEditor id={id} />;
}

const isReviewer = (me: Me) => me.permissions.includes("blog.review");

function PostList({ view }: { view: "mine" | "review" | "all" }) {
  const me = useMe();
  const reviewer = isReviewer(me);
  const q = useQuery({
    queryKey: ["posts", view],
    queryFn: async () =>
      unwrap(await api.GET("/api/v1/posts", { params: { query: view === "review" ? { state: "in_review" } : {} } })),
  });
  // Reviewers get every post from GET /posts; "My posts" keeps only their own.
  const rows = view === "mine" ? q.data?.filter((p) => p.author === me.account.personHandle) : q.data;
  const canWrite = !!me.account.personId;

  return (
    <>
      <SectionHead title="Blog">
        {canWrite && (
          <Link to="/dashboard/$" params={{ _splat: "blog/new" }} className="btn btn-primary">
            <Icon name="plus" size={18} /> New post
          </Link>
        )}
      </SectionHead>
      {reviewer && (
        <SubNav
          current={view === "mine" ? "blog" : `blog/${view}`}
          items={[
            { path: "blog", label: "My posts" },
            { path: "blog/review", label: "Review queue" },
            { path: "blog/all", label: "All posts" },
          ]}
        />
      )}
      {!canWrite && (
        <p className="mb-6 text-ink-2">
          Your account is not linked to a member profile yet, so you can't start posts. Ask a people manager to add you with
          this email address.
        </p>
      )}
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !rows ? (
        <Loading what="Loading posts" />
      ) : rows.length === 0 ? (
        <Empty>{view === "review" ? "Nothing is waiting for review." : view === "mine" ? "You have no posts yet. Start one." : "No posts yet."}</Empty>
      ) : (
        <Table head={view === "mine" ? ["Title", "State", "Updated"] : ["Title", "Author", "State", "Updated"]}>
          {rows.map((p) => (
            <tr key={p.id}>
              <td className={td}>
                <Link to="/dashboard/$" params={{ _splat: `blog/${p.id}` }} className="link font-semibold text-ink">
                  {p.title}
                </Link>
                <span className="block text-sm text-ink-3">/blog/{p.slug}</span>
              </td>
              {view !== "mine" && <td className={td}>{p.authorName}</td>}
              <td className={td}>
                <StateBadge state={p.state} />
              </td>
              <td className={`${td} tabular`}>{formatShort(p.updatedAt)}</td>
            </tr>
          ))}
        </Table>
      )}
    </>
  );
}

function StateBadge({ state }: { state: PostState }) {
  const tone = state === "published" ? "bg-parrot/15" : state === "in_review" ? "bg-yellow/20" : "bg-ground-2";
  return <span className={`inline-block rounded-full px-3 py-0.5 text-sm font-semibold ${tone}`}>{STATE_LABEL[state]}</span>;
}

const emptyPost: PostIn = { title: "", slug: "", excerpt: "", bodyMarkdown: "", coverUrl: null, tags: [] };

function NewPost() {
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const create = useMutation({
    mutationFn: async (t: string) => unwrap(await api.POST("/api/v1/posts", { body: { ...emptyPost, title: t, slug: slugify(t) } })),
    onSuccess: (p) => {
      qc.setQueryData(["posts", p.id], p);
      invalidate(qc, ["posts"]);
      void navigate({ to: "/dashboard/$", params: { _splat: `blog/${p.id}` } });
    },
  });
  const fe = fieldErrors(create.error);
  return (
    <form
      className="max-w-xl space-y-6"
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        create.mutate(title);
      }}
    >
      <SectionHead title="New post">
        <BackLink to="blog">Back to posts</BackLink>
      </SectionHead>
      <Field label="Title" error={fe.title} hint="You can change it later. The draft is private until an editor publishes it.">
        <input className="field" value={title} onChange={(e) => (setTitle(e.target.value), create.reset())} autoFocus />
      </Field>
      <SaveBar pending={create.isPending} error={create.error} success={false} label="Start draft" shown={["title"]} />
    </form>
  );
}

function PostEditor({ id }: { id: string }) {
  const q = useQuery({ queryKey: ["posts", id], queryFn: async () => unwrap(await api.GET("/api/v1/posts/{id}", { params: { path: { id } } })) });
  if (q.error) return <ErrorBox error={q.error} retry={() => void q.refetch()} />;
  if (!q.data) return <Loading what="Loading post" />;
  return <PostForm post={q.data} key={q.data.id} />;
}

const toIn = (p: Post): PostIn => ({ title: p.title, slug: p.slug, excerpt: p.excerpt, bodyMarkdown: p.bodyMarkdown, coverUrl: p.coverUrl, tags: p.tags });
const same = (a: PostIn, b: PostIn) => JSON.stringify(a) === JSON.stringify(b);

function PostForm({ post }: { post: Post }) {
  const me = useMe();
  const qc = useQueryClient();
  const navigate = useNavigate();
  const [f, setF] = useState<PostIn>(() => toIn(post));
  const [tags, setTags] = useState(post.tags.join(", "));
  const [saved, setSaved] = useState<PostIn>(() => toIn(post));
  const [dialog, setDialog] = useState<null | "request_changes" | "unpublish" | "delete">(null);
  const [note, setNote] = useState("");
  const body = (): PostIn => ({ ...f, tags: splitList(tags) });
  const dirty = !same(body(), saved);
  const editable = post.canEdit;

  // Live preview of the body as typed, debounced; rendering writes nothing.
  const [markdown, setMarkdown] = useState(f.bodyMarkdown);
  useEffect(() => {
    const t = setTimeout(() => setMarkdown(f.bodyMarkdown), 300);
    return () => clearTimeout(t);
  }, [f.bodyMarkdown]);
  const preview = useQuery({
    queryKey: ["post-preview", markdown],
    queryFn: async () => unwrap(await api.POST("/api/v1/posts/preview", { body: { markdown } })),
    placeholderData: keepPreviousData,
    staleTime: Infinity,
  });

  const save = useMutation({
    mutationFn: async (b: PostIn) => unwrap(await api.PUT("/api/v1/posts/{id}", { params: { path: { id: post.id } }, body: b })),
    onSuccess: (p, b) => {
      setSaved(b);
      qc.setQueryData(["posts", p.id], p);
      invalidate(qc, ["posts"]);
    },
  });
  const fe = fieldErrors(save.error);
  // Nothing autosaves, so warn before a reload or tab close drops unsaved edits.
  useEffect(() => {
    if (!dirty) return;
    const warn = (e: BeforeUnloadEvent) => e.preventDefault();
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);

  const move = useMutation({
    mutationFn: async ({ action, note }: { action: Action; note?: string }) => {
      if (editable && dirty) await save.mutateAsync(body());
      return unwrap(await api.POST("/api/v1/posts/{id}/transitions", { params: { path: { id: post.id } }, body: { action, note: note ?? "" } }));
    },
    onSuccess: (p) => {
      qc.setQueryData(["posts", p.id], p);
      invalidate(qc, ["posts"]);
      setDialog(null);
    },
  });
  const remove = useMutation({
    mutationFn: async () => unwrap(await api.DELETE("/api/v1/posts/{id}", { params: { path: { id: post.id } } })),
    onSuccess: () => {
      invalidate(qc, ["posts"]);
      void navigate({ to: "/dashboard/$", params: { _splat: "blog" } });
    },
  });

  const isAuthor = post.author === me.account.personHandle;
  const reviewer = isReviewer(me);
  const actions: { action: Action; label: string; primary?: boolean }[] = [];
  if (isAuthor && (post.state === "draft" || post.state === "unpublished")) actions.push({ action: "submit", label: "Submit for review", primary: true });
  if (isAuthor && post.state === "in_review") actions.push({ action: "withdraw", label: "Withdraw" });
  if (reviewer && (post.state === "in_review" || post.state === "unpublished")) actions.push({ action: "publish", label: "Approve & publish", primary: true });
  if (reviewer && post.state === "in_review") actions.push({ action: "request_changes", label: "Request changes" });
  if (reviewer && post.state === "published") actions.push({ action: "unpublish", label: "Unpublish" });
  const canDelete = reviewer || (isAuthor && post.state === "draft");

  const set = <K extends keyof PostIn>(k: K, v: PostIn[K]) => setF((x) => ({ ...x, [k]: v }));
  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    save.mutate(body());
  };

  const pv = preview.data;
  const previewTags = splitList(tags);
  return (
    <>
      <SectionHead title={post.title || "Untitled post"}>
        <BackLink to="blog">Back to posts</BackLink>
      </SectionHead>

      <div className="mb-6 flex flex-wrap items-center gap-3">
        <StateBadge state={post.state} />
        <span className="text-sm text-ink-3">
          by {post.authorName} · updated {formatShort(post.updatedAt)} {formatTime(post.updatedAt)}
        </span>
        <div className="ml-auto flex flex-wrap gap-2">
          {actions.map((a) => (
            <button
              key={a.action}
              type="button"
              className={`btn ${a.primary ? "btn-primary" : "btn-ghost"} !min-h-10 !text-sm`}
              disabled={move.isPending}
              onClick={() => (a.action === "request_changes" || a.action === "unpublish" ? setDialog(a.action) : move.mutate({ action: a.action }))}
            >
              {a.label}
            </button>
          ))}
          {canDelete && (
            <button type="button" className="btn btn-ghost !min-h-10 !text-sm" onClick={() => setDialog("delete")}>
              Delete
            </button>
          )}
        </div>
      </div>
      {move.error && !dialog && (
        <div className="mb-6">
          <ErrorBox error={move.error} />
        </div>
      )}
      {post.reviewNote && post.state === "draft" && (
        <div className="mb-6 rounded-lg border border-yellow/50 bg-yellow/10 p-4">
          <p className="font-semibold">Editor's note</p>
          <p className="mt-1 whitespace-pre-wrap text-ink-2">{post.reviewNote}</p>
        </div>
      )}
      {!editable && (
        <p className="mb-6 text-ink-2">
          This post is {STATE_LABEL[post.state].toLowerCase()}, so only an editor can change it now.
          {isAuthor && post.state === "in_review" && " Withdraw it to keep editing."}
        </p>
      )}

      <div className="grid gap-8 xl:grid-cols-2">
        <form onSubmit={onSubmit} className="min-w-0 space-y-5" noValidate>
          <fieldset disabled={!editable} className="space-y-5">
            <Field label="Title" error={fe.title}>
              <input className="field" value={f.title} onChange={(e) => set("title", e.target.value)} />
            </Field>
            <Field label="URL slug" hint={`/blog/${f.slug || "…"}`} error={fe.slug}>
              <input className="field" value={f.slug} onChange={(e) => set("slug", e.target.value)} />
            </Field>
            <Field label="Excerpt" hint="Shown on the blog list and in link previews." error={fe.excerpt}>
              <textarea className="field min-h-20" value={f.excerpt} onChange={(e) => set("excerpt", e.target.value)} maxLength={400} />
            </Field>
            <Field label="Tags" hint="Comma-separated, at most 8: go, icpc, web." error={errorFor(fe, "tags", true)}>
              <input className="field" value={tags} onChange={(e) => setTags(e.target.value)} />
            </Field>
            <ImageField label="Cover image" purpose="post" value={f.coverUrl} onChange={(v) => set("coverUrl", v)} error={fe.coverUrl} />
            <Field label="Body" hint="Markdown." error={fe.bodyMarkdown}>
              <textarea
                className="field min-h-[28rem] font-[var(--font-code)] text-sm"
                value={f.bodyMarkdown}
                onChange={(e) => set("bodyMarkdown", e.target.value)}
              />
            </Field>
          </fieldset>
          {editable && (
            <SaveBar
              pending={save.isPending}
              error={save.error}
              success={save.isSuccess && !dirty}
              label={post.state === "published" ? "Save (goes live)" : "Save"}
              successText={post.state === "published" ? "Saved. The public site republishes in a few minutes." : "Draft saved."}
              shown={["title", "slug", "excerpt", "tags", "coverUrl", "bodyMarkdown"]}
            >
              {dirty && <span className="text-sm text-ink-3">Unsaved changes</span>}
            </SaveBar>
          )}
        </form>

        <section aria-labelledby="preview-h" className="min-w-0">
          <div className="sticky top-6 rounded-xl border border-line bg-panel p-6">
            <p id="preview-h" className="mb-4 flex items-center justify-between text-sm font-semibold text-ink-3">
              Preview, exactly as published
              {(preview.isFetching || markdown !== f.bodyMarkdown) && <span role="status">Updating…</span>}
            </p>
            {preview.error ? (
              <ErrorBox error={preview.error} retry={() => void preview.refetch()} />
            ) : !pv ? (
              <Loading what="Rendering" />
            ) : (
              <article>
                <h2 className="sign text-2xl">{f.title || "Untitled post"}</h2>
                <p className="mt-2 text-sm text-ink-3">
                  {pv.readingMinutes} min read{previewTags.length > 0 && ` · ${previewTags.join(", ")}`}
                </p>
                {f.coverUrl && <img src={f.coverUrl} alt="" className="mt-4 w-full rounded-lg" />}
                {/* html is sanitized by the API's one renderer (goldmark + bluemonday). */}
                <div className="prose mt-6 max-h-[70vh] overflow-y-auto" dangerouslySetInnerHTML={{ __html: pv.html }} />
              </article>
            )}
          </div>
        </section>
      </div>

      <ConfirmDialog
        open={dialog === "request_changes"}
        title="Request changes"
        confirmLabel="Send back to author"
        pending={move.isPending}
        error={dialog === "request_changes" ? move.error : undefined}
        onClose={() => (setDialog(null), move.reset())}
        onConfirm={() => move.mutate({ action: "request_changes", note })}
      >
        <p>The post goes back to draft, and the author sees your note.</p>
        <Field label="What should change?" error={fieldErrors(move.error).note}>
          <textarea className="field min-h-28" value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </ConfirmDialog>
      <ConfirmDialog
        open={dialog === "unpublish"}
        title="Unpublish this post?"
        confirmLabel="Unpublish"
        pending={move.isPending}
        error={dialog === "unpublish" ? move.error : undefined}
        onClose={() => (setDialog(null), move.reset())}
        onConfirm={() => move.mutate({ action: "unpublish" })}
      >
        <p>It disappears from the public site on the next build. Its first-published date is kept if you publish it again.</p>
      </ConfirmDialog>
      <ConfirmDialog
        open={dialog === "delete"}
        title="Delete this post?"
        confirmLabel="Delete post"
        pending={remove.isPending}
        error={remove.error}
        onClose={() => (setDialog(null), remove.reset())}
        onConfirm={() => remove.mutate()}
      >
        <p>This can't be undone.</p>
      </ConfirmDialog>
    </>
  );
}
