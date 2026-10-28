package tui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/f-tuma/SPEC/internal/spec"
)

func keyPress(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }
func modelFixture(t *testing.T) model {
	t.Helper()
	project := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	manager := spec.New(os.DirFS("../../plugin"))
	manager.LookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	manager.Run = func(context.Context, string, []string, ...string) (string, error) {
		t.Fatal("read-only TUI journey executed an external command")
		return "", nil
	}
	m := newModel(context.Background(), manager, spec.Options{Project: project, Tools: []string{"claude", "codex"}, Method: "auto", Cache: filepath.Join(t.TempDir(), "cache")})
	t.Cleanup(m.cancel)
	return m
}
func press(m model, code rune) (model, tea.Cmd) {
	updated, cmd := m.Update(keyPress(code))
	return updated.(model), cmd
}
func openFixture(t *testing.T) model {
	t.Helper()
	m := modelFixture(t)
	updated, _ := m.Update(m.perform("status", false, m.options)())
	return updated.(model)
}
func withStatus(m model, status map[string]any, commands map[string]bool) model {
	updated, _ := m.Update(resultMsg{operation: "status", workspace: snapshot{status: status, commands: commands}})
	return updated.(model)
}

// choose moves the home cursor to a menu item and activates it.
func choose(t *testing.T, m model, label string) (model, tea.Cmd) {
	t.Helper()
	for i, it := range m.menu() {
		if it.label == label {
			m.cursor = i
			return press(m, tea.KeyEnter)
		}
	}
	t.Fatalf("menu has no %q: %v", label, menuLabels(m))
	return m, nil
}
func menuLabels(m model) []string {
	labels := []string{}
	for _, it := range m.menu() {
		labels = append(labels, it.label)
	}
	return labels
}
func viewText(m model) string { return ansi.Strip(m.View().Content) }
func requireText(t *testing.T, text string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(text, fragment) {
			t.Fatalf("missing %q in:\n%s", fragment, text)
		}
	}
}
func assertNoProjectWrite(t *testing.T, m model) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(m.options.Project, ".spec")); !os.IsNotExist(err) {
		t.Fatal("read-only journey wrote project state")
	}
}
func TestOpeningProjectGoesDirectlyToReadOnlyOverview(t *testing.T) {
	m := modelFixture(t)
	if m.screen != homeScreen || !m.busy || m.Init() == nil {
		t.Fatal("valid project should automatically schedule status")
	}
	updated, _ := m.Update(m.perform("status", false, m.options)())
	m = updated.(model)
	if m.busy || m.workspace.status["status"] != "not-installed" {
		t.Fatal("overview did not load")
	}
	requireText(t, viewText(m), "Setup needed", "Plugin", "Not installed", "> Set up workspace", "Actions", "Next setup")
	assertNoProjectWrite(t, m)
}

func TestMenuOffersOnlyActionsThatCanRun(t *testing.T) {
	m := openFixture(t)
	if got := menuLabels(m); !reflect.DeepEqual(got, []string{"Set up workspace", "Project documents", "Setup options", "Switch project"}) {
		t.Fatal("new workspace menu:", got)
	}
	ready := installedStatus()
	ready["openspec"] = map[string]any{"version": "1.15.0", "tools": []string{"claude", "codex"}}
	ready["clients"] = map[string]spec.Client{"claude": {Status: "installed-by-client"}, "codex": {Status: "installed-by-client"}}
	ready["previous_versions"] = []string{"0.2.0"}
	m = withStatus(m, ready, map[string]bool{"claude": true, "codex": true})
	if got := menuLabels(m); !reflect.DeepEqual(got, []string{"Project documents", "Repair setup", "Setup options", "Switch project"}) {
		t.Fatal("ready workspace menu (no agents, no rollback, no update without a newer release):", got)
	}
	requireText(t, viewText(m), "Ready", "spec run --project . --agent claude")
	ready["clients"] = map[string]spec.Client{"codex": {Status: "pending", Reason: "codex CLI is not installed"}}
	m = withStatus(m, ready, map[string]bool{"claude": true, "codex": false})
	if got := menuLabels(m); !reflect.DeepEqual(got, []string{"Repair setup", "Project documents", "Setup options", "Switch project"}) {
		t.Fatal("attention menu should lead with repair:", got)
	}
	requireText(t, viewText(m), "Agent CLI missing", "Codex: codex CLI is not installed", "> Repair setup")
	if strings.Contains(viewText(m), "spec run --project") {
		t.Fatal("the agent hint belongs to a ready workspace only")
	}
}

