---
version: alpha
name: Retro Pixel
description: A crisp, nostalgic 8/16-bit system with a dark night palette, neon accents, square shapes and pixel display type. Suits indie and arcade games, game jams, retro tools and playful developer products.
colors:
  primary: "#FF5C8A"
  on-primary: "#17132B"
  primary-container: "#4A1D3A"
  on-primary-container: "#FFC2D4"
  secondary: "#3EE6C1"
  on-secondary: "#0D2B25"
  accent: "#FFD23F"
  on-accent: "#2B2100"
  background: "#17132B"
  on-background: "#F2EEE3"
  surface: "#221C3D"
  on-surface: "#F2EEE3"
  surface-muted: "#2D2650"
  on-surface-muted: "#B9B2D6"
  border: "#4B4378"
  error: "#FF6B57"
  on-error: "#1A0703"
  success: "#7BE06A"
  on-success: "#0E2A09"
typography:
  display:
    fontFamily: Press Start 2P
    fontSize: 32px
    fontWeight: 400
    lineHeight: 1.4
  headline:
    fontFamily: Press Start 2P
    fontSize: 20px
    fontWeight: 400
    lineHeight: 1.5
  title:
    fontFamily: Silkscreen
    fontSize: 20px
    fontWeight: 400
    lineHeight: 1.3
  body:
    fontFamily: IBM Plex Mono
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: Silkscreen
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: 0.04em
rounded:
  none: 0px
  sm: 2px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 48px
  gutter: 16px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  button-primary-hover:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
  button-secondary:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-surface}"
    rounded: "{rounded.none}"
    padding: "{spacing.md}"
  dialog:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.on-surface}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
    padding: "{spacing.lg}"
  input:
    backgroundColor: "{colors.background}"
    textColor: "{colors.on-background}"
    typography: "{typography.body}"
    rounded: "{rounded.none}"
    padding: "{spacing.sm}"
  badge:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.none}"
    padding: "{spacing.xs}"
---

# Retro Pixel

<!-- SPEC design starter: retro-pixel. Personalize it with the spec-guide skill, then remove this comment and the "Personalize this starter" section. -->

## Overview

Retro Pixel brings back the glow of cartridge-era games on a modern screen. A deep night `{colors.background}` holds neon pink `{colors.primary}`, mint `{colors.secondary}` and coin yellow `{colors.accent}`. Every edge is square and aligned to a 4px pixel grid, display text uses a true pixel font, and depth comes from hard, unblurred offsets. It should feel crisp and nostalgic, never blurry or noisy.

**Key characteristics:**
- Dark-first palette with a small set of neon accents, like a limited console palette.
- Square corners everywhere; `{rounded.full}` only for tiny round indicators.
- Pixel display type (Press Start 2P, Silkscreen) with a readable monospace body.
- Hard 4px offset shadows and 2px solid borders instead of blur.
- Everything snaps to a 4px grid; sprites and icons scale by whole multiples only.

## Colors

### Brand & accent
- **Neon pink** (`{colors.primary}`): primary actions, the player, selected items.
- **Mint** (`{colors.secondary}`): secondary actions, positive state, links.
- **Coin yellow** (`{colors.accent}`): rewards, score, hover and focus highlights.

### Surfaces & text
- **Night** (`{colors.background}`) is the screen; **dusk** (`{colors.surface}`) holds panels; **twilight** (`{colors.surface-muted}`) holds dialogs and pressed states.
- **Bone** (`{colors.on-background}`) for text; `{colors.on-surface-muted}` for hints and disabled labels.

### Feedback
- `{colors.error}` and `{colors.success}` are bright on dark; pair them with a symbol or word such as "HIT" or "OK".

## Typography

Use Press Start 2P only for logos, titles and short headlines; it is wide and tiring in long text. Silkscreen works for labels, buttons and HUD values. Body and dialogue use IBM Plex Mono for readability. Render pixel fonts at multiples of their native size with font smoothing off where the platform allows.

**Font substitutes:** VT323 or Pixelify Sans for pixel styles; JetBrains Mono for IBM Plex Mono.

## Layout

- **Spacing system:** a 4px pixel unit; every position and size is a multiple of 4.
- **Grid:** fixed tile grids (8, 16 or 32px tiles) for game scenes; a 12-column grid for menus and web pages.
- **Composition:** framed panels with clear borders; HUD in screen corners with a safe margin.

## Elevation & Depth

Raised elements get a hard shadow: 4px right and 4px down in `{colors.background}` or black, never blurred. Pressed elements move 2px down-right and lose the shadow. Borders are 2px solid `{colors.border}`; the focused element gets a 2px `{colors.accent}` outline.

## Shapes

Square by default (`{rounded.none}`). Optional stepped "pixel corners" can be drawn by cutting one pixel from each corner. Icons are drawn on a 16 x 16 or 24 x 24 pixel grid with no anti-aliasing.

## Components

- **Buttons:** `button-primary` with a hard shadow; hover and focus switch to coin yellow. A `>` cursor marks the focused menu item.
- **Panels and dialogs:** `panel` for HUD and menus; `dialog` for conversation boxes with a speaker label in Silkscreen.
- **Inputs:** `input` with a blinking block cursor.
- **Badges:** `badge` for level, rarity or status.
- **Games:** health as segmented bars, score in Silkscreen, pause menus as centered panels, full keyboard and controller navigation.

## Do's and Don'ts

### Do
- Snap everything to the pixel grid and scale artwork by whole numbers.
- Keep the palette limited; add a new color only for a new meaning.
- Support keyboard and controller navigation with a visible cursor.
- Keep body text in the readable monospace and check WCAG AA contrast.

### Don't
- Don't use blur, soft shadows, gradients or glass.
- Don't set paragraphs in Press Start 2P.
- Don't rotate or scale pixel art by fractional amounts.
- Don't add screen effects (scanlines, CRT curve) that cannot be turned off.

## Responsive Behavior

- **Games:** choose an internal resolution (such as 320 x 180) and scale by whole multiples, letterboxing the rest.
- **Web and apps:** the same palette and shapes with normal responsive layout; pixel type stays for headings only.
- **Touch targets:** at least 48 x 48px, even when the visual is smaller.

## Motion

Step-based animation at low frame rates (8-12 fps) for sprites; UI moves in 4px steps over 100-150ms. Blinking cursors and coin spins are welcome. Respect reduced motion and offer an option to turn off flashing.

## Voice & Tone

Short, punchy and a little cheeky, like classic game text. Use verbs and capital words for big moments ("LEVEL UP", "CONTINUE?"). Keep instructions clear; nostalgia never excuses confusion.

## Personalize this starter

- Which era or console should it evoke (8-bit, 16-bit, handheld, arcade), and how strictly?
- What is the internal resolution and tile size of the game or art?
- Which palette fits the world: night neon, earthy dungeon, pastel handheld?
- Is there a mascot or player character whose colors should lead?
- Which games or artists are the creator's references, and what exactly about them?
