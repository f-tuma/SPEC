package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/f-tuma/SPEC/internal/spec"
)

func installedStatus() map[string]any {
	return map[string]any{"status": "installed", "plugin_version": "0.3.0", "managed_files": "unchanged", "catalogs": "unchanged", "openspec": map[string]any{"version": "1.15.0"}, "clients": map[string]spec.Client{}}
}
func TestOverviewDoesNotEquateInstallationWithReadiness(t *testing.T) {
	for _, test := range []struct {
		name, expected string
		change         func(map[string]any)
	}{
		{"prepared", "Ready", func(d map[string]any) {}},
		{"pending", "Agent setup needs attention", func(d map[string]any) {
			d["clients"] = map[string]spec.Client{"claude": {Status: "pending", Reason: "CLI not installed"}}
		}},
		{"stale", "Agent setup needs attention", func(d map[string]any) {
			d["clients"] = map[string]spec.Client{"codex": {Status: "stale", Reason: "refresh required"}}
		}},
		{"legacy", "Agent setup needs attention", func(d map[string]any) { d["clients"] = map[string]spec.Client{"codex": {Status: "version-unverified"}} }},
		{"modified", "Managed files need attention", func(d map[string]any) { d["managed_files"] = "plugin/README.md was modified" }},
		{"catalog", "Workspace metadata needs attention", func(d map[string]any) { d["catalogs"] = "marketplace does not match receipt" }},
		{"partial", "OpenSpec setup needed", func(d map[string]any) { d["openspec"] = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := installedStatus()
			test.change(d)
			r := overviewReport(snapshot{status: d})
			if r.summary != test.expected {
				t.Fatalf("%q, want %q", r.summary, test.expected)
			}
			// Versions are recorded state; an installed client is not a proof of loaded skills.
			text := ansi.Strip(r.render(70, model{dark: true}.palette()))
			requireText(t, text, "OpenSpec", "Documents")
			if d["openspec"] != nil {
				requireText(t, text, "1.15.0 (recorded)")
			}
		})
	}
}
func TestReadableReportsCoverAllOperationsAndPendingClients(t *testing.T) {
	pending := map[string]spec.Client{"codex": {Status: "pending", Reason: "codex CLI is not installed/on PATH"}}
	for _, test := range []struct {
		operation string
		data      map[string]any
		code      int
		want      []string
	}{
		{"install", map[string]any{"plugin": map[string]any{"plugin_version": "0.3.0"}, "clients": pending}, 3, []string{"agent setup is pending", "Setup pending", "codex CLI is not installed/on PATH", "Repair setup"}},
		{"install", map[string]any{"plugin": map[string]any{"plugin_version": "0.3.0"}, "clients": map[string]spec.Client{"claude": {Status: "installed-by-client"}}}, 0, []string{"Installed; verify skills", "check that the SPEC skills load"}},
		{"update", map[string]any{"plugin_version": "0.4.0"}, 0, []string{"0.4.0"}},
		{"rollback", map[string]any{"plugin_version": "0.2.0"}, 0, []string{"0.2.0"}},
		{"update", map[string]any{"status": "already-newer", "current": "0.10.0"}, 0, []string{"newer than the source release"}},
		{"audit", map[string]any{"documents": map[string][]string{"brief": {"docs/project-brief.md"}}}, 0, []string{"1 candidate file", "Project brief", "docs/project-brief.md", "Not found", "behavior specifications", "have not been reviewed"}},
	} {
		t.Run(test.operation+test.want[0], func(t *testing.T) {
			r := operationReport(test.operation, test.data, test.code)
			text := r.title + "\n" + ansi.Strip(r.render(90, model{dark: true}.palette()))
			requireText(t, text, test.want...)
			if strings.Contains(text, `"status":`) || strings.Contains(text, "map[") || strings.Contains(text, "<nil>") {
				t.Fatal("raw CLI data leaked into a report")
			}
		})
	}
}

func TestOverviewNamesTheFirstUnmetPrerequisite(t *testing.T) {
	d := installedStatus()
	d["openspec"] = map[string]any{"version": "1.15.0", "tools": []string{"codex"}}
	d["clients"] = map[string]spec.Client{"codex": {Status: "pending"}}
	w := snapshot{status: d, commands: map[string]bool{"codex": false}}
	r := overviewReport(w)
	if r.summary != "Agent CLI missing" || !strings.Contains(r.next, "Install missing agent CLIs") {
		t.Fatal("retrying plugin setup cannot install the agent CLI:", r)
	}
	d["openspec"] = nil
	if r := overviewReport(w); r.summary != "OpenSpec setup needed" {
		t.Fatal("agent setup hid incomplete OpenSpec:", r)
	}
}
func TestExternalDiagnosticsCannotChangeTerminalStyle(t *testing.T) {
	diagnostic := "\x1b[1mBOLD\x1b[0m\x1b]0;unwanted-title\a\r\t\x00error\nnext line"
	r := errorReport("install", errors.New(diagnostic))
	text := r.render(60, model{dark: true}.palette())
	plain := ansi.Strip(text)
	requireText(t, plain, "BOLD error", "next line")
	if strings.Contains(text, "\x1b[1m") || strings.Contains(text, "unwanted-title") || strings.Contains(text, "\r") || strings.Contains(text, "\x00") {
		t.Fatal("external controls or font weight leaked")
	}
}

