# Visual identity and DESIGN.md

Use this when a project needs a visual identity, when the owner chose a SPEC design starter, or when an existing DESIGN.md still carries the `SPEC design starter` comment. The goal is a design that expresses the creator's own vision, not a tidier copy of the starter.

## Sources and authority

Check for an existing design authority first: a design-system package, token files, a component catalog or a DESIGN.md. Keep one token authority; do not create a second, conflicting token set. A starter is a draft, never accepted intent. Values the creator confirms become decisions; everything else stays marked as a proposal.

## Presenting the starters

Starters are conversation material, not a catalog to click through. Run `spec design --project ROOT` and read the starter files in this skill's `assets/design/` before describing them. Present each relevant starter as a whole system, never only its colors and mood: palette roles and contrast, type pairing and scale, shapes and corner radii, depth (flat, shadows, hard offsets), spacing rhythm and density, the components it defines, responsive behavior, motion and voice, plus what it suits and where it fails. Offer two or three that fit what the creator already said, explain the trade-offs, and invite mixing ("Editorial typography with Cozy Warm materials") or rejecting all of them. Create DESIGN.md from a starter only after the creator chooses a direction; otherwise write it from the interview.

SPEC starters follow the Google DESIGN.md format: YAML front matter with `version`, `name`, `description`, `colors`, `typography`, `rounded`, `spacing` and `components`, then the sections Overview, Colors, Typography, Layout, Elevation & Depth, Shapes, Components and Do's and Don'ts. Extra sections (Responsive Behavior, Motion, Voice & Tone) are allowed. Tokens are the normative values; prose explains how to use them. `spec design --project ROOT` lists the starters; `--starter ID --write` creates DESIGN.md only when none exists.

## The design interview

Hold a conversation, not a form. Ask a few questions at a time, reflect what you heard and show small concrete proposals (a palette, a type pairing, a sample screen description) for the creator to react to. Stop asking once the answers stop changing the design.

1. **Identity.** What is the project, who is it for, and what should someone feel in the first ten seconds? Collect three to five adjectives and one thing it must never feel like.
2. **References.** Which apps, games, films, books, places, materials or artists does the creator love? For each, ask what exactly they like about it. Translate the qualities into rules; do not copy another brand's identity, logo or trademarked look.
3. **World and content.** What will screens mostly show (text, data, images, game scenes)? Which platforms (web, mobile, desktop, console, terminal) matter first?
4. **Color.** Find the signature color in the project's own world. Assign every color a role (`primary`, `secondary`, surfaces, feedback). Decide light, dark or both.
5. **Typography.** Choose families for display, reading and UI from fonts the project may actually use (open licenses such as the Google Fonts catalog, or fonts the creator licensed). Record substitutes.
6. **Shape, depth and density.** Corners, borders, shadows, spacing rhythm and information density. Tie each to the identity adjectives.
7. **Motion and sound.** How things move and respond; for games also feedback, juice and accessibility options.
8. **Voice.** How the product talks in buttons, errors, onboarding and celebration.
9. **Exceptions.** Which moments may break the rules (a title screen, a launch page, a celebration)?

## Writing the design

- Start from the chosen starter's structure and replace its values and prose with the creator's decisions. Rename the system (`name`) to the project's own name.
- Keep color pairs accessible: text and its background meet WCAG AA (4.5:1 for body text, 3:1 for large text and controls). Report any pair you could not verify.
- Keep component tokens as references (`{colors.primary}`) to primitive tokens. Add components the project actually needs (for games: HUD, dialog, inventory, menu; for tools: table, toolbar).
- Make Do's and Don'ts specific to this project; each rule should be checkable in a screenshot.
- Write in the project's document language (see the CLI reference). Keep token names in English.
- When the creator accepts the design, remove the `SPEC design starter` comment and the "Personalize this starter" section, and record remaining open questions in the brief or the active change.

## Checking it

Show the creator a summary of the decisions and the remaining hypotheses before writing large changes. After writing, verify that every token reference resolves, that the section order follows the format and that contrast pairs pass. If the official `@google/design.md` linter is available in the project, it may be run on request; otherwise say it was not run. A DESIGN.md is not proof that the implemented UI matches it; compare real screens separately.
