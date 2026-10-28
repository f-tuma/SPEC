package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

func managerFixture(t *testing.T) (*Manager, string, string) {
	t.Helper()
	base := t.TempDir()
	project := filepath.Join(base, "game with spaces")
	if err := os.Mkdir(project, 0755); err != nil {
		t.Fatal(err)
	}
	// SPEC resolves symlinks in the project path (macOS keeps temporary
	// directories under /var -> /private/var), so tests use that form too.
	project, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", filepath.Join(base, "codex-home"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(base, "claude-home"))
	t.Setenv("HOME", filepath.Join(base, "home"))
	return New(os.DirFS("../../plugin")), project, filepath.Join(base, "cache")
}

// jsonText quotes a value for JSON fixtures; Windows paths contain backslashes.
func jsonText(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
func put(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func mustInstall(t *testing.T, m *Manager, project string) {
	t.Helper()
	if _, err := m.installPlugin(project, m.Plugin, nil, false); err != nil {
		t.Fatal(err)
	}
}
func fakeOpenSpec(m *Manager) {
	m.LookPath = func(name string) (string, error) {
		if name == "openspec" {
			return "/fixture/openspec", nil
		}
		return "", exec.ErrNotFound
	}
	m.Run = func(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
		if argv[len(argv)-1] == "--version" {
			return OpenSpecVersion, nil
		}
		if len(argv) < 3 || argv[1] != "init" {
			return "", fmt.Errorf("unexpected fixture call: %v", argv)
		}
		stage := argv[2]
		for _, rel := range []string{"openspec/config.yaml", ".claude/skills/openspec-propose/SKILL.md", ".agents/skills/openspec-propose/SKILL.md"} {
			path := filepath.Join(stage, rel)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return "", err
			}
			if err := os.WriteFile(path, []byte("schema: spec-driven\n"), 0644); err != nil {
				return "", err
			}
		}
		return "initialized by fixture", nil
	}
}
func TestInstallDryRunNoWritesOrExternalCalls(t *testing.T) {
	m, project, cache := managerFixture(t)
	m.Run = func(context.Context, string, []string, ...string) (string, error) {
		t.Fatal("dry-run executed a command")
		return "", nil
	}
	result, code, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, DryRun: true})
	if err != nil || code != 0 || result["status"] != "planned" {
		t.Fatalf("%v %d %v", result, code, err)
	}
	if exists(filepath.Join(project, ".spec")) || exists(cache) {
		t.Fatal("dry-run wrote files")
	}
}
func TestPluginInstallIdempotentAndPreservesProject(t *testing.T) {
	m, project, _ := managerFixture(t)
	put(t, filepath.Join(project, "rules.md"), "owner rules")
	mustInstall(t, m, project)
	mustInstall(t, m, project)
	state, err := LoadState(project)
	if err != nil || len(state.History) != 0 {
		t.Fatalf("%v %v", state, err)
	}
	if _, err := verifyRelease(project, state.Current); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(project, "rules.md"))
	if string(data) != "owner rules" {
		t.Fatal("project changed")
	}
}
func TestUnownedDirectoryAndModifiedPluginArePreserved(t *testing.T) {
	for _, mode := range []string{"unowned", "modified"} {
		t.Run(mode, func(t *testing.T) {
			m, project, _ := managerFixture(t)
			var path string
			if mode == "unowned" {
				path = filepath.Join(project, ".spec/keep")
			} else {
				mustInstall(t, m, project)
				state, _ := LoadState(project)
				base, _ := releasePath(filepath.Join(project, ".spec"), state.Current)
				path = filepath.Join(base, "README.md")
			}
			put(t, path, "user modification")
			if _, err := m.installPlugin(project, m.Plugin, nil, false); err == nil {
				t.Fatal("expected conflict")
			}
			data, _ := os.ReadFile(path)
			if string(data) != "user modification" {
				t.Fatal("modified bytes lost")
			}
		})
	}
}
func TestWithinRejectsUnsafePaths(t *testing.T) {
	_, project, _ := managerFixture(t)
	for _, path := range []string{"../outside", "/absolute", "bad\x00path", "C:\\file", "a/../../b", ".", "a//b"} {
		t.Run(fmt.Sprintf("%q", path), func(t *testing.T) {
			if _, err := Within(project, path); err == nil {
				t.Fatalf("accepted %q", path)
			}
		})
	}
	if err := os.Symlink(filepath.Dir(project), filepath.Join(project, "link")); err == nil {
		if _, err := Within(project, "link/data"); err == nil {
			t.Fatal("followed symlink")
		}
	}
}
func TestMalformedStateReturnsError(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	for _, raw := range []string{`{}`, `{"schema_version":true}`, `{"schema_version":1,"current":{"version":"0.3.0","digest":"bad"},"history":[],"clients":{},"source":"x"}`, `null`} {
		put(t, filepath.Join(project, ".spec/installation.json"), raw)
		if _, err := m.Status(project); err == nil {
			t.Fatal("malformed state accepted")
		}
	}
}
func TestCatalogAndAllHiddenManifests(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	path, _ := releasePath(filepath.Join(project, ".spec"), state.Current)
	for _, rel := range []string{"plugin.json", ".claude-plugin/plugin.json", ".codex-plugin/plugin.json"} {
		if !exists(filepath.Join(path, rel)) {
			t.Fatal("missing embedded compatibility manifest")
		}
	}
	var catalog struct {
		Name    string
		Plugins []struct{ Source struct{ Path string } }
	}
	if err := readJSON(filepath.Join(project, ".spec/marketplace/.agents/plugins/marketplace.json"), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Name != MarketplaceName(project) || !strings.Contains(catalog.Plugins[0].Source.Path, state.Current.Directory) {
		t.Fatal("invalid catalog")
	}
	if MarketplaceName(project) == MarketplaceName(filepath.Dir(project)) {
		t.Fatal("marketplace collision")
	}
}
func TestStatusNeverExecutesMetadata(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	state.OpenSpec = map[string]any{"command": []string{"must-not-execute"}}
	saveState(project, state)
	m.Run = func(context.Context, string, []string, ...string) (string, error) {
		t.Fatal("status executed metadata")
		return "", nil
	}
	if _, err := m.Status(project); err != nil {
		t.Fatal(err)
	}
}
func TestNativeClientRegistrationAndRetry(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	m.LookPath = func(name string) (string, error) { return "/bin/" + name, nil }
	calls := [][]string{}
	fail := true
	m.Run = func(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
		calls = append(calls, argv)
		if fail && !strings.Contains(strings.Join(argv, " "), "marketplace") {
			return "", fmt.Errorf("fixture interrupted installation")
		}
		return "installed", nil
	}
	first, err := m.registerClients(context.Background(), project, []string{"claude"})
	if err != nil || !first["claude"].Registered || first["claude"].Status != "pending" {
		t.Fatalf("%v %v", first, err)
	}
	fail = false
	calls = nil
	second, err := m.registerClients(context.Background(), project, []string{"claude"})
	if err != nil || second["claude"].Status != "installed-by-client" {
		t.Fatalf("%v %v", second, err)
	}
	// The marketplace is registered; the retry refreshes it and installs the plugin.
	if len(calls) != 2 || calls[0][3] != "update" || calls[1][2] != "install" || second["claude"].InvocationVerified {
		t.Fatal("Claude Code retry must refresh the marketplace and install", calls)
	}
	m.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	missing, err := m.registerClients(context.Background(), project, []string{"claude"})
	if err != nil || pendingCode(missing, nil) != 3 {
		t.Fatalf("%v %v", missing, err)
	}
}
func TestStagedOpenSpecPreservesExistingConfigAndWorkflow(t *testing.T) {
	for _, name := range []string{"config.yaml", "config.yml"} {
		t.Run(name, func(t *testing.T) {
			m, project, cache := managerFixture(t)
			fakeOpenSpec(m)
			config := filepath.Join(project, "openspec", name)
			original := "schema: spec-driven\n\"store\": external-owner-store\ncontext: |\n  USER CONTEXT\n"
			put(t, config, original)
			custom := filepath.Join(project, ".claude/skills/openspec-propose/SKILL.md")
			put(t, custom, "custom workflow")
			options := Options{Project: project, Cache: cache, NoRegister: true, Tools: []string{"claude", "codex"}, Method: "auto"}
			result, code, err := m.Execute(context.Background(), "install", options)
			if err != nil || code != 0 {
				t.Fatalf("%v %d %v", result, code, err)
			}
			data, _ := os.ReadFile(config)
			if string(data) != original {
				t.Fatal("config changed")
			}
			data, _ = os.ReadFile(custom)
			if string(data) != "custom workflow" {
				t.Fatal("workflow overwritten")
			}
			if name == "config.yml" && exists(filepath.Join(project, "openspec/config.yaml")) {
				t.Fatal(".yml shadowed")
			}
			if exists(filepath.Join(project, "openspec/specs")) {
				t.Fatal("existing planning shape changed")
			}
			if _, _, err = m.Execute(context.Background(), "install", options); err != nil {
				t.Fatal("repeat setup failed:", err)
			}
		})
	}
}
func TestOpenSpecNewConfigHasBridgeAndPrivateShim(t *testing.T) {
	m, project, cache := managerFixture(t)
	fakeOpenSpec(m)
	_, _, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, NoRegister: true})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(project, "openspec/config.yaml"))
	if !strings.Contains(string(data), "SPEC workflow bridge") {
		t.Fatal("new config missing bridge")
	}
	if !exists(filepath.Join(project, ".spec/bin", shimName())) {
		t.Fatal("missing private shim")
	}
}
func TestLegacyPromptsAndCacheContainmentBlockBeforeWrites(t *testing.T) {
	for _, kind := range []string{"global", "cache", "project"} {
		t.Run(kind, func(t *testing.T) {
			m, project, cache := managerFixture(t)
			switch kind {
			case "global":
				put(t, filepath.Join(os.Getenv("CODEX_HOME"), "prompts/opsx-apply.md"), "legacy")
			case "cache":
				cache = filepath.Join(project, "cache")
			case "project":
				put(t, filepath.Join(project, ".codex/prompts/openspec-apply.md"), "legacy")
			}
			if _, _, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache}); err == nil {
				t.Fatal("missing migration/containment error")
			}
			if exists(filepath.Join(project, ".spec")) {
				t.Fatal("preflight wrote managed state")
			}
		})
	}
}
func TestChecksumMismatchNeverExecutesDownload(t *testing.T) {
	m, _, cache := managerFixture(t)
	m.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	m.Run = func(context.Context, string, []string, ...string) (string, error) {
		t.Fatal("executed unverified bytes")
		return "", nil
	}
	name, _ := binaryName()
	m.Download = func(ctx context.Context, url, target string, max int64) error {
		data := "corrupted binary"
		if strings.HasSuffix(url, "SHA256SUMS") {
			data = strings.Repeat("0", 64) + "  " + name + "\n"
		}
		return os.WriteFile(target, []byte(data), 0600)
	}
	if _, _, err := m.ensureCLI(context.Background(), cache, "binary", ""); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum error: %v", err)
	}
}
func TestNpmBackendPinnedAndPrivate(t *testing.T) {
	m, _, cache := managerFixture(t)
	m.LookPath = func(name string) (string, error) {
		if name == "openspec" {
			return "", exec.ErrNotFound
		}
		return "/bin/" + name, nil
	}
	var install []string
	m.Run = func(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
		if len(argv) > 1 && argv[1] == "install" {
			install = argv
			return "installed", nil
		}
		if len(argv) == 2 && argv[0] == "/bin/node" {
			return "v24.0.0", nil
		}
		return OpenSpecVersion, nil
	}
	_, method, err := m.ensureCLI(context.Background(), cache, "npm", "")
	if err != nil || method != "npm" {
		t.Fatalf("%s %v", method, err)
	}
	args := strings.Join(install, " ")
	if !strings.Contains(args, "@fission-ai/openspec@1.15.0") || !strings.Contains(args, "--ignore-scripts") || strings.Contains(args, " -g ") {
		t.Fatal("uncontrolled npm installation", install)
	}
}
func TestAgentCommandChangesOnlyChildEnvironment(t *testing.T) {
	m, project, cache := managerFixture(t)
	fakeOpenSpec(m)
	if _, _, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, NoRegister: true}); err != nil {
		t.Fatal(err)
	}
	m.LookPath = func(string) (string, error) { return "/bin/claude", nil }
	original := os.Getenv("PATH")
	cmd, err := m.AgentCommand(context.Background(), project, "claude", []string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(cmd.Args, " "), "--plugin-dir") || os.Getenv("PATH") != original {
		t.Fatal("invalid launcher")
	}
	found := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "PATH=") {
			found = strings.HasPrefix(env, "PATH="+filepath.Join(project, ".spec/bin")+string(os.PathListSeparator))
		}
	}
	if !found {
		t.Fatal("private OpenSpec missing from child PATH")
	}
}
func TestOperationLockRefusesConcurrentMutation(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	unlock, err := lockProject(project)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, _, err = m.Execute(context.Background(), "rollback", Options{Project: project}); err == nil || !strings.Contains(err.Error(), "lock") {
		t.Fatalf("expected lock conflict: %v", err)
	}
}
func TestActualGitUpdateAndRollbackPreserveOwnerBytes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git unavailable")
	}
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	repo := filepath.Join(t.TempDir(), "upstream")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		t.Helper()
		out, err := run(context.Background(), repo, nil, append([]string{"git"}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	git("init", "-b", "main")
	for _, version := range []string{"0.4.0", "0.5.0"} {
		fs.WalkDir(m.Plugin, ".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				t.Fatal(err)
			}
			if entry.IsDir() {
				return nil
			}
			data, _ := fs.ReadFile(m.Plugin, path)
			if path == "plugin.json" || path == ".claude-plugin/plugin.json" || path == ".codex-plugin/plugin.json" {
				var info map[string]any
				json.Unmarshal(data, &info)
				info["version"] = version
				data, _ = json.Marshal(info)
			}
			put(t, filepath.Join(repo, "plugin", path), string(data))
			return nil
		})
		git("add", ".")
		git("-c", "user.name=SPEC test", "-c", "user.email=spec@local.invalid", "commit", "-m", version)
		git("tag", "v"+version)
	}
	put(t, filepath.Join(project, "openspec/config.yml"), "owner context")
	put(t, filepath.Join(project, "rules.md"), "owner rules")
	before, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	options := Options{Project: project, Source: repo, Check: true, NoRegister: true}
	checked, _, err := m.Execute(context.Background(), "update", options)
	if err != nil || !reflect.DeepEqual(checked["available"], []string{"v0.4.0", "v0.5.0"}) {
		t.Fatalf("%v %v", checked, err)
	}
	after, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	if string(before) != string(after) {
		t.Fatal("check changed state")
	}
	options.Check = false
	options.To = "v0.5.0"
	result, _, err := m.Execute(context.Background(), "update", options)
	if err != nil || result["plugin_version"] != "0.5.0" {
		t.Fatalf("%v %v", result, err)
	}
	rolled, _, err := m.Execute(context.Background(), "rollback", Options{Project: project, NoRegister: true})
	if err != nil || rolled["plugin_version"] != Version {
		t.Fatalf("%v %v", rolled, err)
	}
	for path, want := range map[string]string{"openspec/config.yml": "owner context", "rules.md": "owner rules"} {
		data, _ := os.ReadFile(filepath.Join(project, path))
		if string(data) != want {
			t.Fatal("owner bytes changed")
		}
	}
	git("tag", "v9.9.9")
	if _, _, err = m.fetchPlugin(context.Background(), repo, "v9.9.9", filepath.Join(t.TempDir(), "clone")); err == nil {
		t.Fatal("mismatched tag accepted")
	}
}
func TestShellQuoteRoundTrip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell quoting")
	}
	want := "a path/'quote';$(false)& space"
	cmd := exec.Command("sh", "-c", "printf '%s' "+shellQuote(want))
	got, err := cmd.Output()
	if err != nil || string(got) != want {
		t.Fatalf("%q %v", got, err)
	}
}

