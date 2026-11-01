import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, unwrap } from "@binarysust/api-client";
import { formatShort, formatTime } from "~/lib/format";
import { Empty, ErrorBox, Loading, SectionHead } from "../ui";
import { Button } from "@binarysust/ui/components/button";

export function RsvpsSection() {
  const qc = useQueryClient();
  const q = useQuery({ queryKey: ["me", "rsvps"], queryFn: async () => unwrap(await api.GET("/api/v1/me/rsvps")) });
  const cancel = useMutation({
    mutationFn: async (slug: string) => {
      const r = await api.DELETE("/api/v1/events/{slug}/rsvp", { params: { path: { slug } } });
      if (!r.response.ok) throw new Error("Could not cancel that RSVP.");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["me", "rsvps"] }),
  });

  return (
    <>
      <SectionHead title="My RSVPs" />
      {q.error ? (
        <ErrorBox error={q.error} retry={() => void q.refetch()} />
      ) : !q.data ? (
        <Loading what="Loading your RSVPs" />
      ) : q.data.length === 0 ? (
        <Empty>
          You have no RSVPs. <a href="/events" className="link">Find an event</a>.
        </Empty>
      ) : (
        <ul className="space-y-3">
          {q.data.map((r) => (
            <li key={r.eventSlug} className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-line bg-panel p-4">
              <div>
                <a href={`/events/${r.eventSlug}`} className="link font-semibold text-ink">
                  {r.eventTitle}
                </a>
                <p className="text-sm text-ink-2">
                  {formatShort(r.startsAt)}, {formatTime(r.startsAt)} ·{" "}
                  {r.status === "confirmed" ? "Seat confirmed" : `Waitlist #${r.position ?? "?"}`}
                  {r.ticket && <span className="ml-2 text-ink-3">Ticket {r.ticket.slice(0, 8)}</span>}
                </p>
              </div>
              <Button
                type="button"
                variant="outline" size="sm"
                disabled={cancel.isPending && cancel.variables === r.eventSlug}
                onClick={() => cancel.mutate(r.eventSlug)}
              >
                Cancel
              </Button>
            </li>
          ))}
        </ul>
      )}
      {cancel.error && <p role="alert" className="mt-4 font-semibold text-vermilion">{cancel.error.message}</p>}
    </>
  );
}
