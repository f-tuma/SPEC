# Design starters and guided personalization

The owner wants to browse basic DESIGN.md starters, pick one for a project and then shape it with a guide so the project gains a personality that matches the creator's vision. Add this as one bounded checkpoint.

## Required behavior

- Ship six original starters with distinct personalities (Bold Playful, Calm Minimal, Cozy Warm, Editorial, Retro Pixel, Technical Dense) as plugin assets. Each follows the Google DESIGN.md format: front matter tokens (`version`, `name`, `description`, `colors`, `typography`, `rounded`, `spacing`, `components`) and the canonical sections in order, plus Responsive Behavior, Motion, Voice & Tone and a removable "Personalize this starter" section with a starter marker comment.
- Every `X`/`on-X` color pair, muted text on surface and the primary color on its backgrounds meet WCAG AA (4.5:1). Token references resolve. Starter text is ASCII English; fonts have open licenses with named substitutes. No starter imitates a real brand.
- `spec design --project DIR` lists starters as JSON; `--starter ID` previews and `--write` creates DESIGN.md exclusively. An existing DESIGN.md is never replaced. Other design sources are reported so one token authority can be kept.
- Choosing a starter is part of the agent conversation, not the dashboard (owner decision after the first terminal review: a palette and a mood line are not enough to choose a design system). The agent lists starters with the CLI and presents each as a complete system.
- spec-guide gains a design reference: authority checks, a conversational interview (identity, references, world, color, type, shape, motion, voice, exceptions), writing rules (accessible pairs, token references, project-specific do's and don'ts, document language) and checks. It does not certify the official linter or the implemented UI.

## Verification and stop condition

Test starter structure, order, references and contrast in production parsing code; test exclusive creation, preview without writes and refusal to overwrite; test the CLI including the existing-file block. Run the repository Go checks and inspect the browser in a real terminal. Native agents running the interview remain unverified until a fresh session is observed.