func nextPlugin(t *testing.T, m *Manager) fs.FS {
	t.Helper()
	files := fstest.MapFS{}
	fs.WalkDir(m.Plugin, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if entry.IsDir() {
			return nil
		}
		data, _ := fs.ReadFile(m.Plugin, path)
		if path == "plugin.json" || path == ".claude-plugin/plugin.json" || path == ".codex-plugin/plugin.json" {
			var info map[string]any
			json.Unmarshal(data, &info)
			info["version"] = "0.5.0"
			data, _ = json.Marshal(info)
		}
		files[path] = &fstest.MapFile{Data: data, Mode: 0644}
		return nil
	})
	return files
}
func TestCatalogConflictDoesNotSwitchAnyActiveMetadata(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	before, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	codex := filepath.Join(project, ".spec/marketplace/.agents/plugins/marketplace.json")
	prior, _ := os.ReadFile(codex)
	path := filepath.Join(project, ".spec/marketplace/.claude-plugin/marketplace.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := m.installPlugin(project, nextPlugin(t, m), nil, false); err == nil {
		t.Fatal("directory conflict ignored")
	}
	after, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	current, _ := os.ReadFile(codex)
	if string(before) != string(after) || string(prior) != string(current) {
		t.Fatal("partial catalog switch")
	}
	status, err := m.Status(project)
	if err != nil || status["status"] != "inconsistent" {
		t.Fatalf("catalog problem hidden: %v %v", status, err)
	}
}
func TestLateMetadataFailureRestoresCatalogsAndReceipt(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	before, _ := LoadState(project)
	prior := map[string]string{}
	for _, file := range metadataFiles(project, before) {
		data, _ := os.ReadFile(filepath.Join(project, file.relative))
		prior[file.relative] = string(data)
	}
	failed := false
	m.Rename = func(from, to string) error {
		if strings.HasSuffix(to, "installation.json") && !failed {
			failed = true
			return fmt.Errorf("injected final receipt rename failure")
		}
		return os.Rename(from, to)
	}
	if _, err := m.installPlugin(project, nextPlugin(t, m), nil, false); err == nil {
		t.Fatal("fault not surfaced")
	}
	for path, want := range prior {
		data, _ := os.ReadFile(filepath.Join(project, path))
		if string(data) != want {
			t.Fatalf("metadata changed after failed commit: %s", path)
		}
	}
	status, err := m.Status(project)
	if err != nil || status["status"] != "installed" || status["plugin_version"] != Version {
		t.Fatalf("%v %v", status, err)
	}
}
func TestSkippedClientRefreshIsStale(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	state.Clients["codex"] = Client{Status: "installed-by-client", Registered: true, PluginVersion: state.Current.Version, PluginDigest: state.Current.Digest}
	state.Clients["claude"] = Client{Status: "installed-by-client", Registered: true}
	saveState(project, state)
	if _, err := m.installPlugin(project, nextPlugin(t, m), nil, false); err != nil {
		t.Fatal(err)
	}
	state, _ = LoadState(project)
	client := state.Clients["codex"]
	if client.Status != "stale" || client.PluginVersion != Version || client.PluginDigest == state.Current.Digest {
		t.Fatal("skipped refresh misreported", client)
	}
	legacy := state.Clients["claude"]
	if legacy.Status != "stale" || legacy.PluginDigest != "" || legacy.PluginVersion != "" {
		t.Fatal("legacy unknown client version was invented", legacy)
	}
}
func TestPreviewBaselineRejectsConcurrentStateChange(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	expected := state.Current.Digest
	if _, err := m.installPlugin(project, nextPlugin(t, m), nil, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.Execute(context.Background(), "rollback", Options{Project: project, ExpectedCurrent: expected, NoRegister: true}); err == nil || !strings.Contains(err.Error(), "since preview") {
		t.Fatalf("stale preview accepted: %v", err)
	}
	if _, _, err := m.Execute(context.Background(), "rollback", Options{Project: project, ExpectedTarget: strings.Repeat("0", 64), NoRegister: true}); err == nil || !strings.Contains(err.Error(), "target changed") {
		t.Fatalf("wrong target accepted: %v", err)
	}
}
func TestReinstallRetainsPinnedCLIAndRejectsModifiedShimBeforePluginChange(t *testing.T) {
	m, project, cache := managerFixture(t)
	fakeOpenSpec(m)
	options := Options{Project: project, Cache: cache, Method: "auto", NoRegister: true}
	if _, _, err := m.Execute(context.Background(), "install", options); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.Execute(context.Background(), "install", options); err != nil {
		t.Fatal(err)
	}
	state, _ := LoadState(project)
	if state.OpenSpec["installed_by"] != "managed-existing" {
		t.Fatal("auto changed existing backend")
	}
	shim := filepath.Join(project, ".spec/bin", shimName())
	put(t, shim, "owner modified launcher")
	before, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	m.Plugin = nextPlugin(t, m)
	if _, _, err := m.Execute(context.Background(), "install", options); err == nil {
		t.Fatal("modified launcher overwritten")
	}
	after, _ := os.ReadFile(filepath.Join(project, ".spec/installation.json"))
	if string(before) != string(after) {
		t.Fatal("receipt changed before shim conflict")
	}
}
func TestLegacyWindowsQuoting(t *testing.T) {
	got := legacyWindowsShim([]string{`C:\Program Files\node.exe`, `C:\cache\open spec.js`})
	want := "@echo off\r\n\"C:\\Program Files\\node.exe\" \"C:\\cache\\open spec.js\" %*\r\n"
	if got != want {
		t.Fatalf("%q != %q", got, want)
	}
}

func TestOwnedLauncherCanChangeBackendAndUpgradeLegacyWindowsFormat(t *testing.T) {
	m, project, cache := managerFixture(t)
	fakeOpenSpec(m)
	options := Options{Project: project, Cache: cache, Method: "auto", NoRegister: true}
	if _, _, err := m.Execute(context.Background(), "install", options); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		state, _ := LoadState(project)
		put(t, filepath.Join(project, ".spec/bin", shimName()), legacyWindowsShim(stringList(state.OpenSpec["command"])))
	}
	options.OpenSpecBin = filepath.Join(filepath.Dir(project), "alternate openspec")
	if _, _, err := m.Execute(context.Background(), "install", options); err != nil {
		t.Fatal(err)
	}
	state, _ := LoadState(project)
	command := stringList(state.OpenSpec["command"])
	if len(command) != 1 || command[0] != options.OpenSpecBin {
		t.Fatal("backend selection was not recorded")
	}
	shim, _ := os.ReadFile(filepath.Join(project, ".spec/bin", shimName()))
	content, _ := shimContent(command)
	if string(shim) != content {
		t.Fatal("owned launcher was not updated")
	}
}

func TestLatestReleaseIsQuietAndPicksTheNewestTag(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	t.Setenv("GIT_SSH_COMMAND", "")
	var env, argv []string
	m.Run = func(_ context.Context, _ string, e []string, a ...string) (string, error) {
		env, argv = e, a
		return strings.Repeat("0", 40) + "\trefs/tags/v0.3.0\n" + strings.Repeat("1", 40) + "\trefs/tags/v0.10.0\n" + strings.Repeat("2", 40) + "\trefs/tags/v0.9.0", nil
	}
	current, latest, err := m.LatestRelease(context.Background(), project)
	if err != nil || current != Version || latest != "v0.10.0" {
		t.Fatal(current, latest, err)
	}
	if strings.Join(argv[:4], " ") != "git ls-remote --tags --refs" || strings.Join(env, " ") != "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=10" {
		t.Fatal("check is not a quiet read-only tag listing:", argv, env)
	}
	t.Setenv("GIT_SSH_COMMAND", "ssh -i custom")
	if _, _, err = m.LatestRelease(context.Background(), project); err != nil || len(env) != 0 {
		t.Fatal("an owner-provided GIT_SSH_COMMAND must be kept:", env)
	}
}
