-- Anyone who signs in with a verified Google email gets an account (RSVPs need one).
-- person_id is set when a person row has the same email.
CREATE TABLE accounts (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email         text   NOT NULL UNIQUE CHECK (email = lower(email)),
  name          text   NOT NULL DEFAULT '',
  avatar_url    text,
  person_id     uuid UNIQUE REFERENCES persons (id) ON DELETE SET NULL,
  created_at    timestamptz NOT NULL DEFAULT now(),
  last_login_at timestamptz NOT NULL DEFAULT now()
);

-- Which roles a committee position confers while its holder sits in the CURRENT term.
-- Only roles.manage edits this, so editing a roster cannot invent privileges.
CREATE TABLE position_roles (
  position_id uuid NOT NULL REFERENCES positions (id) ON DELETE CASCADE,
  role        text NOT NULL,
  PRIMARY KEY (position_id, role)
);

-- Explicit grants: term_id NULL is permanent (the technical admin who outlives handovers);
-- otherwise the grant only counts while that term is current.
CREATE TABLE role_grants (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  person_id  uuid NOT NULL REFERENCES persons (id) ON DELETE CASCADE,
  role       text NOT NULL,
  term_id    uuid REFERENCES terms (id) ON DELETE CASCADE,
  granted_by uuid REFERENCES accounts (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX role_grants_unique ON role_grants (person_id, role, coalesce(term_id, '00000000-0000-0000-0000-000000000000'));

-- Singleton. current_term_id is the ONLY pointer to the current term: handover is one write.
-- content_changed_at is bumped by the republish middleware after every content write.
CREATE TABLE site_settings (
  id                 boolean PRIMARY KEY DEFAULT true CHECK (id),
  current_term_id    uuid REFERENCES terms (id) ON DELETE SET NULL,
  content_changed_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO site_settings DEFAULT VALUES;