func TestUpdateIsOfferedOnlyWhenANewerReleaseExists(t *testing.T) {
	m := openFixture(t)
	ready := installedStatus()
	m = withStatus(m, ready, nil)
	for _, msg := range []updateMsg{{m.options.Project, "0.3.0", "v0.3.0"}, {m.options.Project, "0.3.0", ""}, {"/other/project", "0.3.0", "v0.9.0"}} {
		updated, _ := m.Update(msg)
		if m = updated.(model); m.workspace.update != "" {
			t.Fatalf("%v must not offer an update", msg)
		}
	}
	updated, _ := m.Update(updateMsg{m.options.Project, "0.3.0", "v0.4.0"})
	m = updated.(model)
	if got := menuLabels(m); got[0] != "Update to v0.4.0" {
		t.Fatal("newer release not offered first:", got)
	}
	requireText(t, viewText(m), "Update", "v0.4.0 available", "> Update to v0.4.0")
	m, cmd := press(m, tea.KeyEnter)
	if !m.busy || cmd == nil {
		t.Fatal("the offer should prepare a plan")
	}
	// A refreshed status keeps nothing stale: the next check decides again.
	m = withStatus(m, ready, nil)
	if m.workspace.update != "" {
		t.Fatal("offer survived a refresh without a new check")
	}
}

