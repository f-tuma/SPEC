# SPEC

SPEC **0.4.0** is a native Go application with a **Bubble Tea v2, Bubbles and Lip Gloss** terminal interface. It prepares coding agents for deliberate work: project intent, OpenSpec changes, bounded checkpoints, skeptical review and verification. The TUI and JSON CLI share one application service.

## Install a prebuilt binary

SPEC is a single executable with the plugin, hidden compatibility manifests and templates embedded. Running it requires neither Go nor Python. Release packages support Linux, macOS and Windows on amd64 and arm64.

The installer selects your platform, verifies SHA-256 and the executable's version, then installs the binary and license notices. Staged replacement preserves or restores previous files on failure.

### Download a published release with one command

Linux and macOS:

```sh
bash -o pipefail -c 'curl -fsSL https://github.com/f-tuma/SPEC/releases/latest/download/install.sh | sh'
```

Windows PowerShell, with the GitHub CLI:

```powershell
$SPEC_INSTALLER = gh release download --repo f-tuma/SPEC --pattern install.ps1 --output -; if ($LASTEXITCODE -ne 0) { throw 'Installer download failed' }; & ([scriptblock]::Create(($SPEC_INSTALLER -join "`n")))
```

The installer uses an authenticated `gh` when available, otherwise HTTPS. Select a transport with `--transport gh|https` or `-Transport gh|https`. It installs the latest stable release by default; `--version` / `-Version` pins a release. Repeat the command to update the CLI binary.

**Online installation needs a published release.** Check [GitHub Releases](https://github.com/f-tuma/SPEC/releases); until the first one appears, build from source. Download errors never trigger a source build or bypass checksum verification.

### From downloaded release files

Download the platform archive, `SHA256SUMS` and the installer for your system from a release into one directory, then run from that directory:

```sh
sh install.sh --from-dir .
```

Windows PowerShell:

```powershell
.\install.ps1 -FromDirectory .
```

The installer detects the version when exactly one archive matches your platform. If several versions are present, select one with `--version v0.4.0` or `-Version v0.4.0`. Nothing is compiled; no Go, Node or Python installation is needed.

The default destination is `~/.local/bin/spec`, or `%LOCALAPPDATA%\SPEC\bin\spec.exe` on Windows. Override it with `--bin-dir DIR` or `-InstallDir DIR`. The installer prints the executable's full path. It leaves PATH and shell profiles unchanged; use that path or add the directory to PATH yourself.

### Build from source

A source build requires Go 1.26+:

```sh
git clone https://github.com/f-tuma/SPEC.git
cd SPEC
CGO_ENABLED=0 go build -trimpath -o bin/spec .
./bin/spec
```

Windows: `go build -trimpath -o bin/spec.exe .`, then `.\bin\spec.exe`. `go install .` produces a command named `SPEC`; the explicit build and release packages use lowercase `spec`.

The optional build wrappers are `scripts/install-from-source.sh` and `scripts/install-from-source.ps1`. Without arguments they build and open the TUI; with `--project DIR` they run project setup. The root `install.sh` / `install.ps1` install a prebuilt binary.

## Workspace dashboard

```sh
spec
spec tui --project /path/to/game
```

A valid project opens directly into a read-only overview. Any existing project directory is supported, including an empty one. Opening it reads local setup, checks whether agent executables are on PATH and discovers candidate project documents. It never installs anything or calls an external tool.

The menu is a sidebar on the left (one column below 72 columns), with the content on the right and the current keys in a help bar at the bottom. Home shows a short status (plugin, recorded OpenSpec setup, agents, candidate documents) and a short menu, the recommended action first: **Set up workspace** or **Repair setup**, **Update to vX.Y.Z** (only when a newer stable release exists), **Project documents**, **Setup options** and **Switch project**. Agents start from your shell with `spec run --project . --agent claude` (or `codex`); rollback is `spec rollback --project DIR`.

After a workspace opens, SPEC quietly lists the release source's stable tags in the background with `git ls-remote`. The check never prompts: Git terminal prompts are off and SSH runs in batch mode unless you set `GIT_SSH_COMMAND` yourself. If it fails or finds nothing newer, no offer appears. Plans, results and errors open as their own screen; Esc returns home.

| Key | Behavior |
| --- | --- |
| Up/Down or j/k | Move in a menu; scroll a plan or report |
| Enter | Select; on a plan, apply it |
| Esc | Back; on a plan, cancel it |
| PgUp/PgDn, Home/End | Scroll a plan or report |
| r | Refresh the workspace status |
| q | Quit from home |
| Ctrl+C | Cancel a running operation and quit after its cleanup |

**Project documents** lists the brief, instructions, design, specs and active OpenSpec changes. Enter reads a document with rendered Markdown (front matter shows as YAML); **e** edits it. **Setup options** chooses the editor, saved for you in your user configuration:

- **Nano keys** (default): type directly; Shift+arrows select text; Ctrl+S saves, Ctrl+X exits (twice to discard changes), Ctrl+K cuts the selection or line, Ctrl+U pastes, Ctrl+W searches, Ctrl+Z/Ctrl+Y undo and redo.
- **Vim keys**: normal, insert and visual modes with `hjkl`, `w b e`, `0 ^ $`, `gg G`, `i a I A o O`, `x dd D yy p P`, `u` and Ctrl+R, `/` with `n N`, and `:w`, `:q`, `:wq`, `:q!`, `:w!`, `:NUMBER`. `v` selects characters and `V` whole lines; in visual mode the same motions extend the selection, and `y d x c p > < o` copy, cut, change, replace, indent, outdent or switch ends. Counts, text objects, block visual mode and named registers are not supported.
- **Your editor**: hands the terminal to `$VISUAL` or `$EDITOR` and returns to the reader afterwards.

Saving refuses to overwrite a file that changed on disk since it was opened (`:w!` overrides in vim keys) and never writes inside `.spec`. Files the built-in editor cannot keep byte-for-byte, for example with tabs or carriage returns, open only in your own editor.

Setup and update always show a plan first; **Enter/y** applies exactly that plan and **Esc/n** cancels it. Setup options are held for this session and affect the next setup; editing them writes no project files. The minimum terminal size is 44 x 22.

The interface uses English, regular-weight terminal text, ASCII controls and colors that adapt to a light or dark terminal background. It inherits the terminal's configured font. No icon font is required.

Recorded versions are not a live execution check. Finding a document does not establish its quality, and successful native plugin installation does not prove skills loaded in a fresh agent session.

## Design starters

SPEC ships six DESIGN.md starters with distinct personalities: **Bold Playful**, **Calm Minimal**, **Cozy Warm**, **Editorial**, **Retro Pixel** and **Technical Dense**. Each follows the [Google DESIGN.md format](https://github.com/google-labs-code/design.md): YAML tokens for colors, typography, corner radii, spacing and components, followed by prose for overview, colors, typography, layout, depth, shapes, components, do's and don'ts, responsive behavior, motion and voice. Every text/background color pair meets WCAG AA contrast. The starters work for web, apps and games; none imitates a real brand.

The design work happens in the conversation with your agent, not in the dashboard. Ask it to shape the project's visual identity (for example "/spec:spec-guide let's design how this game should look"). The spec-guide design interview covers identity, references, the project's world, color, typography, shape, depth, density, components, motion and voice. It uses the starters as complete systems to compare, mix or depart from, then writes `DESIGN.md` with you. The agent and you can also use the CLI directly; it never replaces an existing file:

```sh
spec design --project /path/to/game
spec design --project /path/to/game --starter retro-pixel --write
```

The starters' structure draws on the official examples and on community collections such as the CC0 [free DESIGN.md reference collection](https://github.com/Lucas-vdr-a11y/free-design-md-reference-collection) and [awesome-design-md](https://github.com/VoltAgent/awesome-design-md); their content is original. The official `@google/design.md` linter is not bundled.

## Set up a project and start a conversation

```sh
spec install --project /path/to/game --dry-run
spec install --project /path/to/game
spec run --project /path/to/game --agent claude
# Or use --agent codex
```

Setup writes a versioned plugin into its owned `.spec` directory and creates project-specific catalogs for the selected clients. It prepares **OpenSpec 1.15.0**, runs the real `openspec init` in an isolated cache and adds missing integrations.

`auto` retains a verified OpenSpec from an existing setup or uses a matching executable on PATH. Otherwise it downloads the official standalone binary and verifies SHA-256 before execution. This route requires no Node runtime. For a Node-based installation, choose `--openspec-method pnpm` (or `npm` for compatibility). Both require Node.js >=20.19.0 and the selected package manager on PATH; they install the exact version into a private cache with lifecycle scripts disabled. The pnpm route also disables pnpm hooks and ignores ancestor workspaces. It does not add dependencies or lockfiles to your project. These methods are also available under **OpenSpec source** in the TUI **Setup options**.

```sh
spec install --project /absolute/path/to/project --openspec-method pnpm
```

`--tools claude|codex|claude,codex` selects clients. For Claude Code, setup runs its plugin installer with project scope. Codex has no project scope for plugins, so setup copies the SPEC skills into the project's `.agents/skills/` (next to OpenSpec's own skills), where Codex reads them as repository skills; this needs no Codex CLI and does not touch your Codex configuration. SPEC records each copied file's hash: updates and rollback replace only unchanged copies and stop with a clear message when you edited one. The copies are ordinary project files; commit them if your team should share them. Earlier SPEC versions registered a global Codex plugin; status names it and how to remove it. `--no-register` prepares files without either step. `--cache-dir DIR` selects a cache outside the project; `--openspec-bin FILE` uses an existing executable of the pinned version.

### Document language

`--document-language NAME` (for example `English` or `Czech`) requires one language for the project's written documents: the brief, OpenSpec proposals, specs, design and tasks, reviews and checkpoints. The conversation with the agent stays in your language. Without the option, setup keeps the recorded language; `--document-language conversation` clears it so documents follow the conversation again. In the TUI, choose **Document language** in **Setup options** and type any language name; leave the field empty to follow the conversation.

SPEC records the language in its project state, and `spec status` reports it as `document_language`. The SPEC skills read it before writing documents. A new `openspec/config.yaml` also receives a `Document language:` line in its `context`, which OpenSpec passes to its own artifact instructions. An existing config is never edited: status reports the language it requests as `config_document_language`, and the plan shows the line to add when the two differ.

### Design review and project checks

Every SPEC review runs in a fresh context: the agent starts a reviewer without the conversation. When the agent cannot, it stops and says so; it does a self-review only when you ask for one, and labels it.

`--review egm` uses Google's [Elephant-Goldfish Model](https://github.com/google/elephant-goldfish-model) plugin for design reviews instead of SPEC's own procedure (`--review spec`, the default). Setup asks each agent's CLI to install `egm@egm` from Google's repository at the pinned tag v0.1.0: Claude Code installs it for this project; Codex has no project plugins, so its EGM plugin is enabled in every Codex session. A marketplace or plugin you already have is reused and left as it is. The `spec-review` skill then runs EGM's `egm-review` on the change's OpenSpec `design.md`, keeps EGM's reports in `openspec/changes/<id>/reviews/` and lists any change EGM makes to the design for your confirmation. In the TUI choose **Design review** in **Setup options**. Switching back to SPEC review, or uninstalling, removes the Claude Code project plugin SPEC added; the shared `egm` marketplace and the Codex plugin stay, and the result names the commands that remove them.

Checkpoints follow Helix's small steps: the agent keeps the project's check commands in the `checks` of `.spec-pilot.json` (`fast` after every change, `full` before a checkpoint ends, `ui` for a real UI or integration flow), finds and updates them itself, and records each run with its duration. `spec status` reports them as `project_checks`; SPEC never runs them.

Existing OpenSpec `.yaml` or `.yml` configuration, specifications and modified workflow files are preserved. New configuration receives SPEC's workflow bridge. Legacy layouts or global prompts require separate migration; setup never deletes them automatically.

`spec run` provides the private OpenSpec launcher and SPEC executable to the child agent through PATH. It leaves shell startup files unchanged. Claude also receives the exact managed plugin through `--plugin-dir`; Codex reads the skills from `.agents/skills/`.

In a fresh session, verify **spec-guide**, **spec-review** and **spec-checkpoint**. Invoke `/spec:spec-guide` in Claude Code or `$spec-guide` in Codex. For example:

> We are starting a new game. First I want to describe my intended experience. Help me turn it into a brief, verifiable rules and the first playable scope.

## CLI for agents and version management

```sh
spec --version
spec status --project /path/to/game
spec update --project /path/to/game --check
spec update --project /path/to/game --to v0.4.0 --dry-run
spec update --project /path/to/game --to v0.4.0
spec rollback --project /path/to/game
spec uninstall --project /path/to/game --dry-run
spec uninstall --project /path/to/game
spec openspec --project /path/to/game -- list --json

