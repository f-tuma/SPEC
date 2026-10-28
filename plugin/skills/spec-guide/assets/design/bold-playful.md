---
version: alpha
name: Bold Playful
description: A saturated, rounded and lively system with confident color blocks and chunky type. Suits consumer apps, learning tools, party and casual games.
colors:
  primary: "#5B2BD9"
  on-primary: "#FFFFFF"
  primary-container: "#EDE4FF"
  on-primary-container: "#2A0E73"
  secondary: "#FFB703"
  on-secondary: "#2A1A00"
  accent: "#FF4D8D"
  on-accent: "#2A0016"
  background: "#FFF8EE"
  on-background: "#24123F"
  surface: "#FFFFFF"
  on-surface: "#24123F"
  surface-muted: "#F6EEFF"
  on-surface-muted: "#5A4A78"
  border: "#E3D6F7"
  error: "#C4122F"
  on-error: "#FFFFFF"
  success: "#1C7C54"
  on-success: "#FFFFFF"
typography:
  display:
    fontFamily: Fredoka
    fontSize: 56px
    fontWeight: 700
    lineHeight: 1.05
    letterSpacing: -0.01em
  headline:
    fontFamily: Fredoka
    fontSize: 32px
    fontWeight: 600
    lineHeight: 1.15
  title:
    fontFamily: Fredoka
    fontSize: 22px
    fontWeight: 600
    lineHeight: 1.25
  body:
    fontFamily: Nunito
    fontSize: 17px
    fontWeight: 500
    lineHeight: 1.55
  label:
    fontFamily: Nunito
    fontSize: 14px
    fontWeight: 800
    lineHeight: 1.2
    letterSpacing: 0.01em
rounded:
  sm: 8px
  md: 16px
  lg: 24px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 40px
  gutter: 16px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.md}"
  button-primary-hover:
    backgroundColor: "{colors.on-primary-container}"
    textColor: "{colors.on-primary}"
  button-secondary:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.md}"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  card-highlight:
    backgroundColor: "{colors.primary-container}"
    textColor: "{colors.on-primary-container}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "{spacing.sm}"
  badge:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    typography: "{typography.label}"
    rounded: "{rounded.full}"
    padding: "{spacing.xs}"
---

# Bold Playful

<!-- SPEC design starter: bold-playful. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Bold Playful is cheerful and confident. Electric violet `{colors.primary}`, sunny yellow `{colors.secondary}` and a hot pink `{colors.accent}` meet on a warm cream `{colors.background}`. Shapes are round and chunky, type is friendly and heavy, and every interaction answers with a little bounce. It should feel like a toy that is also well made: loud, but never messy.

**Key characteristics:**
- Three saturated colors with fixed jobs: violet acts, yellow supports, pink celebrates.
- Pill buttons, big radii and thick 2px outlines.
- Rounded display type (Fredoka) paired with a sturdy body face (Nunito).
- Color blocks instead of shadows to separate areas.
- Playful motion on success and progress, calm motion everywhere else.

## Colors

### Brand & accent
- **Electric violet** (`{colors.primary}`): primary actions, progress, selected states.
- **Sunshine** (`{colors.secondary}`): secondary actions and supportive highlights; always with dark `{colors.on-secondary}` text.
- **Pop pink** (`{colors.accent}`): rewards, streaks, "new" badges and celebration moments only.

### Surfaces & text
- **Cream** (`{colors.background}`) keeps the palette warm; **white** (`{colors.surface}`) for cards; **lilac** (`{colors.surface-muted}`) for grouped areas.
- **Night violet** (`{colors.on-background}`) is the text color everywhere; pure black is never used.

### Feedback
- `{colors.success}` and `{colors.error}` keep their usual meaning and always come with an icon and a short label.

## Typography

Fredoka carries headlines with soft, rounded terminals; Nunito keeps body text readable at weight 500. Labels and buttons use Nunito 800 for punch. Keep headlines short: two lines at most. Avoid all-caps except for one-word badges.

**Font substitutes:** Baloo 2 or Quicksand for Fredoka; Rubik for Nunito.

## Layout

- **Spacing system:** 8px rhythm; components use `{spacing.lg}` inner padding to feel plump.
- **Grid:** cards in a flexible grid with a `{spacing.gutter}` gap; on phones one card per row.
- **Composition:** one big hero element per screen (illustration, character, score) and everything else smaller.

## Elevation & Depth

Depth comes from color blocks and a 2px outline in `{colors.on-background}` at 12% opacity. Pressed buttons move down 2px; raised elements may use a solid offset shadow (0 4px 0) in a darker shade of their own color. No blurry, gray shadows.

## Shapes

Everything is round: `{rounded.full}` for buttons, chips and avatars, `{rounded.lg}` for cards and sheets, `{rounded.md}` for inputs. Icons are filled, chunky and rounded. Decorative blobs and confetti may appear in empty states and celebrations.

## Components

- **Buttons:** pill-shaped `button-primary`; secondary actions in `button-secondary`. A press squashes the button slightly.
- **Cards:** `card` for content, `card-highlight` for the recommended or active item.
- **Badges:** `badge` in pop pink for new and rewarded states only.
- **Progress:** thick rounded bars in violet on lilac; milestones get a short celebration.
- **Games:** score and lives in chunky rounded counters; level select as big tiles; dialogs with a character or icon.

## Do's and Don'ts

### Do
- Give each saturated color one job and keep it.
- Celebrate real progress (completed, leveled up, saved) with motion and color.
- Keep text on saturated colors short and check WCAG AA contrast.
- Use illustrations or characters with a consistent style and outline weight.

### Don't
- Don't put long text on violet, yellow or pink backgrounds.
- Don't celebrate trivial actions; delight loses meaning when it is constant.
- Don't mix sharp corners into the round language.
- Don't use thin, light font weights.

## Responsive Behavior

- **Phones:** full-width pill buttons at the bottom; cards stack.
- **Tablets and desktop:** two to four column card grids; illustrations grow, text does not.
- **Touch targets:** at least 48 x 48px.

## Motion

Use springy easing (slight overshoot) for success, rewards and appearing cards, 250-400ms. Navigation and errors use simple 200ms ease-out. Offer reduced motion: no bounce, no confetti.

## Voice & Tone

Upbeat, warm and encouraging, like a friendly coach. Short sentences, active verbs, occasional playful words. Errors stay helpful and never joke at the user's expense.

## Personalize this starter

- Who is the audience (kids, teens, adults, families) and how loud may the experience be?
- Do the product, mascot or world suggest different signature colors?
- Which moments deserve a celebration, and what does progress mean here?
- Is there a character or illustration style that should lead the visual identity?
- Which references (games, toys, cartoons, apps) does the creator love?