func TestQuietUpdateCheckNeverPromptsAndFailsSilently(t *testing.T) {
	m := openFixture(t)
	if _, _, err := m.manager.LatestRelease(context.Background(), m.options.Project); err == nil {
		t.Fatal("a workspace without setup has no release source")
	}
	if msg := m.checkUpdate()(); msg != nil {
		t.Fatal("a failed check must not produce a message:", msg)
	}
}
func TestTerminalBackgroundChangesTheWholePalette(t *testing.T) {
	m := openFixture(t)
	m, _ = choose(t, m, "Project documents")
	updated, _ := m.Update(m.perform("audit", false, m.options)())
	m = updated.(model)
	before := m.viewport.GetContent()
	updated, _ = m.Update(tea.BackgroundColorMsg{Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}})
	m = updated.(model)
	if m.dark || m.viewport.GetContent() == before {
		t.Fatal("light background did not update the report palette")
	}
	updated, _ = m.Update(tea.BackgroundColorMsg{Color: color.RGBA{A: 255}})
	m = updated.(model)
	if !m.dark || m.viewport.GetContent() != before {
		t.Fatal("dark background did not restore the report palette")
	}
}
func TestSwitchProjectValidatesAndCancelKeepsOriginal(t *testing.T) {
	m := openFixture(t)
	original := m.options.Project
	m, _ = choose(t, m, "Switch project")
	requireText(t, viewText(m), "Open a project", "esc cancel")
	m.project.SetValue(filepath.Join(t.TempDir(), "missing"))
	m, cmd := press(m, tea.KeyEnter)
	if cmd != nil || m.screen != projectScreen || !m.messageError {
		t.Fatal("invalid project should remain editable")
	}
	m, _ = press(m, tea.KeyEscape)
	if m.screen != homeScreen || m.options.Project != original || m.project.Value() != original {
		t.Fatal("cancel did not restore original project")
	}
	m, _ = choose(t, m, "Switch project")
	next := t.TempDir()
	m.project.SetValue(next)
	m, cmd = press(m, tea.KeyEnter)
	if cmd == nil || !m.busy || m.options.Project != next || m.workspace.status != nil {
		t.Fatal("switch did not clear old status and schedule a read")
	}
	assertNoProjectWrite(t, m)
}
func TestMissingProjectAsksForPathAndEscQuits(t *testing.T) {
	manager := spec.New(os.DirFS("../../plugin"))
	m := newModel(context.Background(), manager, spec.Options{Project: filepath.Join(t.TempDir(), "missing")})
	t.Cleanup(m.cancel)
	if m.screen != projectScreen || m.busy {
		t.Fatal("missing project should ask for a path")
	}
	requireText(t, viewText(m), "Open a project", "esc quit")
	if _, cmd := press(m, tea.KeyEscape); cmd == nil {
		t.Fatal("Esc without any project should quit")
	}
}
func TestMutationRequiresReadablePlanAndCancelDoesNotWrite(t *testing.T) {
	m := openFixture(t)
	m, cmd := choose(t, m, "Set up workspace")
	if !m.busy || cmd == nil {
		t.Fatal("plan was not scheduled")
	}
	updated, _ := m.Update(m.perform("install", true, m.options)())
	m = updated.(model)
	if m.screen != planScreen || m.pending != "install" {
		t.Fatal("plan not shown")
	}
	requireText(t, viewText(m), "Set up workspace", spec.Version, "1.15.0", "Claude Code, Codex", "enter/y apply", "esc/n cancel")
	if strings.Contains(viewText(m), "baseline_digest") || strings.Contains(viewText(m), `"status":`) {
		t.Fatal("internal JSON leaked into the plan")
	}
	// Setup options are not reachable while a plan waits for confirmation.
	before := m.options
	m, _ = press(m, 's')
	if m.screen != planScreen || !reflect.DeepEqual(before, m.options) {
		t.Fatal("confirmation allowed a settings change")
	}
	m, cmd = press(m, tea.KeyEscape)
	if m.screen != homeScreen || cmd != nil || m.busy {
		t.Fatal("cancel scheduled a mutation or left the plan active")
	}
	requireText(t, viewText(m), "Nothing was changed")
	assertNoProjectWrite(t, m)
	if _, err := os.Stat(m.options.Cache); !os.IsNotExist(err) {
		t.Fatal("plan created the cache")
	}
}
func TestConfirmUsesTheReleaseAndBaselineShownInPlan(t *testing.T) {
	m := openFixture(t)
	options := m.options
	options.NoRegister = true
	updated, _ := m.Update(resultMsg{operation: "update", preview: true, options: options, data: map[string]any{"current": "0.3.0", "selected": "v0.4.0", "source": "owner/release-source", "baseline_digest": "reviewed-baseline", "target_digest": "reviewed-target"}})
	m = updated.(model)
	if m.screen != planScreen || m.pendingOptions.To != "v0.4.0" || m.pendingOptions.Source != "owner/release-source" || m.pendingOptions.ExpectedCurrent != "reviewed-baseline" || m.pendingOptions.ExpectedTarget != "reviewed-target" || !m.pendingOptions.NoRegister {
		t.Fatal("confirmation lost the exact plan options")
	}
	requireText(t, viewText(m), "Update plugin", "v0.4.0", "owner/release-source")
	m, cmd := press(m, 'y')
	if m.screen != homeScreen || !m.busy || cmd == nil {
		t.Fatal("confirmation did not schedule the selected operation")
	}
}
func TestUpdateWithoutNewerReleaseSkipsThePlan(t *testing.T) {
	m := openFixture(t)
	updated, _ := m.Update(resultMsg{operation: "update", preview: true, options: m.options, data: map[string]any{"current": "0.3.0", "selected": "v0.3.0"}})
	m = updated.(model)
	if m.screen != reportScreen || m.pending != "" {
		t.Fatal("an up-to-date plugin should not offer a plan")
	}
	requireText(t, viewText(m), "up to date", "esc back")
	m, _ = press(m, tea.KeyEscape)
	if m.screen != homeScreen {
		t.Fatal("Esc did not return home")
	}
}
func TestBusyCancelWaitsForOperationCleanup(t *testing.T) {
	m := openFixture(t)
	m.busy = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = updated.(model)
	if !m.quitting || cmd != nil || m.ctx.Err() == nil {
		t.Fatal("cancel should wait for cleanup")
	}
	_, cmd = m.Update(resultMsg{operation: "install"})
	if cmd == nil {
		t.Fatal("cleanup completion must quit")
	}
}
func TestSetupOptionsAreVisibleAndDoNotWrite(t *testing.T) {
	m := openFixture(t)
	m, _ = choose(t, m, "Setup options")
	requireText(t, viewText(m), "Setup options", "Claude Code", "Codex", "OpenSpec source", "Native plugin setup", "esc back")
	m, _ = press(m, tea.KeyEnter) // Disable Claude.
	if !reflect.DeepEqual(m.options.Tools, []string{"codex"}) {
		t.Fatal(m.options.Tools)
	}
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyEnter) // Refuse to disable the last agent.
	if !reflect.DeepEqual(m.options.Tools, []string{"codex"}) || !m.messageError {
		t.Fatal("empty agent selection was allowed")
	}
	requireText(t, viewText(m), "Keep at least one agent selected")
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyEnter)
	if m.options.Method != "binary" {
		t.Fatal("OpenSpec option was not changed")
	}
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyEnter)
	if !m.options.NoRegister {
		t.Fatal("native setup option was not changed")
	}
	m, _ = press(m, tea.KeyEscape)
	if m.screen != homeScreen {
		t.Fatal("options did not close")
	}
	assertNoProjectWrite(t, m)
}

