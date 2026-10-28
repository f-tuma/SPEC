---
name: spec-review
description: "Skeptically review a proposed design or final diff against requirements and actual code in a fresh context. Use for SPEC design or checkpoint review and when repository instructions require an independent review."
---

# SPEC: fresh-context review

Read the relevant repository instructions. Identify the exact review subject, revision or file hashes, requirements and review scope. Converse in the user's language; write review records in the project's document language when `spec status --project ROOT` reports `document_language`. The same status reports the project's review procedure as `review.mode`: `spec` uses the procedure below, `egm` routes design reviews to the official EGM plugin (see [EGM review](#egm-review)).

A review runs in a fresh context. Start an independent agent without the conversation and give it the canonical design or final diff, requirements, applicable repository rules and access to actual relevant code. Do not pass the author's conversation, expected findings, preferred answer or your diagnosis. Ask it to review the task, not role-play a test. The reviewer reads the artifact and examines the code itself. Multiple personas in the same conversation are not independent review.

If the host cannot start a fresh-context agent, stop before reviewing: tell the user, and offer a handoff prompt for a fresh session. Perform a self-review only when the user explicitly asks for one in this conversation. Label it as a self-review in the report; it never satisfies a requirement for independent review.

Explain the proposal's behavior and boundaries in your own words. Check consistency with accepted requirements, real files/interfaces, simpler existing solutions, failure and permission boundaries, compatibility and testability according to impact. For a final diff also inspect tests, changed expectations and CI, existing architecture and relevant UI evidence. Do not invent problems to fill a quota.

Report concrete findings with ID, location, evidence, impact and next step; distinguish blockers, decisions and recommendations. Include access limitations and unassessed areas. Use the [review template](../spec-guide/assets/review.md) when a durable report is useful.

Review does not change implementation or product intent. Resolve findings through fixes, evidence-backed dismissal or an owner decision. Re-review materially changed subjects and scope. Explicit approval applies only to the exact reviewed version and review question; it does not prove test execution or authorize publication. See [contracts](../spec-guide/references/contracts.md).

## EGM review

With `review.mode: egm`, the project uses Google's Elephant-Goldfish Model plugin (`egm@egm`), which setup installs for the project's agents or reuses. Use its `egm-review` for design reviews:

- The target is the change's canonical design, normally `openspec/changes/<id>/design.md`, with its proposal and delta specs as Goldfish context. Do not create a parallel `design_draft.md` with `egm-draft` or `egm-brainstorm` for a change OpenSpec already owns; use them only for an idea that has no OpenSpec change yet, and carry the accepted result into OpenSpec.
- Keep EGM's reports next to the change: include the line `EGM artifact directory: openspec/changes/<id>/reviews` when you start the review or ask the user to start it.
- In Claude Code, EGM skills cannot be started through the Skill tool. Read the installed `egm-review` `SKILL.md` and follow it, or ask the user to run `/egm:egm-review`. In Codex, use the installed `egm-review` skill.
- EGM stops when the host cannot start isolated reviewers; report that instead of reviewing in the same context.
- Edits EGM's remediation makes to the design, specs or tasks are changes to canonical sources: list each one for the owner to confirm.

EGM reviews design documents. Review a final diff with the procedure above; `egm-drift-report` can additionally compare implementation with the design. If the EGM plugin is missing from the session, say so and use the procedure above; `spec status` shows whether its setup is pending.
