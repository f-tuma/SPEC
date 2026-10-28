---
version: alpha
name: Technical Dense
description: A data-first, compact and precise dark system with a cool blue accent, tabular figures and tight spacing. Suits dashboards, developer tools, admin panels, strategy games and simulation HUDs.
colors:
  primary: "#4C9BFF"
  on-primary: "#04111F"
  primary-container: "#12304F"
  on-primary-container: "#B8D8FF"
  secondary: "#8B95A1"
  on-secondary: "#0D1117"
  warning: "#D29922"
  on-warning: "#1F1300"
  background: "#0D1117"
  on-background: "#E6EDF3"
  surface: "#161B22"
  on-surface: "#E6EDF3"
  surface-muted: "#1F2630"
  on-surface-muted: "#9BA7B4"
  border: "#2B3440"
  error: "#FF6B6B"
  on-error: "#2A0505"
  success: "#3FB950"
  on-success: "#04170A"
typography:
  display:
    fontFamily: IBM Plex Sans
    fontSize: 32px
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: -0.01em
  headline:
    fontFamily: IBM Plex Sans
    fontSize: 20px
    fontWeight: 600
    lineHeight: 1.3
  title:
    fontFamily: IBM Plex Sans
    fontSize: 15px
    fontWeight: 600
    lineHeight: 1.35
  body:
    fontFamily: IBM Plex Sans
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: IBM Plex Sans
    fontSize: 12px
    fontWeight: 500
    lineHeight: 1.3
    letterSpacing: 0.02em
  data:
    fontFamily: IBM Plex Mono
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.4
    fontFeature: tnum
rounded:
  sm: 3px
  md: 6px
  lg: 8px
  full: 9999px
spacing:
  xs: 2px
  sm: 4px
  md: 8px
  lg: 16px
  xl: 24px
  gutter: 12px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "{spacing.md}"
  button-primary-hover:
    backgroundColor: "{colors.on-primary-container}"
    textColor: "{colors.on-primary}"
  button-secondary:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.on-surface}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "{spacing.md}"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  table-row:
    textColor: "{colors.on-surface}"
    typography: "{typography.data}"
    padding: "{spacing.sm}"
  table-row-selected:
    backgroundColor: "{colors.primary-container}"
    textColor: "{colors.on-primary-container}"
  input:
    backgroundColor: "{colors.background}"
    textColor: "{colors.on-background}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "{spacing.sm}"
  status-badge:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.on-surface-muted}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.xs}"
---

# Technical Dense

<!-- SPEC design starter: technical-dense. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Technical Dense is built for people who work with a lot of information for a long time. A dark graphite `{colors.background}` with stepped surfaces keeps the eyes rested; a single cool blue `{colors.primary}` marks actions and selection; green, amber and red carry status. Type is compact, numbers are tabular, and layouts favor density and scanning over decoration. It should feel precise and dependable, like a well-made instrument.

**Key characteristics:**
- Dark-first, with three surface steps for hierarchy instead of shadows.
- One blue accent; status colors appear only with a label or icon.
- Compact 4px spacing grid and 14px body text.
- Tabular, monospace figures in tables, metrics and logs.
- Keyboard-first interaction with visible focus.

## Colors

### Brand & accent
- **Signal blue** (`{colors.primary}`): primary actions, focus, selection and links.
- **Deep blue** (`{colors.primary-container}`): selected rows, active tabs and highlighted ranges.
- **Steel** (`{colors.secondary}`): secondary icons, axis labels and inactive controls.

### Surfaces & text
- **Graphite** (`{colors.background}`) for the app frame, **panel** (`{colors.surface}`) for content, **raised** (`{colors.surface-muted}`) for headers, hovers and inputs in panels.
- Text is `{colors.on-background}`; secondary values and units use `{colors.on-surface-muted}`.

### Feedback
- `{colors.success}`, `{colors.warning}` and `{colors.error}` are reserved for status. Each status also has a word or symbol (OK, WARN, FAIL) and a consistent icon.

## Typography

IBM Plex Sans covers UI and text at compact sizes; IBM Plex Mono with tabular figures (`fontFeature: tnum`) covers numbers, IDs, code and logs. Headlines stay small; hierarchy comes from weight and position. Right-align numbers in tables and keep units in muted text.

**Font substitutes:** Inter for IBM Plex Sans; JetBrains Mono for IBM Plex Mono.

## Layout

- **Spacing system:** 4px base; most gaps are `{spacing.md}` or `{spacing.lg}`.
- **Grid:** a fixed sidebar, a toolbar and resizable panels; tables fill the available width.
- **Density:** row height 32px by default with an optional compact 24px mode.

## Elevation & Depth

Hierarchy comes from surface steps and 1px `{colors.border}` dividers. Only popovers, menus and command palettes get a shadow (0 8px 24px at 40% black) plus a border. Avoid glow effects except for focus rings.

## Shapes

Small, precise corners: `{rounded.sm}` for tags and checkboxes, `{rounded.md}` for buttons and inputs, `{rounded.lg}` for panels and dialogs. Icons use 1.5px strokes on a 16px grid.

## Components

- **Buttons:** `button-primary` for the main action in a toolbar; `button-secondary` for everything else; icon buttons have tooltips with shortcuts.
- **Panels:** `panel` with a compact header, title on the left and actions on the right.
- **Tables:** `table-row` with zebra-free rows, hover in `{colors.surface-muted}`, selection with `table-row-selected`, sticky headers and sortable columns.
- **Status:** `status-badge` with a dot, a word and the status color on the dot only.
- **Games:** HUD panels with stats in tabular figures, mini-maps and logs; selection outlines in signal blue; tooltips with exact numbers.

## Do's and Don'ts

### Do
- Show exact values, units and timestamps; let people copy them.
- Keep status meaning consistent across the whole product.
- Support keyboard navigation, shortcuts and a command palette.
- Check WCAG AA contrast for muted text on every surface step.

### Don't
- Don't use status colors for decoration or branding.
- Don't round numbers silently or hide units.
- Don't add illustrations, gradients or large empty hero areas.
- Don't animate data updates in ways that hide the new value.

## Responsive Behavior

- **Wide screens:** multiple resizable panels; tables show more columns.
- **Laptops:** collapsible sidebar; secondary panels become tabs.
- **Phones:** read-only summaries and critical actions; avoid full tables.

## Motion

Fast and functional: 100-150ms for hovers, menus and panel changes. Highlight changed values briefly instead of animating numbers. Respect reduced motion.

## Voice & Tone

Precise, neutral and brief. Use exact terms, avoid marketing language, and write errors as cause plus next step ("Connection refused on port 5432. Check that the database is running.").

## Personalize this starter

- Who uses it, for how long per day, and what is their most frequent task?
- Is dark the right default, or does the context (print, daylight, office) need a light theme first?
- What does the brand accent need to be, and are there domain colors (teams, map layers, resources)?
- Which data types dominate: tables, charts, logs, maps or game stats?
- Which tools or games set the bar for density and clarity for the creator?
