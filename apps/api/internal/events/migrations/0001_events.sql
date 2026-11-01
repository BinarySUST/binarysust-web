CREATE TABLE events (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug          text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,79}$'),
  title         text NOT NULL CHECK (title <> ''),
  kind          text NOT NULL CHECK (kind IN ('workshop', 'contest', 'hackathon', 'seminar', 'meetup')),
  status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'cancelled')),
  summary       text NOT NULL DEFAULT '',
  body_md       text NOT NULL DEFAULT '',
  cover_url     text,
  starts_at     timestamptz NOT NULL,
  ends_at       timestamptz NOT NULL,
  venue         text,
  online_url    text,
  reg_opens_at  timestamptz,
  reg_closes_at timestamptz,
  capacity      int CHECK (capacity > 0),            -- NULL with a window: unlimited seats
  recap_md      text NOT NULL DEFAULT '',
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_at > starts_at),
  -- Registration mirrors the Go Registration struct: the window is all or nothing, and a
  -- capacity means nothing without a window.
  CHECK ((reg_opens_at IS NULL) = (reg_closes_at IS NULL)),
  CHECK (capacity IS NULL OR reg_opens_at IS NOT NULL),
  CHECK (reg_closes_at IS NULL OR reg_closes_at > reg_opens_at)
);
CREATE INDEX events_starts ON events (starts_at DESC);

CREATE TABLE event_speakers (
  event_id  uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
  person_id uuid NOT NULL REFERENCES persons (id) ON DELETE CASCADE,
  sort      int  NOT NULL DEFAULT 0,
  PRIMARY KEY (event_id, person_id)
);

CREATE TABLE event_photos (
  id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
  url      text NOT NULL,
  caption  text NOT NULL DEFAULT '',
  sort     int  NOT NULL DEFAULT 0
);
CREATE INDEX event_photos_event ON event_photos (event_id, sort);

CREATE TABLE rsvps (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
  account_id    uuid NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
  status        text NOT NULL CHECK (status IN ('confirmed', 'waitlisted', 'cancelled')),
  ticket        text NOT NULL UNIQUE,                -- random; encoded in the QR code
  checked_in_at timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),  -- waitlist order; reset on re-registering
  updated_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (event_id, account_id)                      -- RSVP is idempotent
);
CREATE INDEX rsvps_waitlist ON rsvps (event_id, created_at) WHERE status = 'waitlisted';
CREATE INDEX rsvps_account ON rsvps (account_id);
