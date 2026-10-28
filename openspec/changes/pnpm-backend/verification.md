# Local verification — 2026-10-27

## Checked subject

Working source based on `3a4c9aae9f619073bd35eab88753d7a3bba053dc`, with the pnpm backend change described in [proposal.md](proposal.md). The selected checkout was fast-forwarded from its direct parent; the supplied handoff and local notes were preserved. CI and publication are outside this checkpoint.

The local executable is `bin/spec`. `bin/validation/pnpm-2026-10-27/subject.json` fingerprints all tracked source files plus the new backend tests and proposal, and records the final binary hash. That ignored directory also retains raw Go JSON output and real setup/error reports. These are self-recorded local artifacts, not authenticated evidence. Temporary integration projects and caches are under `/tmp/spec-local-trial` and `/tmp/spec-pnpm-trial`.

## Results

| Check | Result and boundary |
| --- | --- |
| `go test -json ./...` | 184 passing cases including subtests; zero failures |
| `go vet ./...` | Passed |
| `go test -race ./...` | Passed on Linux with CGO/C compiler |
| `git diff --check` | Passed |
| `go build -trimpath -o bin/spec .` | Passed; native Linux amd64 executable |
| `sh scripts/test-install.sh "$PWD/bin/spec"` | Real native Linux prebuilt-binary installation, caller-relative paths and licenses passed |
| Standalone OpenSpec route | Real official binary download, checksum verification, OpenSpec 1.15.0 initialization and launcher execution passed before the pnpm change |
| pnpm route | Real pnpm 12.4.2, Node.js 22.23.4 installation of exact `@fission-ai/openspec@1.15.0`; private package/lockfile/store/cache, real OpenSpec initialization and launcher version check passed |
| Repeated pnpm setup | Passed; existing README, OpenSpec config.yml, project package.json/lockfile and ancestor workspace files retained identical hashes |
| Workspace/hook boundary | Real setup succeeded beneath an ancestor pnpm workspace with a throwing `.pnpmfile.cjs`; the ancestor hook was not executed |
| Wrong cached version | Final executable returned exit 2 with the expected pin, exact affected backend directory and external `--cache-dir` recovery; no package-manager fallback |
| Embedded plugin | Real extracted resources contained all 23 files, three skills and portable/Claude/Codex manifests at 0.3.0 |
| Claude manifest validator | Passed; advisory directory-listing feedback is not native skill-loading evidence |
| Real terminal | Linux PTYs at 80x24 and 44x22; overview, options, pane switching, pnpm label, change-plan preview/cancel and terminal restoration exercised. Model tests also exercise method cycling and the frozen plan |

Backend unit tests call production `ensureCLI` with controlled executable lookup/process boundaries. They cover pinning, private paths, script/hook/workspace flags, cache reuse, missing Node/pnpm, invalid/old Node, installation failure and an unexpected OpenSpec version. These mocked calls are distinct from the real pnpm installation above.

## Independent review

A separate agent with a fresh context inspected the actual proposal, diff, tests and shared setup/failure behavior using `spec-review`. It independently ran Go tests, vet and race checks. Finding R1 requested actionable cache recovery when a cached OpenSpec executable fails its version check. The fix includes the expected version, exact cache directory and a fresh external `--cache-dir` recovery action; the regression test asserts these details.

The reviewer rechecked R1 and reported no remaining blockers in scope. Final reviewed hashes:

```text
d3d23fd63223cbcd37ef645fb5fe7e2ab697445427f73dc411e1d5c0b7b1e650  internal/spec/openspec.go
2736a9b16722353907fd5884569727344996b739afb5f42fc82edb37c9acd4ec  internal/spec/openspec_test.go
```

No implementation edits followed that verdict. The reviewer did not independently perform native installation, terminal journeys or native-agent invocation; those boundaries must not be attributed to the review.

## Ready for the owner's trial

From this checkout, open an existing test project:

```sh
./bin/spec tui --project /absolute/path/to/project --openspec-method pnpm
```

Opening the project only reads status. Choose **Setup options** to pick the OpenSpec source and agents, then **Set up workspace** to inspect and apply a plan. Select only the agent you intend to test. After setup, launch that agent and verify the SPEC skills in a fresh session, then try one bounded task through planning, implementation and verification.

Native registration and actual skill invocation were deliberately not exercised in the local integration fixtures (`--no-register`). No complete external-project workflow trial, native Windows/macOS run, public release or deployment is claimed. Update/rollback against official release tags remains separate from using the bundled plugin locally.
