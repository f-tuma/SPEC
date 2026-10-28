# SPEC development

This repository owns the generic CLI and plugin. Project/product requirements belong to the user's target repository. Do not embed a pilot application's configuration here.

Use Go 1.26+ with pinned Charm dependencies (Bubble Tea, Bubbles, Lip Gloss). The Go binary embeds all plugin resources including hidden compatibility manifests. No Python runtime or wrapper is allowed. Keep JSON CLI operations and the TUI on the same application service.

Keep all product text and documentation in English. Follow DESIGN.md for the terminal dashboard: regular-weight terminal text, human-readable status and plans, a separate action column, visible prerequisites, and a usable narrow-screen view. JSON belongs to noninteractive reports.

Validate changes with `go test ./...`, `go vet ./...`, and `go test -race ./...` where CGO and a C compiler are available. Exercise the built executable outside the source tree, verify its installed hidden manifests and skills, and check the TUI in a real terminal as well as through model updates. Cross-compilation is not native platform execution.

Installer/version-manager changes need an independent skeptical review before their release. Review actual files and failure behavior, and distinguish native integration tests from mocked client calls. Do not claim client loading when those clients were unavailable.

Managed updates target `.spec` plugin versions and their catalogs. Preserve user-authored project documents and OpenSpec configuration. Keep OpenSpec pinned; stage initialization and expose partial client setup accurately. Never automatically migrate/delete legacy global prompts.
