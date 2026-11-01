-- Handles and emails are stored lowercase (CHECKs below), so plain text UNIQUE is
-- case-insensitive without the citext extension.

-- One row per human, for life. Status (committee, advisor, member, alumni, guest) is derived in
-- Go from affiliation, dates and current-term seats; it is never stored.
CREATE TABLE persons (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  handle       text   NOT NULL UNIQUE CHECK (handle ~ '^[a-z0-9][a-z0-9-]{1,39}$'),
  name         text   NOT NULL CHECK (name <> ''),
  email        text   UNIQUE CHECK (email = lower(email)), -- links an account on sign-in; never published
  affiliation  text   NOT NULL DEFAULT 'student' CHECK (affiliation IN ('student', 'faculty', 'guest')),
  listed       boolean NOT NULL DEFAULT true,        -- false hides the person from the public site
  avatar_url   text,
  headline     text   NOT NULL DEFAULT '',
  department   text   NOT NULL DEFAULT '',
  batch        int    CHECK (batch BETWEEN 1990 AND 2100), -- SUST session start year
  member_since date,
  graduated_on date,                                 -- alumni once this is <= today
  workplace    text   NOT NULL DEFAULT '',
  job_title    text   NOT NULL DEFAULT '',
  bio_md       text   NOT NULL DEFAULT '',
  skills       text[] NOT NULL DEFAULT '{}',
  socials      jsonb  NOT NULL DEFAULT '[]',         -- []people.Social; edited by the owner only
  cv_url       text,
  projects     jsonb  NOT NULL DEFAULT '[]',         -- []people.Project
  achievements jsonb  NOT NULL DEFAULT '[]',         -- []people.Achievement
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Old handles keep working: the site emits a redirect for each.
CREATE TABLE person_handles (
  handle    text PRIMARY KEY,
  person_id uuid NOT NULL REFERENCES persons (id) ON DELETE CASCADE
);

-- Which term is current lives in access.site_settings (the single pointer), not here.
CREATE TABLE terms (
  id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug      text NOT NULL UNIQUE CHECK (slug ~ '^[0-9]{4}-[0-9]{2}$'), -- '2025-26'
  label     text NOT NULL,                                            -- '2025–26'
  starts_on date NOT NULL
);

CREATE TABLE positions (
  id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug     text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,40}$'),
  title    text NOT NULL,
  category text NOT NULL CHECK (category IN ('executive', 'advisor')),
  rank     int  NOT NULL DEFAULT 100
);

CREATE TABLE committee_seats (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  term_id     uuid NOT NULL REFERENCES terms (id) ON DELETE CASCADE,
  position_id uuid NOT NULL REFERENCES positions (id),
  person_id   uuid NOT NULL REFERENCES persons (id) ON DELETE CASCADE,
  UNIQUE (term_id, position_id, person_id)
);
CREATE INDEX committee_seats_person ON committee_seats (person_id);
