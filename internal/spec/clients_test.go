package spec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func noCommands(t *testing.T, m *Manager) {
	m.Run = func(_ context.Context, _ string, _ []string, argv ...string) (string, error) {
		t.Fatalf("unexpected command %v", argv)
		return "", nil
	}
}

func TestCodexSkillsAreCopiedIntoTheProjectWithoutCodex(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	m.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	noCommands(t, m)
	clients, err := m.registerClients(context.Background(), project, []string{"codex"})
	codex := clients["codex"]
	if err != nil || codex.Status != "installed-in-project" || codex.Registered || codex.InvocationVerified {
		t.Fatalf("%+v %v", codex, err)
	}
	for _, skill := range []string{"spec-guide", "spec-review", "spec-checkpoint"} {
		want, _ := os.ReadFile(filepath.Join("../../plugin/skills", skill, "SKILL.md"))
		got, err := os.ReadFile(filepath.Join(project, ".agents/skills", skill, "SKILL.md"))
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s was not copied: %v", skill, err)
		}
		if codex.Files[".agents/skills/"+skill+"/SKILL.md"] != hash(want) {
			t.Fatal("receipt does not record", skill)
		}
	}
	// Relative links between the skills keep working in the copy.
	if !exists(filepath.Join(project, ".agents/skills/spec-guide/assets/review.md")) {
		t.Fatal("spec-review links to spec-guide assets")
	}
	state, _ := LoadState(project)
	if state.Clients["codex"].Status != "installed-in-project" || len(state.Clients["codex"].Files) != len(codex.Files) {
		t.Fatal("receipt not saved")
	}
	if pendingCode(clients, nil) != 0 {
		t.Fatal("project skills need no Codex CLI")
	}
}

func TestCodexSkillRefreshNeverOverwritesLocalEdits(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	noCommands(t, m)
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	edited := filepath.Join(project, ".agents/skills/spec-review/SKILL.md")
	put(t, edited, "my own review rules\n")
	if _, err := m.installPlugin(project, nextPlugin(t, m), nil, false); err != nil {
		t.Fatal(err)
	}
	clients, err := m.registerClients(context.Background(), project, []string{"codex"})
	if err != nil || clients["codex"].Status != "pending" || !strings.Contains(clients["codex"].Reason, "spec-review/SKILL.md has local changes") {
		t.Fatalf("%+v %v", clients["codex"], err)
	}
	if data, _ := os.ReadFile(edited); string(data) != "my own review rules\n" {
		t.Fatal("local edit overwritten")
	}
	if pendingCode(clients, nil) != 3 {
		t.Fatal("blocked refresh must report pending setup")
	}
	// A file SPEC did not write is never taken over either.
	other, otherProject, _ := managerFixture(t)
	mustInstall(t, other, otherProject)
	put(t, filepath.Join(otherProject, ".agents/skills/spec-guide/SKILL.md"), "someone else's skill\n")
	clients, _ = other.registerClients(context.Background(), otherProject, []string{"codex"})
	if clients["codex"].Status != "pending" {
		t.Fatal("unmanaged skill was overwritten")
	}
}

func TestCodexSkillsFollowUpdateAndRollbackAndRemoveObsoleteFiles(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	noCommands(t, m)
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	plugin := nextPlugin(t, m).(fstest.MapFS)
	plugin["skills/spec-guide/references/new.md"] = &fstest.MapFile{Data: []byte("new reference\n"), Mode: 0644}
	if _, err := m.installPlugin(project, plugin, nil, false); err != nil {
		t.Fatal(err)
	}
	state, _ := LoadState(project)
	if state.Clients["codex"].Status != "stale" {
		t.Fatal("plugin change must mark the project skills stale")
	}
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(project, ".agents/skills/spec-guide/references/new.md")) {
		t.Fatal("new file missing")
	}
	// Rollback restores the previous release's skills and removes the new file.
	state, _ = LoadState(project)
	state.OpenSpec = map[string]any{"tools": []any{"codex"}}
	if err := saveState(project, state); err != nil {
		t.Fatal(err)
	}
	result, code, err := m.Execute(context.Background(), "rollback", Options{Project: project, Tools: []string{"codex"}})
	if err != nil || code != 0 || result["clients"].(map[string]Client)["codex"].Status != "installed-in-project" {
		t.Fatalf("%v %d %v", result, code, err)
	}
	if exists(filepath.Join(project, ".agents/skills/spec-guide/references/new.md")) {
		t.Fatal("obsolete file kept after rollback")
	}
}

