// Media: upload a file and get its URL. Uploads go to the club's media repository (local dev:
// .data/media behind Caddy), so photos never slow down the main site's builds.
import { useState } from "react";
import { Icon } from "~/components/Icon";
import { MEDIA_PURPOSES, type MediaObject, type MediaPurpose } from "@binarysust/api-client";
import { MAX_SIDE } from "~/lib/image";
import { UploadButton } from "../MediaUpload";
import { useMe } from "../me";
import { Field, SectionHead } from "../ui";
import { Button } from "@binarysust/ui/components/button";
import { NativeSelect } from "@binarysust/ui/components/native-select";

export function MediaSection() {
  const me = useMe();
  const purposes = (Object.keys(MEDIA_PURPOSES) as MediaPurpose[]).filter((p) => me.permissions.includes(MEDIA_PURPOSES[p].requires));
  const [purpose, setPurpose] = useState<MediaPurpose>(purposes.includes("file") ? "file" : (purposes[0] ?? "file"));
  const [uploads, setUploads] = useState<MediaObject[]>([]);
  const [copied, setCopied] = useState<string | null>(null);

  async function copy(url: string) {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(url);
    } catch {
      setCopied(null);
    }
  }

  return (
    <>
      <SectionHead title="Media" />
      <p className="mb-8 max-w-2xl text-ink-2">
        Upload a file and paste its URL anywhere that takes a link. Photos are resized in your browser to WebP, at most{" "}
        {MAX_SIDE} px on the long side, before they are sent.
      </p>
      <div className="flex max-w-2xl flex-wrap items-end gap-4 rounded-xl border border-line bg-panel p-5">
        <div className="w-64">
          <Field label="What is it for?" hint="Each kind has its own size and type limits.">
            <NativeSelect value={purpose} onChange={(e) => setPurpose(e.target.value as MediaPurpose)}>
              {purposes.map((p) => (
                <option key={p} value={p}>
                  {MEDIA_PURPOSES[p].label}
                </option>
              ))}
            </NativeSelect>
          </Field>
        </div>
        <div className="pb-7">
          <UploadButton purpose={purpose} label="Choose file" onUploaded={(o) => setUploads((u) => [o, ...u])} />
        </div>
      </div>

      {uploads.length > 0 && (
        <section aria-labelledby="uploads-h" className="mt-10">
          <h2 id="uploads-h" className="mb-4 text-lg font-semibold">
            Uploaded this session
          </h2>
          <ul className="space-y-3">
            {uploads.map((o) => (
              <li key={o.key} className="flex flex-wrap items-center gap-4 rounded-lg border border-line bg-panel p-3">
                {o.contentType.startsWith("image/") ? (
                  <img src={o.url} alt="" className="h-16 w-24 rounded object-cover" />
                ) : (
                  <span className="grid h-16 w-24 place-items-center rounded bg-ground-2 text-ink-3">
                    <Icon name="file" />
                  </span>
                )}
                <div className="min-w-0 flex-1">
                  <a href={o.url} target="_blank" rel="noreferrer" className="link block truncate font-[var(--font-code)] text-sm">
                    {o.url}
                  </a>
                  <p className="text-sm text-ink-3">
                    {o.contentType} · {(o.size / 1024).toFixed(0)} KB
                  </p>
                </div>
                <Button type="button" variant="outline" size="sm" onClick={() => void copy(o.url)}>
                  {copied === o.url ? "Copied" : "Copy URL"}
                </Button>
              </li>
            ))}
          </ul>
        </section>
      )}
    </>
  );
}