spec audit --project /path/to/game
spec adopt --project /path/to/game
spec adopt --project /path/to/game --write
spec check --project /path/to/game --phase orient
spec fingerprint --project /path/to/game --input README.md
```

The example tag must actually exist in the source repository. Updates read stable Git tags using existing Git authentication; private repositories require access. `--source` selects another source. The actual source commit is recorded.

`update` and `rollback` manage the plugin and its registration. They preserve product requirements, OpenSpec configuration and specifications. Modified managed files block replacement. Update the CLI itself by replacing its binary or rebuilding; OpenSpec remains separately pinned.

`uninstall` removes what SPEC set up for the agents and then its `.spec` directory: the Claude Code plugin and marketplace for this project, the Claude Code EGM plugin it added, a global Codex plugin from an earlier SPEC version, and the unchanged SPEC skill copies in `.agents/skills/` (edited copies are kept and listed). Project documents, `openspec/` and OpenSpec's own skills and commands stay. When a client CLI is missing, the result is `partial` with the command to run, and `.spec` is kept so you can retry. `--dry-run` lists the steps first. It is a CLI command only; the dashboard does not offer it.

The Go manager reads schema 1 installation state from Python version 0.2 and supports both source layouts. The commands `audit`, `adopt`, `check` and `fingerprint` replace the old Python helper; their `--root` option remains an alias for `--project`.

Noninteractive reports remain JSON for agents and scripts. An interactive agent uses the terminal directly. Without a terminal, an empty invocation prints help. Exit codes: **0** completed steps, **1** mechanical checker findings, **2** error/conflict, **3** workspace prepared with client setup pending.

## Verification and limits

```sh
go test ./...
go vet ./...
go test -race ./...  # Requires CGO and a C compiler
sh scripts/release.sh
```

Tests exercise real files and local Git, configuration preservation, modified-file protection, checksums, TUI journeys, layout bounds and embedded resources. Unit tests use stand-ins for client calls. Native Claude Code/Codex loading, live OpenSpec installation and target-platform execution need separate integration evidence; cross-compilation does not establish those results.

The mechanical checker validates document mappings, simple placeholders and declared file hashes. It does not certify product correctness, authenticate test execution or supply human approval.

Review in a fresh context and fast tests against production logic follow EGM and Helix; the official EGM plugin is optional. JSON is optional for those tests. An existing design authority can replace DESIGN.md. OpenUI, an MCP server and a software factory are outside this version's scope.

See [architecture](docs/architecture.md), [OpenSpec installation](https://openspec.dev/docs/installation), [Charm](https://charm.land/), [Claude CLI](https://code.claude.com/docs/en/plugins/cli-reference) and [Codex CLI](https://learn.chatgpt.com/docs/developer-commands).
