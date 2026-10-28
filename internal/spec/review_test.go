package spec

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeEGMClients answers EGM's install commands by writing the records the
// real clients keep, and returns the commands it saw.
func fakeEGMClients(t *testing.T, m *Manager, project string) *[]string {
	calls := []string{}
	claude, codex := os.Getenv("CLAUDE_CONFIG_DIR"), filepath.Join(os.Getenv("CODEX_HOME"), "config.toml")
	m.LookPath = func(name string) (string, error) { return "/bin/" + name, nil }
	m.Run = func(_ context.Context, _ string, _ []string, argv ...string) (string, error) {
		line := strings.Join(argv, " ")
		calls = append(calls, line)
		switch line {
		case "/bin/claude plugin marketplace add google/elephant-goldfish-model#v0.1.0":
			put(t, filepath.Join(claude, "plugins/known_marketplaces.json"), `{"egm": {"source": {"source": "github", "repo": "google/elephant-goldfish-model", "ref": "v0.1.0"}}}`)
		case "/bin/claude plugin install egm@egm --scope project":
			put(t, filepath.Join(claude, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"egm@egm": [{"scope": "project", "projectPath": "`+project+`"}]}}`)
		case "/bin/claude plugin uninstall egm@egm --scope project":
			put(t, filepath.Join(claude, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {}}`)
		case "/bin/codex plugin marketplace add google/elephant-goldfish-model --ref v0.1.0":
			put(t, codex, "[marketplaces.egm]\nsource_type = \"git\"\nsource = \"https://github.com/google/elephant-goldfish-model.git\"\nref = \"v0.1.0\"\n")
		case "/bin/codex plugin add egm@egm":
			data, _ := os.ReadFile(codex)
			put(t, codex, string(data)+"\n[plugins.\"egm@egm\"]\nenabled = true\n")
		}
		return "ok", nil
	}
	return &calls
}

func TestEGMReviewInstallsThePinnedPluginAndUninstallKeepsSharedParts(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	if err := setReview(project, "egm"); err != nil {
		t.Fatal(err)
	}
	calls := fakeEGMClients(t, m, project)
	plugins, notes, err := m.setupReview(context.Background(), project, []string{"claude", "codex"})
	if err != nil || len(notes) != 0 {
		t.Fatal(err, notes)
	}
	want := []string{
		"/bin/claude plugin marketplace add google/elephant-goldfish-model#v0.1.0",
		"/bin/claude plugin install egm@egm --scope project",
		"/bin/codex plugin marketplace add google/elephant-goldfish-model --ref v0.1.0",
		"/bin/codex plugin add egm@egm",
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatal(*calls)
	}
	for _, key := range []string{"egm:claude", "egm:codex"} {
		if p := plugins[key]; p.Status != "installed" || !p.Added || !p.MarketplaceAdded || p.Ref != EGMRef {
			t.Fatalf("%s %+v", key, p)
		}
	}
	if code := pendingCode(nil, map[string]any{"plugins": plugins}); code != 0 {
		t.Fatal(code)
	}

	// A repeated setup finds both plugins and runs nothing.
	*calls = nil
	if plugins, _, err = m.setupReview(context.Background(), project, []string{"claude", "codex"}); err != nil || len(*calls) != 0 || plugins["egm:codex"].Status != "installed" {
		t.Fatal(err, *calls, plugins)
	}
	status, _ := m.Status(project)
	review := status["review"].(map[string]any)
	if review["mode"] != "egm" || review["plugins"].(map[string]Extension)["egm:claude"].Status != "installed" {
		t.Fatal(review)
	}
	// Removing EGM from Codex by hand shows up as pending setup.
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "")
	status, _ = m.Status(project)
	if p := status["review"].(map[string]any)["plugins"].(map[string]Extension)["egm:codex"]; p.Status != "pending" || !strings.Contains(p.Reason, "no longer installed") {
		t.Fatalf("%+v", p)
	}

	plan, _, err := m.Uninstall(context.Background(), Options{Project: project, DryRun: true})
	if err != nil || !strings.Contains(strings.Join(plan["actions"].([]string), "\n"), "Remove the Claude Code plugin egm@egm from this project") {
		t.Fatal(plan, err)
	}
	kept := strings.Join(plan["kept"].([]string), "\n")
	if !strings.Contains(kept, "claude plugin marketplace remove egm") || !strings.Contains(kept, "codex plugin remove egm@egm && codex plugin marketplace remove egm") {
		t.Fatal(kept)
	}
	*calls = nil
	result, code, err := m.Uninstall(context.Background(), Options{Project: project})
	if err != nil || code != 0 || result["status"] != "uninstalled" {
		t.Fatal(result, code, err)
	}
	if strings.Join(*calls, "\n") != "/bin/claude plugin uninstall egm@egm --scope project" {
		t.Fatal(*calls)
	}
}

