# Native client validation, 2026-10-27

First real check that the SPEC plugin loads and its skills run in fresh Claude Code and Codex sessions. Linux only, one throwaway project.

## Subject and environment

- SPEC 0.3.0 built from the working tree (uncommitted changes on top of `3a4c9aa`), installed as `~/.local/bin/spec`.
- Claude Code 2.1.310 (`claude -p`, stream JSON); Codex CLI 0.164.0 (`codex exec --json`).
- Pilot: a new Git repository with a one-line README, then a tiny Go package (`Notes.Add`) and an OpenSpec change `add-search` whose `design.md` deliberately contradicted its spec (case-sensitive `strings.Contains` against a case-insensitive requirement).
- Real `spec install --project PILOT` with native registration: setup-complete in 1.5 s, OpenSpec 1.15.0 initialized for both tools.

## Results

| Check | Claude Code | Codex |
| --- | --- | --- |
| Plugin registered by `spec install` | `spec@spec-<hash>`, project scope | `spec@spec-<hash>`, user-global |
| Skills offered in a fresh session | `spec:spec-guide`, `spec:spec-review`, `spec:spec-checkpoint` | same names, read from the plugin cache |
| `spec-guide` invoked | Yes, through the Skill tool | Yes, by reading `SKILL.md` |
| `spec-review`: fresh-context reviewer | Subagent; its brief is visible and contained no author diagnosis | `spawn_agent` with `fork_turns: none`; the brief is stored encrypted and could not be audited |
| `spec-review`: planted design defect found | Yes (blocker) | Yes (blocker) |
| `spec-checkpoint`: test calls production code, all pass | Yes | Yes |
| Failing test first and controlled defect | Yes: the design's case-sensitive version failed the new test | Not shown |
| Design corrected to match the spec | Yes, and flagged for owner sign-off | Yes, not flagged |
| Fresh-context review of the final diff | Yes, with six mutants, all caught | Yes, no findings |
| Verification record | `openspec/changes/add-search/verification.md` | same |

Status still reports `invocation_verified: false`; SPEC has no command that records a native check.

## Findings