func TestPnpmSelectionIsVisibleInOptionsAndFrozenPlan(t *testing.T) {
	m := openFixture(t)
	m, _ = choose(t, m, "Setup options")
	m, _ = press(m, tea.KeyDown)
	m, _ = press(m, tea.KeyDown)
	for _, want := range []string{"binary", "pnpm"} {
		m, _ = press(m, tea.KeyEnter)
		if m.options.Method != want {
			t.Fatalf("selected %q, want %q", m.options.Method, want)
		}
	}
	requireText(t, viewText(m), "pnpm (Node.js required)")
	m, _ = press(m, tea.KeyEscape)
	m, _ = choose(t, m, "Set up workspace")
	updated, _ := m.Update(m.perform("install", true, m.options)())
	m = updated.(model)
	if m.screen != planScreen || m.pendingOptions.Method != "pnpm" {
		t.Fatal("plan lost the selected pnpm method")
	}
	requireText(t, viewText(m), "pnpm (Node.js required)")
	m, _ = press(m, 'n')
	assertNoProjectWrite(t, m)
	m, _ = choose(t, m, "Setup options")
	for _, want := range []string{"npm", "auto"} {
		m, _ = press(m, tea.KeyEnter)
		if m.options.Method != want {
			t.Fatalf("selected %q, want %q", m.options.Method, want)
		}
	}
	assertNoProjectWrite(t, m)
}
func TestReportsScrollAndEscReturnsHome(t *testing.T) {
	m := openFixture(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 22})
	m = updated.(model)
	docs := map[string][]string{"brief": {"docs/project-brief.md"}}
	for i := range 12 {
		docs["specification"] = append(docs["specification"], fmt.Sprintf("openspec/specs/s%d/spec.md", i))
	}
	updated, _ = m.Update(resultMsg{operation: "audit", data: map[string]any{"documents": docs}})
	m = updated.(model)
	requireText(t, viewText(m), "Project documents", "Lines 1-", "esc back")
	m, _ = press(m, tea.KeyPgDown)
	if m.viewport.YOffset() == 0 {
		t.Fatal("paging failed to scroll")
	}
	m, _ = press(m, tea.KeyEnd)
	requireText(t, viewText(m), "not been reviewed")
	m, _ = press(m, tea.KeyEscape)
	if m.screen != homeScreen {
		t.Fatal("Esc did not return home")
	}
	requireText(t, viewText(m), "> Set up workspace")
}
func TestFailedMutationKeepsPartialWorkspaceStateVisible(t *testing.T) {
	m := openFixture(t)
	partial := snapshot{status: map[string]any{"status": "installed", "plugin_version": "0.3.0", "managed_files": "unchanged", "catalogs": "unchanged"}}
	updated, _ := m.Update(resultMsg{operation: "install", err: errors.New("OpenSpec download failed"), workspace: partial})
	m = updated.(model)
	requireText(t, viewText(m), "Could not complete workspace setup", "OpenSpec download failed")
	m, _ = press(m, tea.KeyEscape)
	if m.workspace.status["plugin_version"] != "0.3.0" {
		t.Fatal("partial state was lost")
	}
	requireText(t, viewText(m), "OpenSpec setup needed", "v0.3.0", "> Repair setup")
}
func TestEveryScreenFitsSupportedTerminalSizes(t *testing.T) {
	sizes := []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 120, Height: 40}, {Width: 44, Height: 22}, {Width: 79, Height: 24}, {Width: 80, Height: 22}, {Width: 20, Height: 6}, {Width: 10, Height: 2}}
	for _, size := range sizes {
		for _, screen := range []string{"home", "busy", "attention", "options", "language", "docs", "view", "nano", "vim", "project", "plan", "error", "files"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size.Width, size.Height, screen), func(t *testing.T) {
				m := openFixture(t)
				updated, _ := m.Update(size)
				m = updated.(model)
				realProject := m.options.Project
				m.options.Project = "/a/very/long/path/with\nnewline/" + strings.Repeat("project-", 30)
				m.message = "A diagnostic\nwith multiple lines"
				switch screen {
				case "busy":
					m.busy = true
				case "attention":
					d := installedStatus()
					d["managed_files"] = strings.Repeat("plugin/README.md was modified ", 20)
					d["catalogs"] = strings.Repeat("catalog drift ", 20)
					d["previous_versions"] = []string{"0.2.0"}
					d["clients"] = map[string]spec.Client{"claude": {Status: "pending", Reason: strings.Repeat("reason ", 30)}, "codex": {Status: "stale", Reason: "stale"}}
					m.workspace = snapshot{status: d, commands: map[string]bool{"claude": true, "codex": true}}
				case "options":
					m.screen = optionsScreen
				case "language":
					m.screen, m.optionCursor = optionsScreen, languageOption
					m.language.Focus()
					m.language.SetValue("Brazilian Portuguese (pt-BR)")
				case "docs", "view", "nano", "vim":
					m.options.Project = realProject
					m = withDocument(t, m, "DESIGN.md", "# Title\n\n"+strings.Repeat("A long line of design prose. ", 40)+"\n\n- item\n")
					if screen == "docs" {
						m.screen = docsScreen
					}
					if screen == "nano" || screen == "vim" {
						m.prefs.Editor = screen
						m, _ = press(m, 'e')
						if m.screen != editScreen {
							t.Fatal("editor did not open:", m.message)
						}
					}
					m.options.Project = "/a/very/long/path/with\nnewline/" + strings.Repeat("project-", 30)
				case "project":
					m.screen = projectScreen
					m.project.Focus()
				case "plan":
					m.screen, m.report = planScreen, previewReport("install", map[string]any{}, m.options)
				case "error":
					m.screen, m.report = reportScreen, errorReport("install", errors.New(strings.Repeat("unbroken-diagnostic", 80)))
				case "files":
					m.screen, m.report = reportScreen, auditReport(map[string]any{"documents": map[string][]string{"brief": {strings.Repeat("long-file-", 50) + ".md"}}})
				}
				m.refreshContent(true)
				view := m.View().Content
				lines := strings.Split(view, "\n")
				if len(lines) > size.Height {
					t.Fatalf("%d lines exceed %d:\n%s", len(lines), size.Height, ansi.Strip(view))
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > size.Width {
						t.Fatalf("line exceeds width %d:\n%s", size.Width, ansi.Strip(line))
					}
				}
				for _, r := range ansi.Strip(view) {
					if r > 127 {
						t.Fatalf("built-in UI needs a non-ASCII glyph: %q", r)
					}
				}
				// Key hints stay on the last line of every supported screen.
				if size.Width >= 44 && size.Height >= 22 && !m.busy && !strings.Contains(ansi.Strip(lines[len(lines)-1]), "esc") && !strings.Contains(ansi.Strip(lines[len(lines)-1]), "q quit") && !strings.Contains(ansi.Strip(lines[len(lines)-1]), "^X exit") && !strings.Contains(ansi.Strip(lines[len(lines)-1]), ":q quit") {
					t.Fatalf("key hints are not visible:\n%s", ansi.Strip(view))
				}
			})
		}
	}
}

