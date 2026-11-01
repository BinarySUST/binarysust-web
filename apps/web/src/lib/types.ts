// The public snapshot contract, derived from the API's generated types (`pnpm gen`).
// The Go types in apps/api/internal/site are the source of truth (GET /api/v1/site/preview returns
// site.Snapshot). Go string types with an Enum() method arrive here as unions already
// (apps/api/internal/apidoc), so this file only names them; the one narrowing left is the public
// event status, which the snapshot restricts to published and cancelled.
// Enum value lists for code that iterates them (selects, filters) live in @binarysust/api-client (enums.ts).
import type { components } from "@binarysust/api-client";

type S = components["schemas"];
/** T with the keys of R replaced by R's types. */
type Narrow<T, R> = Omit<T, keyof R> & R;

export type PersonStatus = S["people.Status"];
export type SocialKind = S["people.SocialKind"];
export type EventKind = S["events.Kind"];
export type SponsorTier = S["sponsors.Tier"];
export type SeatCategory = S["people.Category"];
/** How the committee page groups seats: patron, adviser, leadership, secretary, assistant, member. */
export type SeatGroup = S["people.Group"];

export type Club = S["site.Club"];
export type Term = S["people.Term"];
export type Social = S["people.Social"];

/** `lifetimeMember`: held President, General Secretary or Director of Training in any term. */
export type PublicPerson = S["people.PublicPerson"];
export type Seat = S["people.Seat"];
export type Committee = S["people.Committee"];
export type PublicEvent = Narrow<S["events.PublicEvent"], { status: "published" | "cancelled" }>;
export type PublicPost = S["blog.PublicPost"];
export type PublicTalk = S["talks.Public"];
export type PublicSponsor = S["sponsors.Public"];

export type Snapshot = Narrow<S["site.Snapshot"], { events: PublicEvent[] }>;
