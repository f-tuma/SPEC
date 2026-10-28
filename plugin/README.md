# SPEC plugin 0.4.0

Portable plugin for Claude Code and Codex. Skills: spec-guide, spec-review, spec-checkpoint. The companion SPEC Go binary embeds this package, installs pinned OpenSpec and manages plugin versions. It offers a Charm TUI and JSON CLI; no Python runtime is required.

Run `spec tui --project /absolute/project` or `spec install --project /absolute/project`. Use --dry-run to preview, or --no-register to skip agent setup. Claude Code gets this plugin with project scope; Codex gets the skills copied into the project's .agents/skills. Start the client with `spec run --project /absolute/project --agent claude` (or codex), then verify skills in a fresh session. Claude Code invokes /spec:spec-guide; Codex uses $spec-guide. `spec uninstall --project /absolute/project` removes this setup again.

For a new project, describe your intended experience first. The guide helps establish a brief, requirements and the first useful scope before implementation. Existing instructions, design systems and native tests remain relevant.

The [CLI contract](skills/spec-guide/references/cli.md) covers status, update and rollback. The companion binary's [mechanical checker](skills/spec-guide/references/contracts.md) checks mapped documents and declared evidence freshness. It does not authenticate execution or certify product correctness. A successful native installer does not prove a fresh session loaded the skill.

Reviews run in a fresh context. With `spec install --review egm`, setup also installs Google's official EGM plugin and spec-review routes design reviews to it. Checkpoints use the project checks the agent keeps in `.spec-pilot.json`. This package has no MCP server or factory adapter.