func TestEGMSwitchBackRemovesOnlyWhatSPECAdded(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	setReview(project, "egm")
	calls := fakeEGMClients(t, m, project)
	if _, _, err := m.setupReview(context.Background(), project, []string{"claude", "codex"}); err != nil {
		t.Fatal(err)
	}
	state, _ := LoadState(project)
	if plan := strings.Join(ReviewPlan(project, state, "spec", nil), "\n"); !strings.Contains(plan, "Remove the Claude Code plugin egm@egm") || !strings.Contains(plan, "codex plugin remove egm@egm") {
		t.Fatal(plan)
	}
	if plan := strings.Join(ReviewPlan(project, state, "egm", []string{"claude", "codex"}), "\n"); strings.Count(plan, "already installed") != 2 {
		t.Fatal(plan)
	}
	setReview(project, "spec")
	*calls = nil
	plugins, notes, err := m.setupReview(context.Background(), project, []string{"claude", "codex"})
	if err != nil || plugins["egm:claude"] != (Extension{Status: "kept", Ref: EGMRef, MarketplaceAdded: true}) || plugins["egm:codex"] != (Extension{Status: "kept", Ref: EGMRef, Added: true, MarketplaceAdded: true}) {
		t.Fatal(err, plugins)
	}
	if strings.Join(*calls, "\n") != "/bin/claude plugin uninstall egm@egm --scope project" {
		t.Fatal(*calls)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "codex plugin remove egm@egm") {
		t.Fatal(notes)
	}
	// The kept records only remember what SPEC added: a repeated setup does
	// nothing, and uninstall still names the shared parts.
	*calls = nil
	if _, notes, err = m.setupReview(context.Background(), project, []string{"claude", "codex"}); err != nil || len(*calls) != 0 || len(notes) != 0 {
		t.Fatal(err, *calls, notes)
	}
	if status, _ := m.Status(project); pendingCode(nil, status["review"].(map[string]any)) != 0 {
		t.Fatal(status["review"])
	}
	plan, _, _ := m.Uninstall(context.Background(), Options{Project: project, DryRun: true})
	if kept := strings.Join(plan["kept"].([]string), "\n"); !strings.Contains(kept, "codex plugin remove egm@egm") || !strings.Contains(kept, "claude plugin marketplace remove egm") {
		t.Fatal(kept)
	}
	// Choosing EGM again recognizes SPEC's own marketplace.
	setReview(project, "egm")
	plugins, _, _ = m.setupReview(context.Background(), project, []string{"claude", "codex"})
	if strings.Join(*calls, "\n") != "/bin/claude plugin install egm@egm --scope project" || !plugins["egm:claude"].MarketplaceAdded || plugins["egm:codex"].Status != "installed" {
		t.Fatal(*calls, plugins)
	}
}

