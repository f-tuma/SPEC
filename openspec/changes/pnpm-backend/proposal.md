# Private pnpm backend for pinned OpenSpec

The owner prefers pnpm to npm for the optional Node-based OpenSpec setup. Add this as one bounded checkpoint before local user testing.

## Required behavior

- CLI setup and the TUI accept `pnpm` as an OpenSpec installation method and display it accurately in settings and change plans.
- Install exactly `@fission-ai/openspec@1.15.0` in SPEC's external cache. Keep package files, lockfiles and dependencies outside the selected project. Disable lifecycle scripts and pnpm hooks and ignore ancestor workspaces.
- Require an available pnpm and Node.js >=20.19.0. Return an actionable error for missing tools, incompatible Node, failed installation or an unexpected OpenSpec version. Do not silently switch package managers.
- Reuse the existing production initialization, private launcher and project-file preservation logic. Preserve the standalone binary default, npm compatibility and existing verified executable reuse.
- This change does not update OpenSpec, install pnpm globally, register agents as a test side effect or publish a release.

## Verification and stop condition

Exercise production backend selection, arguments, errors, cache reuse, CLI JSON parsing and TUI selection/plan behavior. Run the repository Go checks. Perform a real Linux pnpm setup in an isolated project and verify the resulting OpenSpec version, preserved owner files and usable terminal controls. Obtain a separate-context review of the final diff. Stop with a rebuilt local binary and explicit native-agent/platform limits.
