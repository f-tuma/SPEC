# Document reader and editor in the TUI

The owner wants to open, read and edit the available project documents inside the TUI, with rendered Markdown and a choice of vim motions, nano-like keys or their own editor. It must stay small and Charm/Go native.

## Required behavior

- Project documents lists discovered documents and active OpenSpec change files. Enter shows a document; Markdown is rendered with `charm.land/glamour/v2` (pinned v2.0.1) using a SPEC style: palette colors, regular weight, ASCII structure, no syntax highlighting. YAML front matter is shown as a code block. Document text is sanitized so it cannot inject terminal controls.
- `e` edits the document with the key style chosen in Setup options and saved per user (`$XDG_CONFIG_HOME/SPEC/tui.json` or the platform equivalent): nano keys (default), vim keys, or `$VISUAL`/`$EDITOR` through the same terminal hand-over as agents.
- The built-in editor is the Bubbles v2 textarea. Nano keys add Ctrl+S, Ctrl+X with an unsaved-changes guard, line cut/paste, search and undo/redo. Vim keys add normal, insert and visual (`v` character-wise, `V` line-wise) modes, common motions and operators, search, undo/redo and `:w`, `:q`, `:wq`, `:q!`, `:w!`, `:NUMBER`. Visual mode extends with the same motions and supports `y d x c p > < o`; its highlight uses the textarea's own keyboard selection. Nano keys select with Shift+arrows, and Ctrl+K cuts a selection. Counts, text objects, block visual mode and named registers are out of scope.
- Saves go through the application service: inside the project, never in `.spec`, atomic, keeping the file mode, and refused when the file changed since it was read unless forced. Content the textarea would alter on load (tabs, carriage returns, beyond its line limit) is not opened in the built-in editor.
- Ctrl+C never quits from the editor. No other dependency is added; goeditor, vimtea and teacup were evaluated and rejected (risk, Bubble Tea v1, missing nano mode).

## Verification and stop condition

Test the Markdown style for ASCII output without bold or italic and with sanitized controls; test document listing, guarded and atomic writes, the nano and vim journeys with saved bytes, the lossless-content guard, concurrent-change refusal, the external editor command and preference persistence in an isolated configuration directory; keep every screen within supported terminal sizes. Run the repository Go checks and exercise both key styles in a real terminal. Native behavior of each user's external editor is outside this check.
