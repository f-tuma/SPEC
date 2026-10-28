---
version: alpha
name: Editorial
description: A typographic, high-contrast system inspired by print magazines, with expressive serif headlines, a strong grid and one bold signal color. Suits publications, blogs, portfolios, storytelling and narrative games.
colors:
  primary: "#A1121C"
  on-primary: "#FFFFFF"
  primary-container: "#F6DCDA"
  on-primary-container: "#5A0A10"
  secondary: "#141414"
  on-secondary: "#FBF9F4"
  background: "#FBF9F4"
  on-background: "#141414"
  surface: "#FFFFFF"
  on-surface: "#141414"
  surface-muted: "#F1EEE6"
  on-surface-muted: "#57534B"
  border: "#D9D4C7"
  error: "#B42318"
  on-error: "#FFFFFF"
  success: "#2F6B3A"
  on-success: "#FFFFFF"
typography:
  display:
    fontFamily: Newsreader
    fontSize: 72px
    fontWeight: 600
    lineHeight: 1.0
    letterSpacing: -0.03em
  headline:
    fontFamily: Newsreader
    fontSize: 40px
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: -0.015em
  title:
    fontFamily: Newsreader
    fontSize: 24px
    fontWeight: 500
    lineHeight: 1.25
  body:
    fontFamily: Source Serif 4
    fontSize: 19px
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: IBM Plex Sans
    fontSize: 12px
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: 0.08em
  ui:
    fontFamily: IBM Plex Sans
    fontSize: 15px
    fontWeight: 500
    lineHeight: 1.4
rounded:
  none: 0px
  sm: 2px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 32px
  xl: 80px
  gutter: 24px
components:
  button-primary:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
    typography: "{typography.ui}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  button-primary-hover:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
  button-secondary:
    backgroundColor: "{colors.background}"
    textColor: "{colors.on-background}"
    typography: "{typography.ui}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  article-card:
    backgroundColor: "{colors.background}"
    textColor: "{colors.on-background}"
    typography: "{typography.title}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  pull-quote:
    textColor: "{colors.primary}"
    typography: "{typography.headline}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    typography: "{typography.ui}"
    rounded: "{rounded.none}"
    padding: "{spacing.sm}"
  kicker:
    textColor: "{colors.primary}"
    typography: "{typography.label}"
---

# Editorial

<!-- SPEC design starter: editorial. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Editorial treats every screen like a well-designed page. Near-black ink on warm paper `{colors.background}`, large expressive serif headlines and a disciplined column grid create authority and rhythm. A single oxblood signal `{colors.primary}` marks kickers, pull quotes and the moment that matters. The voice is confident and curated: fewer elements, set beautifully.

**Key characteristics:**
- Typography is the main visual element; images are framed, never decorative filler.
- Strong contrast: ink and paper, with one signal red.
- A visible column grid, rules and generous margins.
- Square corners and thin rules instead of shadows or cards.
- Small uppercase sans kickers and captions against large serif headlines.

## Colors

### Brand & accent
- **Oxblood** (`{colors.primary}`): kickers, pull quotes, active links and hover on primary buttons.
- **Ink** (`{colors.secondary}`): primary buttons, rules and the masthead.
- **Rose paper** (`{colors.primary-container}`): rare highlight for featured or breaking items.

### Surfaces & text
- **Paper** (`{colors.background}`) is the page; **white** (`{colors.surface}`) only for inputs and overlays; **newsprint** (`{colors.surface-muted}`) for sidebars and asides.
- Body text is `{colors.on-background}`; captions and metadata use `{colors.on-surface-muted}`.

### Feedback
- `{colors.error}` differs from the brand red in tone; always add an explicit word such as "Error" so the two never get confused.

## Typography

Newsreader carries display and headlines with tight tracking and real italics for emphasis. Source Serif 4 is the reading face at 19px with a 65-character measure. IBM Plex Sans handles UI, captions and uppercase kickers with wide tracking. Use true typographic details: proper quotes, en dashes, old-style figures in text and lining figures in tables.

**Font substitutes:** Playfair Display or Fraunces for Newsreader; Merriweather or Lora for Source Serif 4; Inter for IBM Plex Sans.

## Layout

- **Spacing system:** 8px base with large section breaks (`{spacing.xl}`).
- **Grid:** 12 columns on desktop, text spanning 6-7 columns with margins for notes and pull quotes; 4 columns on phones.
- **Composition:** one dominant headline or image per view; asymmetric layouts welcome. Use 1px rules in `{colors.border}` to divide sections.

## Elevation & Depth

Flat, like print. Separation comes from rules, whitespace and tone shifts. Overlays (menus, dialogs) use a 1px ink border and no shadow, or a very subtle one on screens where focus must be obvious.

## Shapes

Square corners (`{rounded.none}`) for images, buttons and inputs. Circles (`{rounded.full}`) only for author portraits. Image crops follow the grid with fixed aspect ratios (3:2, 4:5, 16:9).

## Components

- **Buttons:** ink `button-primary` that turns oxblood on hover; text links underlined with 1px offset.
- **Article cards:** `article-card` with a `kicker`, serif title, short dek and byline; no boxes around them, only rules.
- **Pull quotes:** `pull-quote` in oxblood serif, spanning the margin.
- **Navigation:** a masthead with wordmark and a single row of sections in uppercase sans.
- **Games:** chapter titles as full-screen typographic moments; dialogue in book-like columns; menus as tables of contents.

## Do's and Don'ts

### Do
- Let type size and white space create hierarchy before color.
- Keep the measure between 55 and 75 characters for reading text.
- Use oxblood for one signal per view.
- Respect WCAG AA contrast, especially for captions and metadata.

### Don't
- Don't put text inside boxes, cards or shadows when a rule or space would do.
- Don't use more than three type families.
- Don't use rounded corners or playful icons.
- Don't justify text without hyphenation.

## Responsive Behavior

- **Phones:** single column, headlines scale down by about 40%, pull quotes become full-width breaks.
- **Desktop:** multi-column front pages; long reads keep one central column with side notes.
- **Touch targets:** at least 44 x 44px, including links in lists.

## Motion

Minimal and purposeful: 200ms fades, image reveals on scroll at most once, no parallax on text. Respect reduced motion.

## Voice & Tone

Confident, precise and curated. Headlines are specific and active; captions are informative. UI text is short and neutral so the content keeps the voice.

## Personalize this starter

- Which publication, book or era is the typographic reference: classic newspaper, avant-garde magazine, literary journal?
- What is the signal color, and what is the one thing it should mark?
- How much imagery is there, and who creates it?
- Are long reads central, or is it a portfolio or narrative experience?
- Which typefaces does the creator already love or license?
