// The one place for enum VALUES in the web app. The TYPES come from the API: Go string types
// with an Enum() method become OpenAPI enums (api/internal/apidoc), so `make gen` turns them into
// unions in schema.d.ts. These lists add display order (and the dashboard's labels), and
// `exhaustive` makes `tsc` fail when the API gains or loses a value, so they cannot drift.
import type { components } from "./schema";

type S = components["schemas"];

export type PersonStatus = S["people.Status"];
export type Affiliation = S["people.Affiliation"];
export type PositionCategory = S["people.Category"];
export type PositionGroup = S["people.Group"];
export type SocialKind = S["people.SocialKind"];
export type EventKind = S["events.Kind"];
export type EventStatus = S["events.Status"];
export type RsvpStatus = S["events.RSVPStatus"];
export type PostState = S["blog.State"];
export type PostAction = S["blog.Action"];
export type SponsorTier = S["sponsors.Tier"];
export type RoleName = S["access.Role"];
export type MediaPurpose = S["media.PurposeName"];

type Missing<T, A extends readonly unknown[]> = Exclude<T, A[number]>;
/** `exhaustive<T>()([...])` accepts the list only if it names every member of T. */
const exhaustive =
  <T extends string>() =>
  <const A extends readonly T[]>(values: A & ([Missing<T, A>] extends [never] ? unknown : { missing: Missing<T, A> })): readonly T[] =>
    values;

export const PERSON_STATUSES = exhaustive<PersonStatus>()(["committee", "advisor", "member", "alumni", "guest"]);
export const AFFILIATIONS = exhaustive<Affiliation>()(["student", "faculty", "guest"]);
export const POSITION_CATEGORIES = exhaustive<PositionCategory>()(["executive", "advisor"]);
export const POSITION_GROUPS = exhaustive<PositionGroup>()(["patron", "adviser", "leadership", "secretary", "assistant", "member"]);
export const SOCIAL_KINDS = exhaustive<SocialKind>()(["github", "linkedin", "x", "website", "codeforces", "leetcode", "other"]);
export const EVENT_KINDS = exhaustive<EventKind>()(["workshop", "contest", "hackathon", "seminar", "meetup"]);
export const EVENT_STATUSES = exhaustive<EventStatus>()(["draft", "published", "cancelled"]);
export const POST_STATES = exhaustive<PostState>()(["draft", "in_review", "published", "unpublished"]);
export const SPONSOR_TIERS = exhaustive<SponsorTier>()(["platinum", "gold", "silver", "partner"]);

export const GROUP_LABEL: Record<PositionGroup, string> = {
  patron: "Patron",
  adviser: "Adviser",
  leadership: "Leadership",
  secretary: "Secretary",
  assistant: "Assistant",
  member: "Member",
};

export const POST_STATE_LABEL: Record<PostState, string> = {
  draft: "Draft",
  in_review: "In review",
  published: "Published",
  unpublished: "Unpublished",
};

/** Narrows a <select> value to the list it was built from (the first value if it is not one). */
export function oneOf<T extends string>(values: readonly T[], v: string): T {
  return (values as readonly string[]).includes(v) ? (v as T) : (values[0] as T);
}