func TestDocumentLanguageStartsFromReceiptAndIsFrozenInPlan(t *testing.T) {
	m := openFixture(t)
	recorded := installedStatus()
	recorded["document_language"] = "Czech"
	recorded["config_document_language"] = "English"
	m = withStatus(m, recorded, nil)
	if m.options.DocumentLanguage != "Czech" || !m.options.SetDocumentLanguage {
		t.Fatal("options did not start from the recorded language:", m.options)
	}
	requireText(t, viewText(m), "Language", "Czech", "Docs: Czech", "requests English; SPEC uses Czech")
	m, _ = choose(t, m, "Setup options")
	for range 4 {
		m, _ = press(m, tea.KeyDown)
	}
	requireText(t, viewText(m), "Document language", "Czech", "Type any", "leave it empty")
	// Enter opens an inline field with the current value; esc keeps it.
	m, _ = press(m, tea.KeyEnter)
	if !m.language.Focused() || m.language.Value() != "Czech" {
		t.Fatal("language field did not open with the current value:", m.language.Value())
	}
	requireText(t, viewText(m), "enter save", "esc cancel")
	m = typeText(m, " q")
	m, _ = press(m, tea.KeyEscape)
	if m.language.Focused() || m.options.DocumentLanguage != "Czech" || m.screen != optionsScreen {
		t.Fatal("esc should cancel the edit and stay in options:", m.options.DocumentLanguage)
	}
	// Clearing the field follows the conversation.
	m, _ = press(m, tea.KeyEnter)
	for range len("Czech") {
		m, _ = press(m, tea.KeyBackspace)
	}
	m, _ = press(m, tea.KeyEnter)
	if m.language.Focused() || m.options.DocumentLanguage != "" {
		t.Fatal("an empty field should follow the conversation:", m.options.DocumentLanguage)
	}
	requireText(t, viewText(m), "Follows conversation")
	// An invalid name is refused and the field stays open.
	m, _ = press(m, tea.KeyEnter)
	m = typeText(m, "en; rm")
	m, _ = press(m, tea.KeyEnter)
	if !m.language.Focused() || m.options.DocumentLanguage != "" || !m.messageError {
		t.Fatal("an invalid language was accepted:", m.options.DocumentLanguage)
	}
	m.language.SetValue("")
	m = typeText(m, "  English ")
	m, _ = press(m, tea.KeyEnter)
	if m.options.DocumentLanguage != "English" {
		t.Fatal(m.options.DocumentLanguage)
	}
	m, _ = press(m, tea.KeyEscape)
	// A refreshed status no longer replaces a language chosen in this session.
	m = withStatus(m, recorded, nil)
	if m.options.DocumentLanguage != "English" {
		t.Fatal("refresh discarded the chosen language")
	}
	updated, _ := m.Update(resultMsg{operation: "install", preview: true, options: m.options, data: map[string]any{"openspec": map[string]any{"document_language": "English", "config_language_line": spec.LanguageLine("English")}}})
	m = updated.(model)
	if m.pendingOptions.DocumentLanguage != "English" || !m.pendingOptions.SetDocumentLanguage {
		t.Fatal("plan lost the language")
	}
	requireText(t, viewText(m), "Language", "English", "existing config is not edited", "Document language: English.")
}

