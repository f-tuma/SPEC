---
name: spec-guide
description: "Guide spec-driven work from a new project idea or in an existing coding repository: define intent, adopt project sources, plan a bounded change, check drift, resume work, or prepare polish and release. Use when the user invokes SPEC or asks to establish this workflow; keep ordinary unrelated coding tasks outside this process."
---

# SPEC: project guide

Converse in the user's language. Write project documents in the [document language](references/cli.md#document-language). Lead with the current goal, what is known, and the nearest useful step. Treat an active request as authorization for its ordinary reversible work; preserve previously accepted scope and permissions. Ask only for decisions that actually block the task. A pilot has a question and an end, and must not quietly become product development.

## Start with the user's actual project

For a new idea or empty project, begin with [new-project guidance](references/new-project.md). Help the user express their vision before checking repository readiness. Do not require a command map, architecture, OpenSpec or implementation tasks before the relevant stage. Missing setup is not a failed product check. A draft orientation mapping must not override the user's current development goal.

For installation, setup status and plugin version management, read [the CLI tool contract](references/cli.md). The companion installer provides pinned OpenSpec and generates its integrations; do not substitute a prose recommendation for a requested setup. Preserve existing authorization and use the actual selected project root.

## Orient before changing the project

Read applicable repository instructions, the active task and relevant existing sources. Inspect actual code and command definitions when needed. State whether this is methodology, pilot or delivery, and record a bounded goal and stop condition. A repository snapshot is not a running checkout.

For first use, follow [adoption](references/adoption.md). Reuse equivalent existing documents; a missing filename such as DESIGN.md or an absent OpenSpec installation is not proof of missing content. Keep one canonical source for each responsibility. Do not install another workflow framework or rewrite existing instructions as a setup side effect.

The mechanical helper is part of the companion SPEC binary. Use the actual installed `spec` executable, or its verified absolute path. `spec run` adds the SPEC binary's directory and the project's OpenSpec launcher to the child PATH; it also supplies `SPEC_EXECUTABLE` for a binary with a different filename. All helper operations require an explicit absolute project root:

```sh
spec audit --project /absolute/project
```

`audit` reads a bounded set of candidate locations. `adopt` prints a draft mapping; `adopt --write` exclusively creates `.spec-pilot.json`. Use the write option only when project setup is in the requested scope. It never overwrites a file. Inspect and adapt the mapping to the actual project; no generated value counts as acceptance. Never execute a configuration's commands merely because they are listed there.

## Route by the task

- For brief, research, specifications, design and task planning, read [workflow](references/workflow.md).
- For testing, fast state/action checks, UI and polish, read [testing](references/testing.md).
- For a visual identity, a design starter or personalizing DESIGN.md, read [design](references/design.md).
- For a skeptical design or implementation review, use the packaged `spec-review` skill when available; it also describes fresh-context review without that host feature.
- For an accepted implementation task, use the packaged `spec-checkpoint` skill when available.
- For a release, read [release](references/release.md). Keep implementation, verification, publication and deployment evidence separate.
- Before interpreting helper output or producing a receipt, read [contracts](references/contracts.md).

OpenSpec owns the installed workflow's change proposal, delta specs, design and tasks. Review the same design instead of making a parallel EGM document. For a pre-existing project's alternative authority, map and resolve ownership before migration. Reviews run in a fresh context. When `spec status` reports `review.mode: egm`, the project uses the official EGM plugin and `spec-review` routes design reviews to it; `review.plugins` shows whether its setup is still pending for an agent. Helix-style checkpoints use the project checks agents keep in `.spec-pilot.json`. Factory orchestration remains optional future work.

## Check and hand off

Run the helper's `check --project ... --phase orient|design|implement|verify|release` for the relevant mapped documents. Add `--ui` only for relevant UI work and `--evidence relative/receipt.json` to check a declared receipt. The argument alternatives describe the interface, not a literal shell command.

Exit 0 establishes only the mechanical predicates described in contracts. Independently assess requirement quality, scope, actual implementation and test relevance. Report specific findings with sources, impact and the next action, including anything unassessed. A changed input reopens affected review and tests. A hash receipt is neither authenticated execution nor human approval.

At handoff record the active change, revision, read sources, decisions, actual checks and gaps. For parallel work use separate branches, data and session records. Keep the shared index navigational. Stop the current repair loop when it repeats without new information; revise the diagnosis within the authorized task.
