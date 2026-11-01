---
name: Binary SUST
description: The student technology club of SUST, set as an engineering drawing sheet.
colors:
  primary: "oklch(0.5 0.13 149)"
  primary-hover: "oklch(0.44 0.12 149)"
  primary-ink: "oklch(0.45 0.12 149)"
  primary-soft: "oklch(0.95 0.035 149)"
  primary-foreground: "oklch(1 0 0)"
  brand: "#2e9e4f"
  brand-red: "#e5252a"
  danger: "oklch(0.52 0.19 27)"
  warn: "oklch(0.72 0.15 70)"
  ground: "oklch(0.985 0.003 149)"
  ground-2: "oklch(0.962 0.005 149)"
  panel: "oklch(1 0 0)"
  line: "oklch(0.885 0.007 149)"
  line-strong: "oklch(0.76 0.01 149)"
  ink: "oklch(0.21 0.014 155)"
  ink-2: "oklch(0.39 0.014 155)"
  ink-3: "oklch(0.49 0.013 155)"
  code-bg: "oklch(0.965 0.005 149)"
typography:
  display: { fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif", fontSize: "clamp(3rem, 9vw, 6rem)", fontWeight: 620, lineHeight: 0.96, letterSpacing: "-0.022em", fontVariation: "'wdth' 112" }
  headline: { fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif", fontSize: "clamp(2.25rem, 5.5vw, 3.75rem)", fontWeight: 620, lineHeight: 1.06, letterSpacing: "-0.022em", fontVariation: "'wdth' 112" }
  title: { fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif", fontSize: "clamp(1.5rem, 3vw, 2rem)", fontWeight: 620, lineHeight: 1.06, letterSpacing: "-0.022em", fontVariation: "'wdth' 112" }
  body: { fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif", fontSize: "1.0625rem", fontWeight: 400, lineHeight: 1.6 }
  label: { fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, monospace", fontSize: "0.8125rem", fontWeight: 400, lineHeight: 1.45, letterSpacing: "0", fontFeature: "'tnum'" }
  bangla: { fontFamily: "Anek Bangla Variable, Noto Sans Bengali, sans-serif", fontSize: "1.625rem", fontWeight: 500, lineHeight: 1 }
rounded: { xs: "2px", sm: "3px", md: "4px", lg: "6px" }
spacing: { gutter: "16px", gutter-sm: "24px", section: "96px", head: "32px" }
components:
  button-primary: { backgroundColor: "{colors.primary}", textColor: "{colors.primary-foreground}", rounded: "{rounded.md}", padding: "8px 18px", height: "44px" }
  button-primary-hover: { backgroundColor: "{colors.primary-hover}" }
  button-ghost: { backgroundColor: "transparent", textColor: "{colors.ink}", rounded: "{rounded.md}", padding: "8px 18px", height: "44px" }
  button-ghost-hover: { backgroundColor: "{colors.ground-2}" }
  tag: { textColor: "{colors.ink-2}", typography: "{typography.label}", rounded: "{rounded.sm}", padding: "2px 8px", height: "28px" }
  field: { backgroundColor: "{colors.panel}", textColor: "{colors.ink}", rounded: "{rounded.md}", padding: "8px 12px", height: "44px" }
  window: { backgroundColor: "{colors.panel}", rounded: "{rounded.lg}" }
---

# Design System: Binary SUST

**Why.** On 2026-11-02 the user rejected the previous rickshaw-backboard world: "not whole fun type, it's a university website, not a school's website". Technical elements must carry real meaning (data, code, metadata) or sit in the background as ambience; never a captioned gimmick.

## Overview

**Creative North Star: "The Engineering Drawing Sheet"**

A well-made engineering document crossed with a developer tool: hairline frames, title blocks for facts, precise mono metadata, real club data shown the way engineers show it (a contribution graph, a code window, a dated schedule). Calm, credible, university-grade. Ornament lives only in the hero background (a faint grid and a pointer-following glow); the text column carries none.

**Key Characteristics:**
- Light and dark are both first-class; light is the default, the system preference decides until the visitor picks.
- One accent (brand green) for action and "active"; everything else is tinted neutral.
- One 1px hairline weight for every rule, border and frame; depth by flat plane steps only.
- Archivo semi-expanded for display, JetBrains Mono for anything a machine would print.

## Colors

Neutrals tinted toward the brand green hue (149); one green accent; brand red as a single spark. Frontmatter holds the light values; dark ("night graphite", same hue) is in `.impeccable/design.json`. Theme mechanics: `data-theme` on `<html>` overrides; the dark set is duplicated under `prefers-color-scheme: dark` on `:root:not([data-theme])` so no-JS visitors still get it. Any token change must be made in all three blocks.

### Primary
- **Drafting Green** (`--primary`): buttons, the prompt `$`, section indices, graph peak, focus. Darkened from the logo green until white text on it passes AA. **Primary Ink** is its text-strength variant for links and hover; **Primary Soft** is selection.
- **Logo Green** (brand): the logo colour itself, graphics only (3:1 on ground), never text.

### Tertiary
- **Live Red** (brand-red): only the pulsing live dot beside an upcoming event.

### Neutral
- **Drafting Film** (ground), **Sunken Plane** (ground-2: footer, advisor band, hover fills), **Sheet White** (panel): the three planes.
- **Hairline** (line) and **Strong Hairline** (line-strong: frames, section rules, tag and field borders).
- **Ink / Ink 2 / Ink 3**: text, secondary text, metadata labels. All verified ≥4.5:1 on ground and panel in both themes.
- **Status**: danger (brand red at text strength), warn (amber), ok (= primary).

**The One Spark Rule.** Brand red appears once: the live dot. It is never a fill, rule, heading or button.

**The Legacy Alias Rule.** The dashboard still uses old names; keep them resolving, never use them on new public surfaces: `--vermilion` / `text-vermilion` = danger, `--yellow` / `yellow` = warn, `--parrot` / `parrot` = ok, `--font-code` = mono.

## Typography

**Display Font:** Archivo Variable (wdth axis), `font-stretch: 112%`, weight 620, tracking −0.022em (`.display`, legacy alias `.sign`).
**Body Font:** Archivo Variable at normal width, 17px, line-height 1.6; prose capped at 68ch.
**Label/Mono Font:** JetBrains Mono Variable (`.meta`, 13px, tabular): dates, handles, counts, labels, code.
**Bangla:** Anek Bangla only for the wordmark বাইনারি; never body or headings.

Hierarchy: hero name (display), page titles (headline), section heads (title), event and person titles in Archivo 600 at 19-22px, body, mono labels. Numbers are tabular everywhere they line up.

**The Inline Index Rule.** No eyebrows or kickers above headings. A section's mono index ("01") sits on the heading's own baseline, in primary ink, with a strong hairline running beneath to the optional action.

## Layout

Container 80rem (`max-w-7xl`) with 16px / 24px (sm+) gutters; holds at 360px. Sections open 96px apart; section heads take 32px below. The hero is a two-column grid (text + 25rem next-event window) collapsing to one column below `lg`. The header is sticky, 64px compacting to 52px on scroll with its hairline strengthening; `html` carries `scroll-padding-top: 5rem` so anchors and focus are never covered. Lists (schedule, committee, sponsors, stats) are hairline-ruled grids, not card rows.

## Elevation & Depth

Flat. No shadows, no glass, no gradient fills on surfaces or text. Depth is a plane step: ground → ground-2 (sunken bands, footer) or → panel (sheets, windows). The only gradient-built pixels are the ambient hero grid (96px / 24px lines) and glow, both `aria-hidden` background.

**The Hairline Rule.** Every rule, border and frame is 1px, in `line` or `line-strong`. The only 2px stroke is the focus ring.

## Shapes

Small, firm radii: 2px (focus ring, graph cells), 3px (tags, inline code), 4px (buttons, fields, portraits, code blocks), 6px (windows, framed panels). Sheets, title blocks and date blocks are square. Portraits are square (4px), never circles; the only circles are the window's hollow title-bar dots and the live dot.

## Components

- **Buttons:** primary (primary fill) and ghost (strong hairline, sunken fill on hover); 44px min height, 36px `btn-sm`; 1px press-down on active; 150ms colour transitions on the expo-out ease.
- **Tag** (`.tag`, legacy `.plate`): hairline mono chip for kinds, statuses and post tags; current state inverts to ink.
- **Field:** strong hairline, panel fill, focus swaps border to primary with a flush outline.
- **Window:** strong-hairline 6px frame, quiet title bar, `.kv` mono-label/value rows. Holds the next event with a real Reserve button.
- **Sheet / title block:** `.sheet` square framed panel; `.tblock` hairline cells, each a mono `.tlabel` over a value. `.trim` (legacy) is the 6px framed panel.
- **Section head, DateBlock, EventRow, Avatar** (initials on a handle-hashed quiet tint), **ActivityGraph** (4 green levels from real activity), **SponsorGrid** (shared-hairline cells), **prose** with sugar-high syntax colours (`--sh-*`) and a mono language label on each fence.
- **Motion:** the prompt types once then the caret blinks six times and rests; the live dot pulses; the glow follows a fine pointer. All of it stops under `prefers-reduced-motion`, and all content is present without JS.

## Do's and Don'ts

### Do:
- **Do** give every technical device a real payload: data, code or metadata.
- **Do** keep focus visible: 2px primary outline, 3px offset, everywhere.
- **Do** verify AA contrast for every new text/background pair in both themes.
- **Do** keep the print stylesheet dropping chrome so portfolios print as a CV.

### Don't:
- **Don't** add eyebrow or kicker labels above headings.
- **Don't** use shadows, glass, gradient fills or gradient text.
- **Don't** use brand red for anything but the live dot.
- **Don't** add decorative binary, ASCII plots or fake terminal output; the ASCII/binary plot was rejected as "silly, makes no sense".
- **Don't** bring back painted, hand-lettered or playful display faces.

## shadcn/ui vocabulary

`src/styles.css` also defines the shadcn/ui theme variables in both themes, resolved from the system tokens:
`--background` (ground), `--foreground` (ink), `--card`/`--popover` (panel), `--primary` (drafting green) and
`--primary-foreground`, `--secondary`/`--muted`/`--accent` (sunken ground; in shadcn "accent" is a quiet hover
surface, not a brand colour), `--muted-foreground` (ink-2), `--destructive` (danger), `--border` (line), `--input`
(line-strong), `--ring` (focus), `--radius` (6px), `--chart-1..5`, `--sidebar*`. Tailwind utilities exist for each
(`bg-background`, `text-muted-foreground`, `border-border`). Small primitives with shadcn-shaped APIs live in
`src/components/ui/`: `Button` / `buttonVariants` (variant default | outline | ghost | link, size default | sm),
`Badge` / `badgeVariants` (outline | default | destructive), `Card` (+ Header, Title, Content, Footer).