func TestCommandLineLanguageOverridesReceipt(t *testing.T) {
	manager := spec.New(os.DirFS("../../plugin"))
	m := newModel(context.Background(), manager, spec.Options{Project: t.TempDir(), DocumentLanguage: "German", SetDocumentLanguage: true})
	t.Cleanup(m.cancel)
	recorded := installedStatus()
	recorded["document_language"] = "Czech"
	m = withStatus(m, recorded, nil)
	if m.options.DocumentLanguage != "German" {
		t.Fatal("receipt replaced the command-line language")
	}
}

func TestDesignReviewStartsFromReceiptAndShowsEGMSetup(t *testing.T) {
	m := openFixture(t)
	status := installedStatus()
	status["review"] = map[string]any{"mode": "egm", "plugins": map[string]spec.Extension{}}
	m = withStatus(m, status, map[string]bool{"claude": true, "codex": true})
	if m.options.Review != "egm" || !m.options.SetReview {
		t.Fatalf("%q %v", m.options.Review, m.options.SetReview)
	}
	requireText(t, viewText(m), "Review", "EGM; setup pending", "Checks", "Not recorded", "Review: EGM")
	m, _ = choose(t, m, "Setup options")
	for range 5 {
		m, _ = press(m, tea.KeyDown)
	}
	requireText(t, viewText(m), "Design review", "EGM plugin", "for every session in Codex")
	m, _ = press(m, tea.KeyEnter)
	if m.options.Review != "spec" {
		t.Fatal(m.options.Review)
	}
	// A later status refresh keeps the owner's choice.
	m = withStatus(m, status, map[string]bool{"claude": true, "codex": true})
	if m.options.Review != "spec" {
		t.Fatal("status overwrote the chosen review mode")
	}
	assertNoProjectWrite(t, m)
}