func TestPlanExplainsConfigLanguageItCannotEdit(t *testing.T) {
	r := previewReport("install", map[string]any{"openspec": map[string]any{"document_language": "", "config_document_language": "Czech"}}, spec.Options{})
	text := ansi.Strip(r.render(90, model{dark: true}.palette()))
	requireText(t, text, "Follows conversation", "still requests Czech", "Remove its Document language")
}

func TestCodexProjectSkillsAreReadyAndAnOldGlobalPluginIsNamed(t *testing.T) {
	d := installedStatus()
	d["clients"] = map[string]spec.Client{"codex": {Status: "installed-in-project"}}
	r := overviewReport(snapshot{status: d})
	text := ansi.Strip(r.render(70, model{dark: true}.palette()))
	if r.summary != "Ready" {
		t.Fatal(r.summary)
	}
	requireText(t, text, "Skills in project; verify")
	d["codex_global_plugin"] = "spec@spec-0123456789"
	d["codex_global_cleanup"] = "codex plugin remove spec@spec-0123456789 && codex plugin marketplace remove spec-0123456789"
	text = ansi.Strip(overviewReport(snapshot{status: d}).render(70, model{dark: true}.palette()))
	requireText(t, text, "enabled spec@spec-0123456789 in every Codex", "codex plugin remove spec@spec-0123456789", "marketplace remove", "spec-0123456789.")
	result := operationReport("install", map[string]any{"clients": map[string]spec.Client{"codex": {Status: "installed-in-project"}}, "codex_global_plugin": "spec@spec-0123456789"}, 0)
	requireText(t, ansi.Strip(result.render(70, model{dark: true}.palette())), "Skills in project; verify", "earlier setup enabled")
}

func TestReportsShowTheReviewProcedureAndChecks(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir())
	d := installedStatus()
	d["review"] = map[string]any{"mode": "egm", "plugins": map[string]spec.Extension{"egm:claude": {Status: "installed", Ref: "v0.1.0"}, "egm:codex": {Status: "pending", Reason: "codex CLI is not installed/on PATH"}}}
	d["project_checks"] = map[string][]string{"full": {"go", "test", "./..."}, "fast": {"go", "vet"}}
	r := overviewReport(snapshot{status: d})
	text := ansi.Strip(r.render(80, model{dark: true}.palette()))
	requireText(t, text, "EGM; setup pending", "EGM in Codex: codex CLI is not installed/on PATH", "fast, full")
	if r.summary != "Agent setup needs attention" {
		t.Fatal(r.summary)
	}
	d["review"] = map[string]any{"mode": "egm", "plugins": map[string]spec.Extension{"egm:claude": {Status: "installed", Ref: "v0.1.0"}}}
	requireText(t, ansi.Strip(overviewReport(snapshot{status: d}).render(80, model{dark: true}.palette())), "EGM plugin v0.1.0")

	plan := previewReport("install", map[string]any{"review": map[string]any{"mode": "egm", "planned": spec.ReviewPlan(t.TempDir(), nil, "egm", []string{"claude", "codex"})}}, spec.Options{Tools: []string{"claude", "codex"}})
	requireText(t, ansi.Strip(plan.render(80, model{dark: true}.palette())), "EGM plugin", "Claude Code: install egm@egm for this project", "for every Codex session")

	done := operationReport("install", map[string]any{"clients": map[string]spec.Client{}, "review": map[string]any{"mode": "spec", "plugins": map[string]spec.Extension{}, "notes": []string{"Codex keeps the EGM plugin SPEC installed for every session"}}}, 0)
	requireText(t, ansi.Strip(done.render(80, model{dark: true}.palette())), "SPEC, fresh context", "Codex keeps the EGM plugin")
}

func TestFailedEGMRemovalAndUpdatePlanAreVisible(t *testing.T) {
	d := installedStatus()
	d["review"] = map[string]any{"mode": "spec", "plugins": map[string]spec.Extension{"egm:claude": {Status: "pending", Reason: "removing EGM failed: permission denied"}, "egm:codex": {Status: "kept"}}}
	r := overviewReport(snapshot{status: d})
	text := ansi.Strip(r.render(80, model{dark: true}.palette()))
	requireText(t, text, "SPEC; EGM removal pending", "permission denied", "Repair setup")
	if r.tone != "warning" {
		t.Fatal(r.summary)
	}
	plan := previewReport("update", map[string]any{"current": "0.3.0", "selected": "v0.4.0", "review": map[string]any{"mode": "spec", "planned": []string{"Remove the Claude Code plugin egm@egm from this project"}}}, spec.Options{})
	requireText(t, ansi.Strip(plan.render(80, model{dark: true}.palette())), "SPEC, fresh context", "Remove the Claude Code plugin egm@egm")
}