1. **Codex plugins are user-global.** Codex has no project scope, so every SPEC project adds another enabled `spec@spec-<hash>` plugin. During the run Codex saw two copies of every SPEC skill (the owner's real project and the pilot) and read both. Copies from different projects can be different plugin versions. Codex does load repository skills from `.agents/skills/` (where OpenSpec writes its skills), which could replace global registration.
2. **No unregister path.** SPEC has no uninstall command. Removing the plugin and marketplace through the clients left plugin cache directories behind, and `codex exec` added a `trust_level` entry for the project to `~/.codex/config.toml`.
3. **Codex wrote build caches into `.spec/`** (`go-build`, 67 MB) because its sandbox blocks the user cache. `.spec/` is ignored by Git and status stayed `installed unchanged`, but the managed folder is not meant for tool caches.
4. **Checkpoint discipline differed by host.** The same skill text produced a failing-first test, a controlled defect and an owner decision in Claude Code, but not in Codex. The skill could state those steps more explicitly.
5. **Isolation evidence differs by host.** `codex exec --json` does not report `spawn_agent`; the session log under `~/.codex/sessions` does, but the reviewer brief is encrypted there.

## Follow-up changes

The same day, findings 1 to 4 were addressed in the working tree:

- **Codex skills live in the project.** Setup, update and rollback copy the plugin's `skills/` into `.agents/skills/` and record each file's hash in the receipt; no global Codex plugin is registered and Codex's configuration is not written. A copy counts as SPEC's when it holds the recorded content or any installed release's file; anything else stops the refresh with a pending Codex client. Status names a global plugin from an earlier version (`codex_global_plugin`) and the commands that remove it (`codex_global_cleanup`).
- **`spec uninstall`** removes the Claude Code project plugin and marketplace, an old global Codex plugin, and the SPEC skill copies, keeps edited copies, project documents and OpenSpec files, and then removes `.spec`. A failed client step returns `partial` and keeps the receipt for a retry.
- **`spec-checkpoint`** now asks for a failing test before implementation, a controlled defect for important rules, an explicit owner item for every changed canonical design, spec or task, and keeps caches and scratch copies out of the repository and `.spec/`.

Native recheck after the change (Linux, same client versions): a real `spec install` left Codex's configuration untouched; a fresh `codex exec` listed `spec-guide`, `spec-review` and `spec-checkpoint` from the project and read the project copy; a fresh `claude -p` listed `spec:spec-guide`; an edited copy stopped a repair with exit 3 and was not overwritten; `spec uninstall` removed the setup, kept the edited copy and OpenSpec's skills, and both clients' configuration matched a backup by content. The owner's older project still shows its global Codex plugin in status; it was not changed.

An independent fresh-context review of the installer change ran twice with native client checks in temporary homes. The first round found four should-fix issues (retry blocked by a stale Claude settings entry, unrecorded files after an interrupted refresh, the receipt read before the lock, a symlinked Codex config) and several smaller ones; the second confirmed them resolved and found one more (unlock could remove another operation's lock), which was then fixed. A third check confirmed that fix (the lock race no longer reproduces) and approved the tree; the Windows path comparison compiles but was not run on Windows.

## Codex checkpoint recheck

The planted-defect checkpoint was repeated in Codex with the project skills and the stricter `spec-checkpoint` text. Codex ran the new tests before implementing (they failed because `Search` did not exist yet), broke the case normalization on purpose and showed both case tests fail, restored and reran, listed the corrected design as an item for owner confirmation, and had a `spawn_agent` reviewer with `fork_turns: none` check the diff. Build caches went to a temporary folder inside the repository that it removed afterwards; the prompt had said to work only inside the repository, and Codex recorded that conflict. Its failing-first test failed on compilation rather than behavior, so the skill now asks for a minimal stub first. `codex exec` in workspace-write mode again added a `trust_level` entry for the project, which was removed.

## EGM review, 2026-10-28

Setup gained a review procedure option (`--review spec|egm`). With `egm`, each client's own CLI installs Google's EGM plugin pinned at v0.1.0 (Apache-2.0; SPEC does not copy or redistribute it). Native run on Linux with the same client versions, the built binary and a copy of the planted-defect pilot:

- `spec install --review egm` added the `egm` marketplace and `egm@egm` in both clients: Claude Code with project scope (commit b3a3595), Codex in its user configuration. A second pilot set up afterwards found Codex's plugin already there and recorded it as `present` without running anything.
- A fresh `claude -p` session, asked to review the change's design with the project's procedure, invoked `spec-review`, read `review.mode: egm` from status, read the installed `egm-review` skill (the Skill tool cannot start it), started the `egm:egm-zoom-out` reviewer as a new agent and wrote `openspec/changes/add-search/reviews/zoom_out_report.md`. Only stage 5 was requested. The reviewer noted that the design does not justify case-sensitive matching (major, as a missing decision). The session first called `spec status` without `--project`; the skill now names the flag.
- A fresh `codex exec` session did the same through the project's `spec-review` copy and the installed `egm-review`; its session log shows `spawn_agent` with `fork_turns: "none"` (the brief itself is encrypted). The stage 5 reviewer reported no findings: stage 5 judges simplicity and fit, and without the proposal and specs as Goldfish context it did not compare the design with the requirement.
- Switching back with `spec install --review spec` removed the Claude Code project plugin and named the commands for the shared marketplace and Codex plugin; `spec uninstall` of the second pilot left the Codex plugin it had not added. The test entries, caches and Codex trust entries were then removed, and both clients' configuration matched the backup.

An independent fresh-context review of the installer change, with real client CLIs in temporary homes, found one blocker: Codex's configuration was matched with regular expressions, so an `egm` marketplace written as an inline table was missed and replaced. Codex's config.toml is now parsed with a pinned TOML library. The same round found that an unreadable Claude Code record counted as an installed plugin, that a failed removal was hidden in the TUI, that update and rollback previews left out the review change, that an agent EGM was never set up for was not reported, and that switching back forgot what SPEC had added. All were fixed and a second round confirmed them natively. Its remaining notes (a plugin disabled in Claude Code settings, error wording, a note after `--no-register`) were addressed afterwards.

Not covered: stages 6 to 9 and `egm-resolve`, interactive sessions, Windows and macOS, and Claude Code marketplaces declared only in project settings.

## CI

The GitHub Actions runs for every commit so far never started: the annotation on each job says the account's recent payments failed or its spending limit needs to be increased. Nothing in the workflow or code caused it. Once runs start, a successful push to `main` creates release `v<version>` with the built archives.

## Cleanup

The pilot marketplace and plugin were removed from both clients, the leftover caches and the Codex trust entry were deleted, and the client configuration was compared with a backup taken before the run (only key order differed in Claude's JSON files). The owner's existing SPEC registration for another project was not touched.

## Not covered

Windows and macOS, interactive (non-headless) sessions, plugin update and stale refresh, and a multi-checkpoint project.
