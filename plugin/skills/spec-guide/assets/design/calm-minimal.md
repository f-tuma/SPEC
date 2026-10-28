---
version: alpha
name: Calm Minimal
description: A quiet, airy system with one restrained accent, generous whitespace and clear typography. Suits focused tools, wellbeing apps, portfolios and contemplative games.
colors:
  primary: "#2E5E5A"
  on-primary: "#FFFFFF"
  primary-container: "#DCEAE7"
  on-primary-container: "#173532"
  secondary: "#5B6470"
  on-secondary: "#FFFFFF"
  background: "#FAFAF7"
  on-background: "#1D1F21"
  surface: "#FFFFFF"
  on-surface: "#1D1F21"
  surface-muted: "#F1F1EC"
  on-surface-muted: "#565B60"
  border: "#DEDED7"
  error: "#B3261E"
  on-error: "#FFFFFF"
  success: "#2E7D4F"
  on-success: "#FFFFFF"
typography:
  display:
    fontFamily: Manrope
    fontSize: 48px
    fontWeight: 600
    lineHeight: 1.1
    letterSpacing: -0.02em
  headline:
    fontFamily: Manrope
    fontSize: 28px
    fontWeight: 600
    lineHeight: 1.25
  title:
    fontFamily: Manrope
    fontSize: 20px
    fontWeight: 600
    lineHeight: 1.35
  body:
    fontFamily: Manrope
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: Manrope
    fontSize: 13px
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: 0.02em
  mono:
    fontFamily: IBM Plex Mono
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sm: 4px
  md: 8px
  lg: 12px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 32px
  xl: 64px
  gutter: 24px
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
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "{spacing.md}"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "{spacing.sm}"
  badge:
    backgroundColor: "{colors.primary-container}"
    textColor: "{colors.on-primary-container}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.xs}"
---

# Calm Minimal

<!-- SPEC design starter: calm-minimal. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Calm Minimal lets content breathe. Most of the screen is quiet off-white `{colors.background}`; one deep teal accent `{colors.primary}` marks the single most important action or state. Hierarchy comes from type size, weight and space rather than color, borders or shadows. The experience should feel unhurried and trustworthy, like a well-kept notebook.

**Key characteristics:**
- One accent color, used sparingly; everything else is ink on paper.
- Generous whitespace with a strict 8px rhythm and wide margins.
- Flat surfaces separated by space and hairline `{colors.border}`, rarely by shadow.
- A single humanist sans family (Manrope) in a small, consistent scale.
- Motion is short and soft; nothing bounces or flashes.

## Colors

### Brand & accent
- **Deep teal** (`{colors.primary}`): primary actions, focus rings, the active navigation item and selected states.
- **Teal mist** (`{colors.primary-container}`): quiet highlights, badges and selected rows.
- **Slate** (`{colors.secondary}`): secondary icons and controls that must stay visible without competing.

### Surfaces & text
- **Paper** (`{colors.background}`) is the page; **white** (`{colors.surface}`) lifts cards and dialogs a single step.
- **Ink** (`{colors.on-background}`) carries all primary text. Use `{colors.on-surface-muted}` for supporting text on `{colors.surface-muted}` panels.

### Feedback
- `{colors.error}` and `{colors.success}` appear only with a text label or icon; color is never the only signal.

## Typography

Manrope gives a calm, slightly rounded humanist voice that stays legible at small sizes. Headlines use weight 600, never heavier. Body text uses 1.6 line height and a measure of 60-75 characters. Labels use a little positive tracking. IBM Plex Mono is reserved for codes, numbers in tables and technical values.

**Font substitutes:** Inter or system-ui for Manrope; any neutral monospace for IBM Plex Mono.

## Layout

- **Spacing system:** multiples of 8px (`{spacing.sm}` to `{spacing.xl}`); `{spacing.xs}` only inside dense controls.
- **Grid:** 12 columns with a `{spacing.gutter}` gutter on wide screens, 4 columns on phones. Content measure stays under 720px for reading views.
- **Whitespace:** separate sections with `{spacing.xl}`. When a screen feels empty, remove elements before adding decoration.

## Elevation & Depth

Depth is nearly flat. Cards sit on `{colors.surface}` over `{colors.background}` with an optional hairline border. Only overlays (menus, dialogs, toasts) get a soft shadow: 0 8px 24px with 8% ink opacity. Never stack more than two levels.

## Shapes

Gentle, consistent corners: `{rounded.md}` for buttons and inputs, `{rounded.lg}` for cards and dialogs, `{rounded.full}` only for badges, avatars and toggles. Icons use 1.5px strokes with rounded caps.

## Components

- **Buttons:** one `button-primary` per view. Secondary actions use `button-secondary` or a text link. Minimum height 40px, 44px on touch.
- **Cards:** `card` with `{spacing.lg}` padding; a title, one line of supporting text and at most two actions.
- **Inputs:** labels above the field, helper text below in `{colors.on-surface-muted}`, a 2px `{colors.primary}` focus ring.
- **Badges:** short status words in `badge`; never more than one per row.
- **Empty states:** one sentence, one illustration or icon at most, and one action.
- **Games:** HUD elements stay at screen edges with plenty of safe area; menus are lists, not grids of tiles.

## Do's and Don'ts

### Do
- Keep a single primary action per screen and make it the only saturated element.
- Let whitespace group content before reaching for lines or boxes.
- Maintain WCAG AA contrast (4.5:1 for text, 3:1 for large text and controls).
- Write short, plain copy; one idea per sentence.

### Don't
- Don't add a second accent color for decoration.
- Don't use heavy shadows, gradients or glass effects.
- Don't fill empty space with illustrations, tips or banners.
- Don't use bold weights above 600 or all-caps headlines.

## Responsive Behavior

- **Phones:** single column, `{spacing.md}` side margins, primary action reachable by thumb.
- **Desktop:** content centered with a maximum width; navigation in a quiet sidebar or top bar.
- **Touch targets:** at least 44 x 44px.

## Motion

Transitions last 150-200ms with ease-out. Fade and slide by at most 8px. Respect reduced-motion settings by removing movement and keeping fades.

## Voice & Tone

Calm, direct and kind. Prefer verbs over nouns ("Save changes", not "Change saving"). Never use exclamation marks in system messages.

## Personalize this starter

- What should a person feel in the first ten seconds? Is calm the right promise for this project?
- Which single color should carry the accent? Does it come from the product's world, logo or subject?
- Is the project mostly reading, doing, or playing? Adjust the measure, density and type scale.
- Which screens or moments deserve an exception to the quiet rule?
- Which real references (apps, games, print, places) does the creator love, and what exactly about them?
