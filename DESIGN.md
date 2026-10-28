# SPEC terminal interface

SPEC helps a developer understand a workspace and choose the next useful action. The first screen must answer: which project is open, what is prepared, what needs attention, and what can I do next?

## Layout

- A header line names SPEC, the CLI version, the project and its path. The last line is a Bubbles help bar with the keys of the current screen; above it, one status line holds progress, messages and scroll position.
- At 72 columns and wider, the menu is a filled sidebar on the left; the content column on the right is at most 80 columns. The sidebar also shows the options for the next setup. On screens opened from the menu, the item that opened them stays marked with a quieter highlight, because focus is in the content column.
- Below 72 columns there is one column: home shows the status above the menu, and other screens use the full width. Support 44 x 22; key hints never scroll off-screen.
- Home shows a compact status block (summary, plugin, OpenSpec, agents, documents, notes) and the selected action's explanation. The status block is cut first when space runs out; the menu stays whole.
- Plans, results, errors and document lists replace the content column and scroll within it. Esc returns home.
- Keep the key set small: Up/Down (j/k) and Enter everywhere, Esc back, q quit from home, r refresh. No modes that change what the arrow keys do.

## Text and typography

- Use English for all product copy, commands, examples and documentation.
- Inherit the terminal font and use regular weight throughout. Avoid bold, italic, icon fonts, emoji and decorative Unicode glyphs, which can introduce fallback fonts.
- Use ASCII selection markers and an ASCII spinner. Preserve Unicode in user-authored project names and paths.
- Use spacing, alignment and color to create hierarchy: muted labels, values colored by meaning, filled backgrounds for the sidebar and selection. Provide legible light and dark palettes and textual status labels, so color is not the only signal.
- Wrap reports and long diagnostic tokens; keep their complete content available through scrolling. Strip terminal controls from paths and external diagnostics.

## First use and actions

- A valid project opens directly into its local, read-only overview. Ask for a path when it is missing or invalid. Cancelling a project switch keeps the original project.
- The menu offers only actions that can run now, with the recommended one first: Set up workspace for a new project, Repair setup when something needs attention, otherwise opening an agent. The status block explains why other actions are absent.
- The selected item's explanation appears below the status (in the status line on narrow terminals).
- Setup options are a menu item. They change the next setup in memory: selected agents, OpenSpec source, native plugin setup, document language and design review (SPEC or the EGM plugin). The options start from the recorded document language and review procedure; the side column shows both under Next setup. Document language is an inline text field: Enter edits and saves, Esc cancels, an empty value follows the conversation and an invalid name keeps the field open with an error. Never allow an empty agent selection.
- Project documents is a list; Enter reads a document with rendered Markdown in the content column, e edits it. Rendered Markdown keeps regular weight and ASCII structure (# headings, - lists, | quotes); document content may contain any Unicode. The editor shows its mode and unsaved state in the title, and its keys in the help bar. Ctrl+C never quits from the editor.
- Keep the menu short. Agents start from the shell (spec run), rollback is a CLI command, and design work happens in the agent conversation. Update appears only as an offer when a quiet background check finds a newer stable release; that check never prompts and fails silently.
- Setup and update show a readable plan with relevant versions and options. Enter/y applies that exact plan; Esc/n cancels without mutation.
- Give every error an explanation and a recovery step. Refresh recorded state after operations, including partial failures. Esc returns to the overview for a retry.
- Returning from an external editor restores the dashboard. Ctrl+C during a managed operation waits for cleanup before quitting.

## Meaning of status

- Distinguish an unconfigured workspace, incomplete OpenSpec, pending or stale clients, modified managed files and inconsistent catalogs.
- Show recorded versions as recorded state. A status read does not execute OpenSpec or an agent.
- A successful native installer means installation completed. It does not prove skills were invoked in a fresh session.
- Candidate document discovery supplies paths for review. It does not establish content quality, completeness or passing tests.
- Show human summaries, labeled values and next actions in the TUI. Preserve the JSON CLI contract for automation.
