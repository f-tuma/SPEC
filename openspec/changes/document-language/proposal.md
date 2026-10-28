# Required document language

The owner wants a project to require one language for its specifications, even when the conversation uses another language. Add this as one bounded checkpoint.

## Required behavior

- The TUI setup options take the language as typed text; an empty field follows the conversation.
- `spec install` and `spec tui` accept `--document-language NAME`; `conversation` clears the setting. Names are validated as plain language names, so the value cannot inject YAML or prompt text.
- The setting applies to all written project documents: the brief, OpenSpec proposals, specs, design and tasks, reviews, checkpoints and verification notes. The conversation keeps the user's language. Without a setting, documents follow the conversation as before.
- SPEC records the language in its schema 1 receipt as an additive field. Setup without the option keeps the recorded value. `spec status` reports `document_language` and the language an existing OpenSpec config requests as `config_document_language`.
- A new OpenSpec config receives a `Document language:` line in its context. An existing config is never edited; the plan shows the line to add or remove when it differs.
- The SPEC skills read the language through `spec status` and write documents in it. They do not translate existing documents unless asked.
- The TUI shows the language in the status, the sidebar's next-setup summary, setup options and the change plan. Its options start from the recorded language unless the command line chose one.
- This change does not edit existing OpenSpec configs, translate documents, change the pinned OpenSpec version or publish a release.

## Verification and stop condition

Exercise validation, receipt persistence, the new-config line, preservation of existing configs and the mismatch reports in production code. Cover CLI parsing, TUI option cycling, the recorded/command-line precedence and the frozen plan. Run the repository Go checks. Perform a real Linux setup with the pinned OpenSpec binary in an isolated project and confirm that `openspec instructions` includes the line in its project context. Native agent behavior with the updated skills remains unverified until a fresh agent session writes a document.
