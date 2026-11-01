// The one live island on public pages. The page around it is static; seats and your RSVP come from the API.
import { type ReactNode, useEffect, useState } from "react";
import { type Schemas, api, hasSessionHint, loginUrl, unwrap, useWaking, warmApi } from "~/lib/api/client";
import { formatShort } from "~/lib/format";
import { useClientNow, useClientValue } from "~/lib/useNow";
import type { PublicEvent } from "~/lib/types";
import { Icon } from "./Icon";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";

type Props = Pick<PublicEvent, "slug" | "registration" | "endsAt" | "status"> & { compact?: boolean };

type Window = "none" | "soon" | "open" | "closed" | "over" | "cancelled";

function windowOf({ registration, endsAt, status }: Props, now: number): Window {
  if (status === "cancelled") return "cancelled";
  if (Date.parse(endsAt) < now) return "over";
  if (!registration) return "none";
  if (now < Date.parse(registration.opensAt)) return "soon";
  if (now > Date.parse(registration.closesAt)) return "closed";
  return "open";
}

export function Rsvp(props: Props) {
  const { slug, registration, compact } = props;
  // Pre-render has no "now" worth trusting; judge the window once mounted, from the static data.
  const now = useClientNow();
  const signedIn = useClientValue(hasSessionHint, false);
  const [avail, setAvail] = useState<Schemas["events.Availability"] | null>(null);
  const [mine, setMine] = useState<Schemas["events.MyRSVP"] | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const waking = useWaking();

  useEffect(() => {
    warmApi();
    if (!hasSessionHint()) return;
    let live = true;
    api
      .GET("/api/v1/events/{slug}/availability", { params: { path: { slug } } })
      .then(unwrap)
      .then((a) => {
        if (!live) return;
        setAvail(a);
        setMine(a.mine);
      })
      .catch(() => {
        /* seats are a nicety; the button still works */
      });
    return () => {
      live = false;
    };
  }, [slug]);

  const w: Window = now === null ? (registration ? "open" : "none") : windowOf(props, now);

  async function reserve() {
    if (!signedIn) {
      window.location.assign(loginUrl(`/events/${slug}`));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const r = await api.POST("/api/v1/events/{slug}/rsvp", { params: { path: { slug } } });
      if (r.response.status === 401) {
        window.location.assign(loginUrl(`/events/${slug}`));
        return;
      }
      setMine(unwrap(r));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not reserve a seat. Try again in a moment.");
    } finally {
      setBusy(false);
    }
  }

  async function cancel() {
    setBusy(true);
    setError(null);
    try {
      const r = await api.DELETE("/api/v1/events/{slug}/rsvp", { params: { path: { slug } } });
      if (!r.response.ok) throw new Error("Could not cancel. Try again in a moment.");
      setMine(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not cancel.");
    } finally {
      setBusy(false);
    }
  }

  const seats =
    avail && avail.capacity !== null ? `${Math.max(0, avail.capacity - avail.confirmed)} of ${avail.capacity} seats left` : null;

  let body: ReactNode;
  if (w === "cancelled") body = <p className="font-semibold text-danger">This event was cancelled.</p>;
  else if (w === "over") body = <p className="text-ink-2">This event has ended.</p>;
  else if (w === "none") body = <p className="text-ink-2">No registration needed. Just turn up.</p>;
  else if (w === "soon" && registration)
    body = <p className="text-ink-2">Registration opens {formatShort(registration.opensAt)}.</p>;
  else if (w === "closed") body = <p className="text-ink-2">Registration has closed.</p>;
  else if (mine)
    body = (
      <div className="flex flex-wrap items-center gap-3">
        <Badge variant="default">
          <Icon name="check" size={14} />
          {mine.status === "confirmed" ? "Seat confirmed" : `Waitlist #${mine.position ?? "?"}`}
        </Badge>
        <Button variant="outline" size="sm" onClick={cancel} disabled={busy}>
          {busy ? "Cancelling…" : "Cancel my RSVP"}
        </Button>
      </div>
    );
  else
    body = (
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Button onClick={reserve} disabled={busy}>
          <Icon name="ticket" size={18} />
          {busy ? "Reserving…" : signedIn ? "Reserve a seat" : "Sign in to reserve"}
        </Button>
        <span className="meta text-ink-2">
          {seats ?? (registration && `Closes ${formatShort(registration.closesAt)}`)}
        </span>
      </div>
    );

  return (
    <div className={compact ? "" : "space-y-2"} aria-live="polite">
      {body}
      {busy && waking && (
        <p className="text-sm text-ink-2">Waking the server (up to a minute)…</p>
      )}
      {error && (
        <p role="alert" className="flex items-center gap-2 text-sm font-medium text-danger">
          <Icon name="alert" size={16} /> {error}
        </p>
      )}
    </div>
  );
}
