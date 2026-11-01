# Product

<!-- impeccable:product-schema 1 -->

> Written unattended from the build brief and `../docs/ARCHITECTURE.md`. Facts marked _(inferred)_ were not
> confirmed by a person; replace them when the club reviews this file.

## Platform

web

## Users

- Students at Shahjalal University of Science & Technology (SUST), Sylhet, deciding whether to join or show
  up to a Binary SUST event. Not only programmers: sessions range from workshops and contests to "Tech Hour",
  "Road to Tech Career", career talks and live sessions on marketing and brand scaling. _(inferred: mostly on
  phones, often on mobile data)_
- Members and alumni who keep a public portfolio and point recruiters at it.
- Committee members who run events, review blog posts and hand over roles every year, via the dashboard.
- Sponsors and guest speakers checking the club is real and active.

## Product Purpose

The public face and the operating tool of Binary SUST, the university's tech club: who is in it, what it
runs (workshops, contests, hackathons, seminars, meetups), what it writes and who backs it. Success: a
visitor finds the next event and RSVPs; a member's portfolio is worth linking; a new committee takes over
without anyone revoking access by hand.

## Positioning

Built and maintained by the club itself on $0 hosting: a static site generated from one snapshot of the
club's database, with every person, event and post a real record. Committee history is kept term by term,
so the site doubles as the club's institutional memory.

## Operating Context

- Yearly committee handover (term `2025-26` -> `2026-27`).
- Events with RSVP windows, capacity, waitlists and QR check-in.
- Blog posts move draft -> in review -> published.
- The API sleeps on Render's free tier; signed-in actions may wait up to a minute.

## Capabilities and Constraints

- Public pages are pre-rendered static HTML (Cloudflare Pages). Content never calls the API.
- Google-only sign-in. Dashboard sections are filtered by permissions from `GET /api/v1/auth/me`.
- Must stay fast on low-end Android phones and slow networks; responsive down to 360px.
- No external image hosts; avatars fall back to generated initials.

## Brand Commitments

- Name: **Binary SUST** (Bangla: বাইনারি). Logo: a "BINARY" wordmark with an antenna/circuit-tree letterform and the
  line "technology for real life", in green and red on white. The logo file is not in the repo yet; `<Logo>` sets
  the name in type until `BRAND.logoSrc` points at `public/brand/logo.svg`. Never trace or redraw the logo.
- Colours: brand green (≈ #2E9E4F) is the primary; brand red (≈ #E5252A) is used sparingly (a live dot, a rule),
  never for buttons or links.
- Mission (quote verbatim): "Aiming to boost digital skills, foster teamwork, and inspire creativity among
  members." Constitution objectives: computer and teamwork skills, presenting ideas, sharing technological
  information and analysis, creativity.
- Scale: about 6.5K followers on Facebook; the 2025-26 committee is the 10th executive committee. An executive
  committee can hold about 30 seats (officers, secretaries, assistant secretaries, executive members) and an
  advisory committee (Chief Patron: the SUST Vice-Chancellor, a Chief Adviser, advisers).
- Contact: binary.sust@gmail.com, facebook.com/binary.sust, linkedin.com/company/binary-sust. Production domain:
  binarysust.org.
- Register: **a university technology club, not a school club.** On 2026-11-02 the user rejected the first design
  (a painted rickshaw backboard: Bungee lettering, tassels, painted borders, binary-stud date panel) as "not whole
  fun type, it's a university website, not a school's website" and "silly, makes no sense". The site must read as
  professional, credible and calm, with a clear technical character: technical elements carry real meaning
  (data, code, metadata) or stay ambient background. No gimmick that needs a caption to explain itself.
- Light and dark are both first-class; the theme follows the system until the visitor picks one.

## Evidence on Hand

- Until the API exists, all content is `apps/web/.snapshot/fixture.json`: synthetic people, events, posts, talks
  and sponsors. Every name, company and number there is fake and must be replaced by the real snapshot.
- No real photos, logos, testimonials or member counts exist yet. Do not invent claims beyond the snapshot.

## Product Principles

1. The snapshot is the truth: every public number and name comes from it.
2. Show the club working (events, people, code), don't describe it.
3. The obvious change is the correct one: student maintainers copy the nearest file.
4. Nothing on the public path waits on the sleeping API.

## Accessibility & Inclusion

WCAG 2.2 AA contrast, full keyboard navigation with visible focus, `prefers-reduced-motion` respected,
usable at 360px wide. _(inferred standard from the brief)_
