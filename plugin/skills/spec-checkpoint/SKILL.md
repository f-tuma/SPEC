---
name: spec-checkpoint
description: "Implement and verify one accepted SPEC checkpoint with production-code tests, relevant UI and integration checks, review and revision-bound evidence. Use after a task and its intended behavior are sufficiently defined."
---

# SPEC: implement one checkpoint

Read applicable repository instructions and the active accepted task. Identify requirements, expected observable behavior, actual boundaries and the relevant baseline. Converse in the user's language; write checkpoint records, tasks and verification notes in the project's document language when `spec status` reports `document_language`. Preserve previous authorization and the user's current objective; do not add a fresh approval gate for ordinary reversible work already in scope.

Choose a small reviewable checkpoint. Use the existing canonical OpenSpec change or project equivalent. Apply the risk-scaled [workflow](../spec-guide/references/workflow.md); a trivial fix may need a concise scope and targeted check rather than a document bundle. Missing files with our preferred names are not blockers when equivalent sources exist.

For a regression, reproduce the fault or establish a failing relevant test. For new rules prefer a behavioral test from accepted requirements. Before implementing, run the new or changed test and confirm it fails for the expected reason; record that output. When the code does not exist yet, add a minimal stub first so the test fails on behavior, not on compilation. Implement through actual production boundaries. For an important rule, show the test catches a controlled defect: break the production code temporarily (or in a copy), confirm the test fails, restore it and rerun. Report when the risk does not justify this step. Read [testing](../spec-guide/references/testing.md), then run the project's checks: `fast` after each change and `full` before the checkpoint ends, as recorded in [project checks](../spec-guide/references/testing.md#project-checks). Keep those commands current yourself. Follow with affected integration and actual UI verification. A fixture-backed browser does not prove server/database/provider behavior. JSON is optional for state/action representations.

Compare the result with requirements before code-quality review. Review the final diff in a fresh context with `spec-review`; a self-review counts only when the user explicitly asked for one. Honor repository-specific independent review requirements. Explain changes to tests, CI, snapshots and expectations. Fix concrete findings, repeat affected checks and obtain another review when the reviewed subject materially changes.

Keep build caches, scratch programs and defect copies outside the repository and outside `.spec/`, which SPEC manages; use the system temporary directory or the tool's normal cache.

Record exact commands, results, durations, revision, replaced boundaries and evidence limitations using [verification](../spec-guide/assets/verification.md). The helper described in [contracts](../spec-guide/references/contracts.md) can check declared document paths and hashes, not semantic correctness or authentic execution. Do not turn its exit 0 into a release certificate.

If you change a canonical design, spec or task while implementing (for example to align a design with an accepted requirement), list each change in your report as an item for the owner to confirm. Do not block the checkpoint on it when it only restores accepted intent. If a finding requires different product intent, surface the impact and update the canonical sources through the project's decision process. Repeated unsuccessful repairs without new evidence require a changed diagnosis. Finish at the checkpoint's stop condition, update the session handoff, and report what is actually implemented, verified or unknown. Follow repository git/PR rules. Release is a separate task covered by [release](../spec-guide/references/release.md).