func TestEGMAlreadyInstalledByTheUserIsLeftAlone(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	setReview(project, "egm")
	home := os.Getenv("CLAUDE_CONFIG_DIR")
	put(t, filepath.Join(home, "plugins/known_marketplaces.json"), `{"egm": {"source": {"source": "github", "repo": "google/elephant-goldfish-model"}}}`)
	put(t, filepath.Join(home, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"egm@egm": [{"scope": "project", "projectPath": "`+project+`"}]}}`)
	calls := fakeEGMClients(t, m, project)
	plugins, _, err := m.setupReview(context.Background(), project, []string{"claude"})
	if err != nil || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	if p := plugins["egm:claude"]; p.Status != "present" || p.Added || p.MarketplaceAdded || p.Ref != "" {
		t.Fatalf("%+v", p)
	}
	// Neither switching back nor uninstalling removes a plugin SPEC did not add.
	setReview(project, "spec")
	if _, notes, err := m.setupReview(context.Background(), project, []string{"claude"}); err != nil || len(*calls) != 0 || len(notes) != 0 {
		t.Fatal(err, *calls, notes)
	}
	setReview(project, "egm")
	m.setupReview(context.Background(), project, []string{"claude"})
	if result, _, err := m.Uninstall(context.Background(), Options{Project: project}); err != nil || len(*calls) != 0 {
		t.Fatal(result, err, *calls)
	}
}

func TestEGMSetupIsPendingForAForeignMarketplaceOrMissingCLI(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	setReview(project, "egm")
	put(t, filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), "[marketplaces.\"egm\"]\nsource = \"https://example.com/someone/egm.git\"\n")
	calls := fakeEGMClients(t, m, project)
	plugins, _, err := m.setupReview(context.Background(), project, []string{"codex"})
	if err != nil || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	if p := plugins["egm:codex"]; p.Status != "pending" || !strings.Contains(p.Reason, "different marketplace named egm") {
		t.Fatalf("%+v", p)
	}
	if code := pendingCode(nil, map[string]any{"plugins": plugins}); code != 3 {
		t.Fatal(code)
	}
	m.LookPath = func(name string) (string, error) { return "", os.ErrNotExist }
	plugins, _, _ = m.setupReview(context.Background(), project, []string{"claude"})
	if p := plugins["egm:claude"]; p.Status != "pending" || p.Reason != "claude CLI is not installed/on PATH" {
		t.Fatalf("%+v", p)
	}
}

func TestInstallRecordsTheReviewModeAndPreviewsEGM(t *testing.T) {
	m, project, cache := managerFixture(t)
	fakeOpenSpec(m)
	plan, code, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, DryRun: true, Review: "egm", SetReview: true})
	if err != nil || code != 0 {
		t.Fatal(err)
	}
	if review := plan["review"].(map[string]any); review["mode"] != "egm" || len(review["planned"].([]string)) != 2 {
		t.Fatal(review)
	}
	result, code, err := m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, NoRegister: true, Review: "egm", SetReview: true})
	if err != nil || code != 0 || result["review"].(map[string]any)["mode"] != "egm" {
		t.Fatal(result, code, err)
	}
	// Setup without --review keeps the recorded choice.
	result, _, err = m.Execute(context.Background(), "install", Options{Project: project, Cache: cache, NoRegister: true})
	if err != nil || result["review"].(map[string]any)["mode"] != "egm" {
		t.Fatal(result, err)
	}
	if _, err := ReviewMode("goldfish"); err == nil {
		t.Fatal("an unknown review mode must be rejected")
	}
}

func TestStatusReportsTheProjectChecks(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	status, _ := m.Status(project)
	if len(status["project_checks"].(map[string][]string)) != 0 {
		t.Fatal(status["project_checks"])
	}
	put(t, filepath.Join(project, Mapping), `{"schema_version": 1, "mode": "delivery", "goal": "g", "stop_condition": "s", "documents": {}, "checks": {"fast": ["go", "test", "./internal/..."], "full": ["go", "test", "./..."]}}`)
	status, _ = m.Status(project)
	if checks := status["project_checks"].(map[string][]string); len(checks) != 2 || strings.Join(checks["fast"], " ") != "go test ./internal/..." {
		t.Fatal(checks)
	}
	// An invalid mapping reports no checks rather than guessing.
	put(t, filepath.Join(project, Mapping), `{"schema_version": 2, "checks": {"fast": ["x"]}}`)
	status, _ = m.Status(project)
	if len(status["project_checks"].(map[string][]string)) != 0 {
		t.Fatal(status["project_checks"])
	}
}

func TestEGMReadsEveryTOMLFormAndNeverTakesOverUnknownRecords(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	setReview(project, "egm")
	calls := fakeEGMClients(t, m, project)
	codex := filepath.Join(os.Getenv("CODEX_HOME"), "config.toml")
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	for _, test := range []struct {
		name, config, status, reason string
	}{
		{"inline foreign marketplace", "[marketplaces]\negm = { source_type = \"git\", source = \"https://github.com/openai/plugins.git\" }\n", "pending", "different marketplace named egm"},
		{"lookalike repository", "[marketplaces.egm]\nsource = \"https://github.com/google/elephant-goldfish-model-fork.git\"\n", "pending", "different marketplace named egm"},
		{"unparseable", "[marketplaces.egm\n", "pending", "could not be read"},
		{"disabled", "[marketplaces.egm]\nsource = \"google/elephant-goldfish-model\"\n[plugins]\n\"egm@egm\" = { enabled = false }\n", "pending", "disabled"},
		{"inline plugin of the user", "[marketplaces]\negm = { source = \"git@github.com:google/elephant-goldfish-model.git\", ref = \"main\" }\n[plugins]\n\"egm@egm\" = { enabled = true }\n", "present", ""},
	} {
		put(t, codex, test.config)
		*calls = nil
		plugins, _, err := m.setupReview(context.Background(), project, []string{"codex"})
		p := plugins["egm:codex"]
		if err != nil || len(*calls) != 0 || p.Status != test.status || !strings.Contains(p.Reason, test.reason) || p.Added || p.MarketplaceAdded {
			t.Fatalf("%s: %v %v %+v", test.name, err, *calls, p)
		}
	}
	if p, _ := LoadState(project); p.Extensions["egm:codex"].Ref != "main" {
		t.Fatal("the existing marketplace's ref was not reported", p.Extensions)
	}

	// Claude Code: an unreadable install record is unknown, not "installed".
	put(t, filepath.Join(claude, "plugins/known_marketplaces.json"), `{"egm": {"source": {"source": "github", "repo": "google/elephant-goldfish-model"}}}`)
	put(t, filepath.Join(claude, "plugins/installed_plugins.json"), `{"version": 2, "plu`)
	plugins, _, _ := m.setupReview(context.Background(), project, []string{"claude"})
	if p := plugins["egm:claude"]; p.Status != "pending" || !strings.Contains(p.Reason, "installed_plugins.json could not be read") || len(*calls) != 0 {
		t.Fatalf("%+v %v", p, *calls)
	}
	// A user-scope install already serves this project.
	put(t, filepath.Join(claude, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"egm@egm": [{"scope": "user"}]}}`)
	plugins, _, _ = m.setupReview(context.Background(), project, []string{"claude"})
	if p := plugins["egm:claude"]; p.Status != "present" || p.Added || len(*calls) != 0 {
		t.Fatalf("%+v %v", p, *calls)
	}
}

func TestStatusReportsAnAgentEGMWasNeverSetUpFor(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	state, _ := LoadState(project)
	state.Review, state.OpenSpec = "egm", map[string]any{"tools": []any{"claude", "codex"}}
	saveState(project, state)
	status, _ := m.Status(project)
	review := status["review"].(map[string]any)
	if p := review["plugins"].(map[string]Extension)["egm:codex"]; p.Status != "pending" || pendingCode(nil, review) != 3 {
		t.Fatalf("%+v", review)
	}
}

func TestEGMDisabledInClaudeSettingsIsPending(t *testing.T) {
	m, project, _ := managerFixture(t)
	mustInstall(t, m, project)
	setReview(project, "egm")
	calls := fakeEGMClients(t, m, project)
	home := os.Getenv("CLAUDE_CONFIG_DIR")
	put(t, filepath.Join(home, "plugins/known_marketplaces.json"), `{"egm": {"source": {"source": "github", "repo": "google/elephant-goldfish-model"}}}`)
	put(t, filepath.Join(home, "plugins/installed_plugins.json"), `{"version": 2, "plugins": {"egm@egm": [{"scope": "user"}]}}`)
	put(t, filepath.Join(home, "settings.json"), `{"enabledPlugins": {"egm@egm": false}}`)
	plugins, _, _ := m.setupReview(context.Background(), project, []string{"claude"})
	if p := plugins["egm:claude"]; p.Status != "pending" || !strings.Contains(p.Reason, "disable egm@egm") || len(*calls) != 0 {
		t.Fatalf("%+v %v", p, *calls)
	}
	// Project settings override the user's.
	put(t, filepath.Join(project, ".claude/settings.json"), `{"enabledPlugins": {"egm@egm": true}}`)
	if plugins, _, _ = m.setupReview(context.Background(), project, []string{"claude"}); plugins["egm:claude"].Status != "present" {
		t.Fatalf("%+v", plugins["egm:claude"])
	}
}
