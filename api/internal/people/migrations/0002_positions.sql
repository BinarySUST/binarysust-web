-- The club's constitution as reference data: every executive and advisory position, in rank
-- order, with the display group the public committee page sorts by. These rows exist in every
-- database (production included), so they live in a migration rather than in the demo seed.
-- Which roles each position carries is NOT here: that is access.position_roles, edited by
-- roles.manage in the dashboard (cmd/seed fills a default mapping for local data).
--
-- display_group: "group" is a reserved word in SQL; the API calls it `group`.
-- lifetime: holding this position in any term makes the person a lifetime member
-- (constitution: President, General Secretary and Director of Training of every committee).
-- Several people may hold the same position in a term (assistants, executive members):
-- committee_seats is UNIQUE (term_id, position_id, person_id), not (term_id, position_id).
ALTER TABLE positions
  ADD COLUMN display_group text NOT NULL DEFAULT 'member'
    CHECK (display_group IN ('patron', 'adviser', 'leadership', 'secretary', 'assistant', 'member')),
  ADD COLUMN lifetime boolean NOT NULL DEFAULT false;

UPDATE positions SET display_group = 'adviser' WHERE category = 'advisor';

INSERT INTO positions (slug, title, category, rank, display_group, lifetime) VALUES
  ('president',                            'President',                             'executive',  10, 'leadership', true),
  ('vice-president',                       'Vice President',                        'executive',  20, 'leadership', false),
  ('general-secretary',                    'General Secretary',                     'executive',  30, 'leadership', true),
  ('director-of-training',                 'Director of Training',                  'executive',  40, 'leadership', true),
  ('project-coordinator',                  'Project Co-ordinator',                  'executive',  50, 'leadership', false),
  ('assistant-general-secretary',          'Assistant General Secretary',           'executive',  60, 'assistant',  false),
  ('organizing-secretary',                 'Organizing Secretary',                  'executive',  70, 'secretary',  false),
  ('assistant-organizing-secretary',       'Assistant Organizing Secretary',        'executive',  80, 'assistant',  false),
  ('event-management-secretary',           'Event Management Secretary',            'executive',  90, 'secretary',  false),
  ('assistant-event-management-secretary', 'Assistant Event Management Secretary',  'executive', 100, 'assistant',  false),
  ('publication-secretary',                'Publication Secretary',                 'executive', 110, 'secretary',  false),
  ('assistant-publication-secretary',      'Assistant Publication Secretary',       'executive', 120, 'assistant',  false),
  ('it-design-secretary',                  'IT & Design Secretary',                 'executive', 130, 'secretary',  false),
  ('assistant-it-design-secretary',        'Assistant IT & Design Secretary',       'executive', 140, 'assistant',  false),
  ('finance-secretary',                    'Finance Secretary',                     'executive', 150, 'secretary',  false),
  ('office-secretary',                     'Office Secretary',                      'executive', 160, 'secretary',  false),
  ('assistant-office-secretary',           'Assistant Office Secretary',            'executive', 170, 'assistant',  false),
  ('internal-affairs-secretary',           'Secretary of Internal Affairs',         'executive', 180, 'secretary',  false),
  ('assistant-internal-affairs-secretary', 'Assistant Secretary of Internal Affairs', 'executive', 190, 'assistant', false),
  ('senior-executive-member',              'Senior Executive Member',               'executive', 200, 'member',     false),
  ('executive-member',                     'Executive Member',                      'executive', 210, 'member',     false),
  ('chief-patron',                         'Chief Patron',                          'advisor',    10, 'patron',     false),
  ('chief-adviser',                        'Chief Adviser',                         'advisor',    20, 'adviser',    false),
  ('adviser',                              'Adviser',                               'advisor',    30, 'adviser',    false)
ON CONFLICT (slug) DO UPDATE
  SET title = EXCLUDED.title, category = EXCLUDED.category, rank = EXCLUDED.rank,
      display_group = EXCLUDED.display_group, lifetime = EXCLUDED.lifetime;
