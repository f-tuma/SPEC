---
version: alpha
name: Cozy Warm
description: A soft, earthy and hand-made system with warm neutrals, a terracotta accent and gentle serif headlines. Suits lifestyle apps, journals, recipes, crafts and cozy games.
colors:
  primary: "#9C4A2F"
  on-primary: "#FFF8F1"
  primary-container: "#F4DCCB"
  on-primary-container: "#4A2112"
  secondary: "#5E6F4E"
  on-secondary: "#FFFFFF"
  background: "#F7F0E6"
  on-background: "#3A2A20"
  surface: "#FFFBF5"
  on-surface: "#3A2A20"
  surface-muted: "#EFE5D6"
  on-surface-muted: "#6B5848"
  border: "#E0D2BF"
  error: "#A8322B"
  on-error: "#FFFFFF"
  success: "#4F7A3A"
  on-success: "#FFFFFF"
typography:
  display:
    fontFamily: Fraunces
    fontSize: 52px
    fontWeight: 500
    lineHeight: 1.1
    letterSpacing: -0.01em
  headline:
    fontFamily: Fraunces
    fontSize: 30px
    fontWeight: 500
    lineHeight: 1.2
  title:
    fontFamily: Fraunces
    fontSize: 21px
    fontWeight: 600
    lineHeight: 1.3
  body:
    fontFamily: Nunito Sans
    fontSize: 17px
    fontWeight: 400
    lineHeight: 1.65
  label:
    fontFamily: Nunito Sans
    fontSize: 14px
    fontWeight: 700
    lineHeight: 1.3
rounded:
  sm: 6px
  md: 12px
  lg: 20px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 28px
  xl: 56px
  gutter: 20px
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
    backgroundColor: "{colors.primary-container}"
    textColor: "{colors.on-primary-container}"
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
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.xs}"
---

# Cozy Warm

<!-- SPEC design starter: cozy-warm. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Cozy Warm feels like a kitchen table on a slow afternoon. Warm oat `{colors.background}`, cream `{colors.surface}` and brown ink create a soft base; terracotta `{colors.primary}` and sage `{colors.secondary}` add color drawn from clay, plants and wood. Gentle serif headlines and textured details make it feel hand-made rather than manufactured.

**Key characteristics:**
- Warm neutrals only; no pure white, no pure black, no cold grays.
- Terracotta for action, sage for calm confirmation and categories.
- Fraunces serif headlines with a soft, slightly quirky character.
- Rounded, pillowy shapes and soft warm shadows.
- Room for texture: paper grain, hand-drawn icons, photography with natural light.

## Colors

### Brand & accent
- **Terracotta** (`{colors.primary}`): primary actions, links, selected states.
- **Clay blush** (`{colors.primary-container}`): secondary buttons, highlights, gentle selection.
- **Sage** (`{colors.secondary}`): tags, categories, positive confirmation next to text.

### Surfaces & text
- **Oat** (`{colors.background}`) for the page, **cream** (`{colors.surface}`) for cards, **linen** (`{colors.surface-muted}`) for grouped areas and sidebars.
- **Cocoa** (`{colors.on-background}`) for text; `{colors.on-surface-muted}` for supporting text.

### Feedback
- `{colors.error}` and `{colors.success}` are earthy versions of red and green and always come with a label.

## Typography

Fraunces at weight 500 gives headlines warmth and personality; keep its optical size high for display text. Nunito Sans keeps body text friendly and readable with generous 1.65 line height. Use italic Fraunces sparingly for quotes or hand-written notes.

**Font substitutes:** Recoleta or Lora for Fraunces; Source Sans 3 for Nunito Sans.

## Layout

- **Spacing system:** 4px base with a relaxed rhythm (`{spacing.lg}` between related groups, `{spacing.xl}` between sections).
- **Grid:** soft masonry or two-column layouts for collections; single column for reading.
- **Composition:** let photos or illustrations bleed to card edges; keep text on calm areas.

## Elevation & Depth

Use soft, warm shadows tinted with cocoa (0 6px 20px at 10% opacity) for cards and sheets. Layers may overlap slightly, like paper on a table. Avoid hard outlines; when a border is needed use `{colors.border}`.

## Shapes

Pillowy corners: `{rounded.lg}` for cards and sheets, `{rounded.md}` for buttons and inputs, `{rounded.full}` for tags and avatars. Icons are hand-drawn or rounded with slightly irregular strokes.

## Components

- **Buttons:** `button-primary` for the main action; `button-secondary` in clay blush for gentle alternatives.
- **Cards:** `card` with an image on top, a serif title and a short description.
- **Tags:** sage `badge` for categories and states.
- **Lists:** comfortable rows with an icon or thumbnail and plenty of padding.
- **Games:** inventory and crafting in soft cards; dialogs as paper notes; day and season changes tint the background.

## Do's and Don'ts

### Do
- Keep every neutral warm; tint grays toward brown.
- Use natural textures and imagery with soft light.
- Keep contrast at WCAG AA even with soft colors.
- Write as if speaking to a friend at the table.

### Don't
- Don't use pure white backgrounds or pure black text.
- Don't use neon, electric blue or saturated purple.
- Don't use sharp corners or hard drop shadows.
- Don't crowd screens; coziness needs room.

## Responsive Behavior

- **Phones:** single column, bottom navigation with labeled icons.
- **Desktop:** two columns for collections; reading views keep a 680px measure.
- **Touch targets:** at least 44 x 44px.

## Motion

Slow and soft: 250-300ms ease-in-out, gentle fades and small lifts. Small ambient motion (steam, leaves, flicker) is welcome in illustrations but never behind text. Respect reduced motion.

## Voice & Tone

Warm, gentle and personal. Use "you" and "we", invite rather than instruct ("Let's start a new recipe"). Errors are calm and reassuring.

## Personalize this starter

- What everyday place, season or ritual should the project feel like?
- Which materials (clay, wood, wool, paper, herbs) belong to its world, and which colors follow from them?
- Is there photography, illustration or hand lettering? Who makes it?
- How much texture and imperfection is welcome?
- Which references (books, cafes, games, apps) does the creator find cozy, and why?
