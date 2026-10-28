# SPEC CLI tool

The companion SPEC CLI installs this plugin and OpenSpec and manages plugin versions. Use the actual installed `spec` Go binary. Running it without arguments opens the Charm TUI in a terminal; automation uses explicit subcommands and JSON. Do not invent an executable or checkout path. The CLI requires an explicit project root.

| Task | CLI operation |
| --- | --- |
| Inspect managed setup | `spec status --project ROOT` |
| Preview initial setup | `spec install --project ROOT --dry-run` |
| Install plugin and initialize OpenSpec | `spec install --project ROOT` (`--review egm` adds the EGM plugin) |
| List stable available plugin versions | `spec update --project ROOT --check` |
| Preview a selected release | `spec update --project ROOT --to VERSION --dry-run` |
| Update the managed plugin | `spec update --project ROOT --to VERSION` |
| Restore its previous managed version | `spec rollback --project ROOT` |
| Remove SPEC's agent setup and `.spec` | `spec uninstall --project ROOT` (preview with `--dry-run`) |
| Use the recorded OpenSpec CLI | `spec openspec --project ROOT -- ARGS` |
| List design starters | `spec design --project ROOT` |
| Create DESIGN.md from a starter | `spec design --project ROOT --starter ID --write` |

Status is read-only. In Codex the SPEC skills are project files in `.agents/skills/`; `status` reports their recorded hashes under the Codex client and names an older global Codex plugin (`codex_global_plugin`) that duplicates them in every Codex session. Removing that plugin or uninstalling SPEC changes the user's client setup, so do it only when asked. Version checks may contact the configured Git remote. Install/update/rollback are mutations and should match the requested task and existing authorization. A question about a version does not request an upgrade. Dry-run does not grant authorization for a later write.

The CLI's outputs distinguish tool version, installed plugin version, pinned OpenSpec version, pending client setup and unverified native invocation. Read actual results. Exit 3 means native client setup remains pending; do not report a fully loaded plugin. The first new client session must verify skill availability.

Plugin updates and rollback preserve project intent, OpenSpec specifications and configuration. Local modifications to managed plugin resources stop replacement. A full OpenSpec migration, product deployment or CLI self-update is a separate operation. Do not bypass a detected conflict or install another workflow to make a check green.

## Document language

A project can require one language for its written documents: the brief, OpenSpec proposals, specs, design and tasks, reviews, checkpoints and verification notes. `spec status --project ROOT` reports it as `document_language`. When it is empty, write documents in the user's language. Keep the conversation in the user's language either way, and keep code, identifiers, commands, file names and quoted sources unchanged.

The owner sets it with `spec install --project ROOT --document-language NAME` or in the TUI setup options; `conversation` clears it. A new OpenSpec config receives a `Document language:` line in its context. An existing config is never edited: `config_document_language` reports what it requests. If the two differ, follow `document_language` and tell the user that the config line needs updating. Do not translate existing documents unless asked.

## Review procedure and project checks

`spec status` reports `review.mode`: `spec` for SPEC's own fresh-context review, `egm` for the official [Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model) plugin. The owner chooses it with `--review spec|egm` on install, update or rollback, or in the TUI setup options. With `egm`, setup installs `egm@egm` from Google's repository at a pinned tag: in Claude Code for this project, in Codex for every session (Codex has no project plugins). `review.plugins` reports each agent's EGM setup: `installed` (SPEC added it), `present` (it was already there, including a Claude Code user-scope install), `pending` with a reason (not ready, or a removal that failed), or `kept` (SPEC review is selected again; the record remembers the shared parts SPEC added). Uninstall removes the Claude Code project plugin SPEC added and names the shared marketplace and Codex plugin it leaves.

`project_checks` lists the check commands recorded in `.spec-pilot.json` (`fast`, `full`, `ui`). SPEC only reports them; the agent runs them and keeps them current, as described in [testing](testing.md#project-checks).
