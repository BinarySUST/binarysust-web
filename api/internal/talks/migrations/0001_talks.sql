-- Advisor and guest talks. The speaker is a person (advisor or guest), so their profile links.
CREATE TABLE talks (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,79}$'),
  title       text NOT NULL CHECK (title <> ''),
  speaker_id  uuid NOT NULL REFERENCES persons (id),
  given_at    timestamptz NOT NULL,
  summary_md  text NOT NULL DEFAULT '',
  video_url   text,
  slides_url  text,
  featured    boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
