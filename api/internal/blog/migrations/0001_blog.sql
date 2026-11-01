CREATE TABLE posts (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug               text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,99}$'),
  author_id          uuid NOT NULL REFERENCES persons (id),
  title              text NOT NULL CHECK (title <> ''),
  excerpt            text NOT NULL DEFAULT '',
  body_md            text NOT NULL DEFAULT '',
  cover_url          text,
  state              text NOT NULL DEFAULT 'draft'
                     CHECK (state IN ('draft', 'in_review', 'published', 'unpublished')),
  review_note        text NOT NULL DEFAULT '',
  first_published_at timestamptz,          -- set on the first publish and never cleared
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now(),
  CHECK (state NOT IN ('published', 'unpublished') OR first_published_at IS NOT NULL)
);
CREATE INDEX posts_published ON posts (first_published_at DESC) WHERE state = 'published';
CREATE INDEX posts_author ON posts (author_id);

CREATE TABLE post_tags (
  post_id uuid NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
  tag     text NOT NULL CHECK (tag ~ '^[a-z0-9][a-z0-9-]{0,29}$'),
  PRIMARY KEY (post_id, tag)
);