func TestStatusAndResultsNameAnOldGlobalCodexPlugin(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	status, _ := m.Status(project)
	if status["codex_global_plugin"] != "" {
		t.Fatal("no global plugin expected", status["codex_global_plugin"])
	}
	name := MarketplaceName(project)
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "[marketplaces."+name+"]\nsource_type = \"local\"\n\n[plugins.\"spec@"+name+"\"]\nenabled = true\n")
	status, _ = m.Status(project)
	if status["codex_global_plugin"] != "spec@"+name || status["codex_global_cleanup"] != "codex plugin remove spec@"+name+" && codex plugin marketplace remove "+name {
		t.Fatal(status["codex_global_plugin"], status["codex_global_cleanup"])
	}
	// Another project's registration is not this project's.
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "[plugins.\"spec@spec-0000000000\"]\nenabled = true\n")
	if status, _ = m.Status(project); status["codex_global_plugin"] != "" {
		t.Fatal("matched another project")
	}
}

func TestUninstallRemovesOnlyWhatSPECSetUp(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	noCommands(t, m)
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	name := MarketplaceName(project)
	claudeInstalled(t, project, name)
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "[plugins.\"spec@"+name+"\"]\nenabled = true\n")
	put(t, filepath.Join(project, ".agents/skills/openspec-propose/SKILL.md"), "OpenSpec skill\n")
	put(t, filepath.Join(project, "openspec/config.yaml"), "schema: spec-driven\n")
	edited := filepath.Join(project, ".agents/skills/spec-guide/references/testing.md")
	put(t, edited, "my testing notes\n")

	plan, code, err := m.Uninstall(context.Background(), Options{Project: project, DryRun: true})
	if err != nil || code != 0 || plan["status"] != "planned" || len(plan["actions"].([]string)) != 5 {
		t.Fatalf("%v %d %v", plan, code, err)
	}
	if !exists(filepath.Join(project, ".spec")) {
		t.Fatal("dry-run removed state")
	}

	// A missing client CLI leaves a partial result and keeps the receipt.
	m.LookPath = func(name string) (string, error) {
		if name == "codex" {
			return "", exec.ErrNotFound
		}
		return "/bin/" + name, nil
	}
	calls := []string{}
	m.Run = func(_ context.Context, _ string, _ []string, argv ...string) (string, error) {
		calls = append(calls, strings.Join(argv, " "))
		return "ok", nil
	}
	result, code, err := m.Uninstall(context.Background(), Options{Project: project})
	if err != nil || code != 3 || result["status"] != "partial" || !exists(filepath.Join(project, ".spec/installation.json")) {
		t.Fatalf("%v %d %v", result, code, err)
	}
	if strings.Join(calls, "\n") != "/bin/claude plugin uninstall spec@"+name+" --scope project\n/bin/claude plugin marketplace remove "+name {
		t.Fatal(calls)
	}
	if exists(filepath.Join(project, ".agents/skills/spec-guide/SKILL.md")) {
		t.Fatal("unchanged skill kept")
	}
	if !strings.Contains(strings.Join(result["kept"].([]string), "\n"), "references/testing.md has local changes") {
		t.Fatal("edited copy not reported", result["kept"])
	}
	if state, _ := LoadState(project); state.Clients["codex"].Status != "pending" {
		t.Fatal("a partial uninstall must not report the skills as set up")
	}

	// The retry finishes once Codex is available.
	put(t, filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "plugins/installed_plugins.json"), `{"version": 2, "plugins": {}}`)
	put(t, filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "plugins/known_marketplaces.json"), `{}`)
	m.LookPath = func(name string) (string, error) { return "/bin/" + name, nil }
	calls = nil
	result, code, err = m.Uninstall(context.Background(), Options{Project: project})
	if err != nil || code != 0 || result["status"] != "uninstalled" {
		t.Fatalf("%v %d %v", result, code, err)
	}
	if strings.Join(calls, "\n") != "/bin/codex plugin remove spec@"+name {
		t.Fatal(calls)
	}
	if exists(filepath.Join(project, ".spec")) {
		t.Fatal(".spec kept")
	}
	for _, kept := range []string{edited, filepath.Join(project, ".agents/skills/openspec-propose/SKILL.md"), filepath.Join(project, "openspec/config.yaml")} {
		if !exists(kept) {
			t.Fatal("removed a file SPEC must keep:", kept)
		}
	}
	if _, _, err = m.Uninstall(context.Background(), Options{Project: project}); err == nil {
		t.Fatal("uninstalling twice should report that SPEC is not installed")
	}
}

