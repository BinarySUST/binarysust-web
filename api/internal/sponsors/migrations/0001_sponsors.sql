CREATE TABLE sponsors (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name       text NOT NULL CHECK (name <> ''),
  tier       text NOT NULL CHECK (tier IN ('platinum', 'gold', 'silver', 'partner')),
  logo_url   text NOT NULL,
  url        text NOT NULL DEFAULT '',
  since      int  NOT NULL CHECK (since BETWEEN 2000 AND 2100),   -- first active year
  until      int  CHECK (until IS NULL OR until >= since),        -- last active year; NULL = ongoing
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
