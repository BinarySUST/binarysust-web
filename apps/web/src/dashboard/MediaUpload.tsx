// The one uploader: Media section, avatar, event cover and photos, post cover, sponsor logo.
// Images are resized to WebP (at most 1600 px) in the browser before POST /api/v1/media.
import { useMutation } from "@tanstack/react-query";
import { useId, useRef } from "react";
import { Icon } from "~/components/Icon";
import { MEDIA_PURPOSES, type MediaObject, type MediaPurpose, uploadMedia } from "@binarysust/api-client";
import { prepareUpload } from "~/lib/image";

/** api/internal/media Purposes MaxBytes, so the resizer knows when to drop quality. */
const MAX_BYTES: Record<MediaPurpose, number> = {
  avatar: 1 << 20,
  cv: 4 << 20,
  post: 2 << 20,
  event: 3 << 20,
  talk: 2 << 20,
  logo: 1 << 20,
  file: 5 << 20,
};

export function useUpload(purpose: MediaPurpose, onDone?: (o: MediaObject) => void) {
  return useMutation({
    mutationFn: async (file: File) => {
      const { blob, name } = await prepareUpload(file, MAX_BYTES[purpose]);
      return uploadMedia(blob, purpose, name);
    },
    onSuccess: (o) => onDone?.(o),
  });
}

/** A file button. Resizes, uploads, and reports the stored object. */
export function UploadButton({
  purpose,
  onUploaded,
  label = "Upload",
}: {
  purpose: MediaPurpose;
  onUploaded: (o: MediaObject) => void;
  label?: string;
}) {
  const input = useRef<HTMLInputElement>(null);
  const id = useId();
  const up = useUpload(purpose, onUploaded);
  return (
    <span className="inline-flex flex-wrap items-center gap-3">
      <input
        ref={input}
        id={id}
        type="file"
        accept={MEDIA_PURPOSES[purpose].accept}
        className="sr-only"
        tabIndex={-1}
        aria-label={`${label}: choose a file`}
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) up.mutate(f);
          e.target.value = "";
        }}
      />
      <button type="button" className="btn btn-ghost !min-h-10 !text-sm" disabled={up.isPending} onClick={() => input.current?.click()}>
        <Icon name="image" size={16} /> {up.isPending ? "Uploading…" : label}
      </button>
      <span aria-live="polite" className="text-sm">
        {up.error && <span className="font-semibold text-vermilion">{up.error.message}</span>}
      </span>
    </span>
  );
}

/** A URL field with an Upload button and a thumbnail: for avatarUrl, coverUrl, logoUrl. */
export function ImageField({
  label,
  purpose,
  value,
  onChange,
  error,
  hint,
}: {
  label: string;
  purpose: MediaPurpose;
  value: string | null;
  onChange: (url: string | null) => void;
  error?: string;
  hint?: string;
}) {
  const id = useId();
  return (
    <div>
      <label htmlFor={id} className="mb-1.5 block text-sm font-semibold text-ink-2">
        {label}
      </label>
      <div className="flex flex-wrap items-start gap-4">
        {value ? (
          <img src={value} alt="" className="h-20 w-20 rounded-lg border border-line bg-panel object-cover" />
        ) : (
          <span className="grid h-20 w-20 place-items-center rounded-lg border border-dashed border-line text-ink-3" aria-hidden="true">
            <Icon name="image" size={22} />
          </span>
        )}
        <div className="min-w-0 flex-1 space-y-2">
          <input
            id={id}
            className="field"
            type="url"
            aria-describedby={error || hint ? `${id}-note` : undefined}
            aria-invalid={error ? true : undefined}
            placeholder="https://"
            value={value ?? ""}
            onChange={(e) => onChange(e.target.value.trim() ? e.target.value.trim() : null)}
          />
          <div className="flex flex-wrap items-center gap-3">
            <UploadButton purpose={purpose} onUploaded={(o) => onChange(o.url)} />
            {value && (
              <button type="button" className="link text-sm" onClick={() => onChange(null)}>
                Remove
              </button>
            )}
          </div>
        </div>
      </div>
      {error ? (
        <span id={`${id}-note`} className="mt-1 block text-sm font-semibold text-vermilion">
          {error}
        </span>
      ) : (
        hint && (
          <span id={`${id}-note`} className="mt-1 block text-sm text-ink-3">
            {hint}
          </span>
        )
      )}
    </div>
  );
}