// claudeInstalled writes Claude Code's own records of a project install.
func claudeInstalled(t *testing.T, project, name string) {
	home := os.Getenv("CLAUDE_CONFIG_DIR")
	put(t, filepath.Join(home, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"spec@`+name+`": [{"scope": "project", "projectPath": `+jsonText(project)+`}]}}`)
	put(t, filepath.Join(home, "plugins/known_marketplaces.json"), `{"`+name+`": {}}`)
}

func TestUninstallFinishesWhenClaudeNoLongerHasThePlugin(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	name := MarketplaceName(project)
	// The project still enables the plugin, but Claude Code installed it for
	// another checkout only: there is nothing to uninstall here.
	put(t, filepath.Join(project, ".claude/settings.json"), `{"enabledPlugins": {"spec@`+name+`": true}}`)
	put(t, filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"spec@`+name+`": [{"scope": "project", "projectPath": "/elsewhere"}]}}`)
	m.LookPath = func(name string) (string, error) { return "/bin/" + name, nil }
	calls := []string{}
	m.Run = func(_ context.Context, _ string, _ []string, argv ...string) (string, error) {
		calls = append(calls, strings.Join(argv, " "))
		return "", nil
	}
	result, code, err := m.Uninstall(context.Background(), Options{Project: project})
	if err != nil || code != 0 || len(calls) != 0 {
		t.Fatalf("%v %d %v %v", result, code, err, calls)
	}
	// An unreadable record still asks Claude Code, which may answer "not found".
	mustInstall(t, m, project)
	put(t, filepath.Join(os.Getenv("CLAUDE_CONFIG_DIR"), "plugins/installed_plugins.json"), `not json`)
	m.Run = func(_ context.Context, _ string, _ []string, argv ...string) (string, error) {
		return "", fmt.Errorf(`claude failed: Plugin "spec@%s" not found in installed plugins`, name)
	}
	if result, code, err = m.Uninstall(context.Background(), Options{Project: project}); err != nil || code != 0 || result["status"] != "uninstalled" {
		t.Fatalf("%v %d %v", result, code, err)
	}
}

func TestInterruptedRefreshIsRecognizedBySetupRollbackAndUninstall(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	noCommands(t, m)
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	plugin := nextPlugin(t, m).(fstest.MapFS)
	plugin["skills/spec-guide/SKILL.md"].Data = append(plugin["skills/spec-guide/SKILL.md"].Data, []byte("\nNew guidance.\n")...)
	plugin["skills/spec-guide/references/new.md"] = &fstest.MapFile{Data: []byte("new reference\n"), Mode: 0644}
	if _, err := m.installPlugin(project, plugin, nil, false); err != nil {
		t.Fatal(err)
	}
	// Simulate a refresh interrupted after writing two new files: the receipt
	// still records the old copies.
	put(t, filepath.Join(project, ".agents/skills/spec-guide/SKILL.md"), string(plugin["skills/spec-guide/SKILL.md"].Data))
	put(t, filepath.Join(project, ".agents/skills/spec-guide/references/new.md"), "new reference\n")
	state, _ := LoadState(project)
	state.OpenSpec = map[string]any{"tools": []any{"codex"}}
	saveState(project, state)
	result, code, err := m.Execute(context.Background(), "rollback", Options{Project: project, Tools: []string{"codex"}})
	if err != nil || code != 0 {
		t.Fatalf("rollback after an interrupted refresh: %v %d %v", result, code, err)
	}
	if exists(filepath.Join(project, ".agents/skills/spec-guide/references/new.md")) {
		t.Fatal("rollback kept the newer release's file")
	}
	// A first setup that failed midway recorded nothing; uninstall still
	// recognizes SPEC's files by the released hashes.
	state, _ = LoadState(project)
	state.Clients["codex"] = Client{Status: "pending"}
	saveState(project, state)
	put(t, filepath.Join(project, ".agents/skills/spec-guide/references/new.md"), "new reference\n")
	put(t, filepath.Join(project, ".agents/skills/spec-mine/SKILL.md"), "my own skill\n")
	result, code, err = m.Uninstall(context.Background(), Options{Project: project})
	if err != nil || code != 0 {
		t.Fatalf("%v %d %v", result, code, err)
	}
	if exists(filepath.Join(project, ".agents/skills/spec-guide")) || exists(filepath.Join(project, ".agents/skills/spec-review")) {
		t.Fatal("uninstall left SPEC files behind")
	}
	if !exists(filepath.Join(project, ".agents/skills/spec-mine/SKILL.md")) {
		t.Fatal("uninstall removed the user's own skill")
	}
}

