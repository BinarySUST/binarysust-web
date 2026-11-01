// Browser-only: shrink a photo before upload, so a 12 MP phone picture becomes a few hundred KB.
// The API's per-purpose limits (1 to 5 MB) then rarely bite, and the media repo stays small.

export const MAX_SIDE = 1600;

/** Images the browser can decode become WebP no larger than MAX_SIDE on either side. Other files
 *  (PDFs, animated GIFs) pass through unchanged. Tries lower quality until it fits `maxBytes`. */
export async function prepareUpload(file: File, maxBytes?: number): Promise<{ blob: Blob; name: string }> {
  if (!file.type.startsWith("image/") || file.type === "image/gif" || file.type === "image/svg+xml") {
    return { blob: file, name: file.name };
  }
  const bitmap = await createImageBitmap(file);
  const scale = Math.min(1, MAX_SIDE / Math.max(bitmap.width, bitmap.height));
  const w = Math.max(1, Math.round(bitmap.width * scale));
  const h = Math.max(1, Math.round(bitmap.height * scale));
  const canvas = document.createElement("canvas");
  canvas.width = w;
  canvas.height = h;
  const ctx = canvas.getContext("2d");
  if (!ctx) return { blob: file, name: file.name };
  ctx.drawImage(bitmap, 0, 0, w, h);
  bitmap.close();

  const name = file.name.replace(/\.[^.]+$/, "") + ".webp";
  let blob: Blob | null = null;
  for (const q of [0.85, 0.72, 0.6, 0.45]) {
    blob = await new Promise<Blob | null>((res) => canvas.toBlob(res, "image/webp", q));
    // Browsers without a WebP encoder fall back to PNG; keep the original then.
    if (!blob || blob.type !== "image/webp") return { blob: file, name: file.name };
    if (!maxBytes || blob.size <= maxBytes) break;
  }
  return { blob: blob ?? file, name };
}