func TestReceiptCannotNameFilesOutsideProjectSkills(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	put(t, filepath.Join(project, "README.md"), "# Game\n")
	state, _ := LoadState(project)
	state.Clients["codex"] = Client{Status: "installed-in-project", Files: map[string]string{"README.md": hash([]byte("# Game\n"))}}
	saveState(project, state)
	if _, err := LoadState(project); err == nil {
		t.Fatal("receipt with a file outside .agents/skills accepted")
	}
	state.Clients["codex"] = Client{Files: map[string]string{".agents/skills/../../README.md": strings.Repeat("0", 64)}}
	saveState(project, state)
	if _, err := LoadState(project); err == nil {
		t.Fatal("receipt with a traversal accepted")
	}
}

func TestCodexConfigDetectionFollowsLinksAndTOMLQuoting(t *testing.T) {
	_, project, _ := managerFixture(t)
	name := MarketplaceName(project)
	home := os.Getenv("CODEX_HOME")
	real := filepath.Join(t.TempDir(), "dotfiles-config.toml")
	put(t, real, "[ plugins . 'spec@"+name+"' ]\nenabled = true\n")
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(home, "config.toml")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if plugin, marketplace := codexGlobal(name); !plugin || marketplace {
		t.Fatal(plugin, marketplace)
	}
	put(t, real, "[marketplaces.\""+name+"\"]\n")
	if plugin, marketplace := codexGlobal(name); plugin || !marketplace {
		t.Fatal(plugin, marketplace)
	}
}

func TestUnlockLeavesALockItDoesNotOwn(t *testing.T) {
	_, project, _ := managerFixture(t)
	unlock, err := lockProject(project)
	if err != nil {
		t.Fatal(err)
	}
	// Uninstall moved .spec aside and another operation locked a new one.
	path := filepath.Join(project, ".spec/operation.lock")
	put(t, path, "pid=1\ntoken=other\n")
	unlock()
	if !exists(path) {
		t.Fatal("unlock removed another operation's lock")
	}
	os.Remove(path)
	unlock, _ = lockProject(project)
	unlock()
	if exists(path) {
		t.Fatal("unlock kept its own lock")
	}
}

func TestUninstallNotesWhatItCannotCheckOrChange(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	name := MarketplaceName(project)
	put(t, filepath.Join(project, ".claude/settings.json"), `{"enabledPlugins": {"spec@`+name+`": true}}`)
	config := filepath.Join(os.Getenv("CODEX_HOME"), "config.toml")
	put(t, config, "[plugins.\"spec@"+name+"\"]\n")
	if err := os.Chmod(config, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(config, 0644) })
	result, _, err := m.Uninstall(context.Background(), Options{Project: project, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	kept := strings.Join(result["kept"].([]string), "\n")
	if _, readErr := os.ReadFile(config); readErr != nil && !strings.Contains(kept, "config.toml could not be read") {
		t.Fatal("unreadable Codex config not reported:", kept)
	}
	if !strings.Contains(kept, ".claude/settings.json still enables spec@"+name) {
		t.Fatal("leftover project setting not reported:", kept)
	}
}

func TestStatusReportsALegacyCodexSetupWithoutItsPlugin(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	state.Clients["codex"] = Client{Status: "installed-by-client", Registered: true, Materialized: true, PluginVersion: state.Current.Version, PluginDigest: state.Current.Digest}
	saveState(project, state)
	name := MarketplaceName(project)
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "[plugins.\"spec@"+name+"\"]\n")
	if status, _ := m.Status(project); status["clients"].(map[string]Client)["codex"].Status != "installed-by-client" {
		t.Fatal("the global plugin still provides the skills")
	}
	// The owner removed the global plugin before running setup again.
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "")
	codex := func() Client { status, _ := m.Status(project); return status["clients"].(map[string]Client)["codex"] }
	if c := codex(); c.Status != "pending" || !strings.Contains(c.Reason, "run setup again") {
		t.Fatal(c)
	}
	noCommands(t, m)
	if _, err := m.registerClients(context.Background(), project, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	if c := codex(); c.Status != "installed-in-project" {
		t.Fatal(c)
	}
}
