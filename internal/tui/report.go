package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/f-tuma/SPEC/internal/spec"
)

// Reports are presentation models, never formatted CLI JSON. The title names
// the outcome; the body holds the relevant facts and any next step.
type report struct {
	title, summary, next, tone string
	sections                   []section
}
type section struct {
	title string
	lines []string
}

// field joins a label and value with a unit separator; render aligns the
// pair, mutes the label and colors the value by meaning.
const fieldSep = "\x1f"
const labelWidth = 13

func field(label, value string) string { return label + fieldSep + value }

func valueStyle(value string, p palette) lipgloss.Style {
	switch value {
	case "Not configured", "Not installed", "None found", "Native agent setup was skipped.", "Follows conversation", "Not recorded":
		return p.muted
	case "Installed; verify skills", "Skills in project; verify":
		return p.accent
	case "Setup pending", "Refresh needed", "Version unknown", "CLI not on PATH", "EGM; setup pending", "SPEC; EGM removal pending":
		return p.warning
	}
	return p.text
}

func renderField(line string, width int, p palette) string {
	label, value, _ := strings.Cut(line, fieldSep)
	label, value = singleLine(label), singleLine(value)
	if width < labelWidth+12 {
		return p.muted.Render(wrap(label, width)) + "\n" + valueStyle(value, p).Render(wrap(value, width))
	}
	style := valueStyle(value, p)
	rows := strings.Split(wrap(value, width-labelWidth), "\n")
	for i := range rows {
		prefix := strings.Repeat(" ", labelWidth)
		if i == 0 {
			prefix = p.muted.Render(fmt.Sprintf("%-*s", labelWidth, label))
		}
		rows[i] = prefix + style.Render(rows[i])
	}
	return strings.Join(rows, "\n")
}

func (r report) render(width int, p palette) string {
	style := p.accent
	if r.tone == "warning" {
		style = p.warning
	}
	if r.tone == "error" {
		style = p.failure
	}
	lines := []string{}
	if r.summary != "" {
		lines = append(lines, style.Render(wrap(r.summary, width)))
	}
	if r.next != "" {
		lines = append(lines, wrap(r.next, width))
	}
	for _, s := range r.sections {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		if s.title != "" {
			lines = append(lines, p.muted.Render(wrap(s.title, width)))
		}
		for _, line := range s.lines {
			if strings.Contains(line, fieldSep) {
				lines = append(lines, renderField(line, width, p))
				continue
			}
			lines = append(lines, wrap(line, width))
		}
	}
	return strings.Join(lines, "\n")
}

// overviewReport is the compact status block on the home screen. The menu
// below it carries the next action, so "next" only names steps the menu
// cannot perform.
func overviewReport(w snapshot) report {
	if w.err != nil {
		return report{summary: "Cannot read workspace status", tone: "error", sections: []section{{"", []string{w.err.Error()}}}}
	}
	if w.status == nil {
		return report{summary: "Reading workspace..."}
	}
	r := report{summary: "Ready"}
	installed := w.status["status"] != "not-installed"
	plugin, openspec := "Not installed", "Not configured"
	open := object(w.status["openspec"])
	if installed {
		plugin = "v" + value(w.status, "plugin_version", "unknown")
		if w.status["managed_files"] == "unchanged" {
			plugin += ", files unchanged"
		}
		if len(open) != 0 {
			openspec = value(open, "version", "Version unknown") + " (recorded)"
		} else {
			r.summary, r.tone = "OpenSpec setup needed", "warning"
		}
	} else {
		r.summary, r.tone = "Setup needed", "warning"
	}
	lines := []string{field("Plugin", plugin), field("OpenSpec", openspec)}
	clients := clientMap(w.status["clients"])
	configuredTools := stringList(open["tools"])
	missingCLI, clientSetupNeeded := false, false
	notes := []string{}
	for _, name := range []string{"claude", "codex"} {
		c := clients[name]
		status := clientLabel(c)
		if found, checked := w.commands[name]; checked && !found {
			status = "CLI not on PATH"
			if selectedTool(configuredTools, name) || c.Status != "" {
				missingCLI = true
			}
		}
		lines = append(lines, field(agentName(name), status))
		if c.Status == "pending" || c.Status == "stale" || c.Status == "version-unverified" {
			clientSetupNeeded = true
		}
		if c.Status == "" && selectedTool(configuredTools, name) {
			clientSetupNeeded = true
		}
		if c.Reason != "" {
			notes = append(notes, agentName(name)+": "+c.Reason)
		}
	}
	if installed && len(open) != 0 {
		if clientSetupNeeded {
			r.summary, r.tone = "Agent setup needs attention", "warning"
		}
		if missingCLI {
			r.summary, r.next, r.tone = "Agent CLI missing", "Install missing agent CLIs, then press r to refresh.", "warning"
		}
	}
	if installed && w.status["managed_files"] != "unchanged" {
		r.summary, r.next, r.tone = "Managed files need attention", "Restore or remove the modified plugin files, then refresh.", "warning"
		notes = append(notes, value(w.status, "managed_files", "File integrity is unknown."))
	}
	if plugin := value(w.status, "codex_global_plugin", ""); installed && plugin != "" {
		notes = append(notes, globalCodexNote(plugin, value(w.status, "codex_global_cleanup", "")))
	}
	if installed && w.status["catalogs"] != "unchanged" {
		r.summary, r.next, r.tone = "Workspace metadata needs attention", "Repair setup rewrites the catalogs from the receipt.", "warning"
		notes = append(notes, value(w.status, "catalogs", "Catalog integrity is unknown."))
	}
	if r.tone == "" {
		r.next = "Start an agent here with spec run --project . --agent claude (or codex)."
	}
	if w.update != "" {
		lines = append(lines, field("Update", w.update+" available"))
	}
	lines = append(lines, field("Documents", documentsLabel(documentMap(w.documents["documents"]))))
	if installed {
		language, requested := value(w.status, "document_language", ""), value(w.status, "config_document_language", "")
		lines = append(lines, field("Language", languageLabel(language)))
		if requested != "" && requested != language {
			notes = append(notes, "openspec/config.yaml requests "+requested+"; SPEC uses "+languageLabel(language)+". Update its Document language line.")
		}
		label, reasons := reviewState(object(w.status["review"]))
		lines = append(lines, field("Review", label), field("Checks", checksLabel(w.status["project_checks"])))
		notes = append(notes, reasons...)
		if strings.HasSuffix(label, "pending") && len(open) != 0 && r.tone == "" {
			r.summary, r.next, r.tone = "Agent setup needs attention", "Choose Repair setup to finish the review plugin setup.", "warning"
		}
	}
	r.sections = append(r.sections, section{lines: lines})
	if len(notes) > 0 {
		r.sections = append(r.sections, section{lines: notes})
	}
	return r
}

func languageLabel(language string) string {
	if language == "" {
		return "Follows conversation"
	}
	return language
}

func documentsLabel(docs map[string][]string) string {
	names := []string{}
	for _, role := range []struct{ key, label string }{
		{"brief", "brief"}, {"specification", "specs"}, {"visual_design", "design"}, {"instructions", "instructions"},
	} {
		if n := len(docs[role.key]); n == 1 {
			names = append(names, role.label)
		} else if n > 1 {
			names = append(names, fmt.Sprintf("%s (%d)", role.label, n))
		}
	}
	if len(names) == 0 {
		return "None found"
	}
	return strings.Join(names, ", ")
}

func previewReport(operation string, data map[string]any, options spec.Options) report {
	registration := "Claude Code plugin installer; Codex skills in .agents/skills"
	if options.NoRegister {
		registration = "Skipped; agents may need setup later"
	}
	switch operation {
	case "install":
		plugin, open := object(data["plugin"]), object(data["openspec"])
		method := methodLabel(options.Method)
		if options.OpenSpecBin != "" {
			method = "Existing executable: " + options.OpenSpecBin
		}
		lines := []string{
			field("Plugin", "v"+value(plugin, "plugin_version", spec.Version)),
			field("OpenSpec", value(open, "version", spec.OpenSpecVersion)+" via "+method),
			field("Agents", toolNames(options.Tools)),
			field("Native setup", registration),
			field("Language", languageLabel(value(open, "document_language", options.DocumentLanguage))),
			field("Review", reviewLabel(value(object(data["review"]), "mode", options.Review))),
		}
		if cache := value(open, "cache", options.Cache); cache != "" {
			lines = append(lines, field("Cache", cache))
		}
		kept := "Existing project documents and OpenSpec configuration are kept."
		if open["existing_config_preserved"] == true {
			kept = "Existing project documents and " + value(open, "existing_config_path", "OpenSpec configuration") + " are kept."
		}
		sections := []section{{"", lines}, {"", []string{
			"Writes the versioned plugin and catalogs into .spec and adds missing OpenSpec integrations. " + kept,
		}}}
		if line := value(open, "config_language_line", ""); line != "" {
			sections = append(sections, section{"OpenSpec config", []string{
				"The existing config is not edited. For OpenSpec to request this language too, add under context:", "  " + line,
			}})
		} else if requested := value(open, "config_document_language", ""); requested != "" && value(open, "document_language", "") == "" {
			sections = append(sections, section{"OpenSpec config", []string{
				"The existing config is not edited and still requests " + requested + ". Remove its Document language line to follow the conversation.",
			}})
		}
		if planned := stringList(object(data["review"])["planned"]); len(planned) > 0 {
			sections = append(sections, section{"EGM plugin", planned})
		}
		return report{title: "Set up workspace", sections: sections}
	case "update":
		sections := []section{{"", []string{
			field("Installed", value(data, "current", "Unknown")),
			field("New", value(data, "selected", "Unknown")),
			field("Source", value(data, "source", "Configured repository")),
			field("Native setup", registration),
			field("Review", reviewLabel(value(object(data["review"]), "mode", options.Review))),
		}}, {"", []string{"Switches the managed plugin and catalogs to this release. Project documents and OpenSpec stay as they are."}}}
		if planned := stringList(object(data["review"])["planned"]); len(planned) > 0 {
			sections = append(sections, section{"EGM plugin", planned})
		}
		return report{title: "Update plugin", sections: sections}
	}
	return report{title: "Change plan"}
}

func operationReport(operation string, data map[string]any, code int) report {
	if operation == "audit" {
		return auditReport(data)
	}
	r := report{title: "Setup complete", next: "Open an agent and check that the SPEC skills load."}
	if operation == "update" {
		r.title = "Plugin updated"
	}
	if data["status"] == "already-newer" {
		r.title, r.next = "No update applied", "The installed plugin is newer than the source release. Use the CLI with an explicit tag to downgrade."
	}
	version := value(data, "plugin_version", value(data, "current", ""))
	if nested := object(data["plugin"]); version == "" {
		version = value(nested, "plugin_version", "Unknown")
	}
	lines := []string{field("Plugin", "v"+strings.TrimPrefix(version, "v"))}
	notes := []string{}
	if open := object(data["openspec"]); len(open) > 0 {
		lines = append(lines, field("OpenSpec", value(open, "version", "Unknown")))
		if created := stringList(open["created_files"]); len(created) > 0 {
			notes = append(notes, fmt.Sprintf("Added %d missing OpenSpec integration files.", len(created)))
		}
		if preserved := stringList(open["preserved_files"]); len(preserved) > 0 {
			notes = append(notes, fmt.Sprintf("Kept %d existing OpenSpec integration files.", len(preserved)))
		}
	}
	clients := clientMap(data["clients"])
	for _, name := range []string{"claude", "codex"} {
		if c, ok := clients[name]; ok {
			lines = append(lines, field(agentName(name), clientLabel(c)))
			if c.Reason != "" {
				notes = append(notes, agentName(name)+": "+c.Reason)
			}
		}
	}
	if len(clients) == 0 {
		notes = append(notes, "Native agent setup was skipped.")
	}
	if review := object(data["review"]); len(review) != 0 {
		label, reasons := reviewState(review)
		lines = append(lines, field("Review", label))
		notes = append(notes, reasons...)
		notes = append(notes, stringList(review["notes"])...)
	}
	if plugin := value(data, "codex_global_plugin", ""); plugin != "" {
		notes = append(notes, globalCodexNote(plugin, value(data, "codex_global_cleanup", "")))
	}
	r.sections = []section{{"", lines}}
	if len(notes) > 0 {
		r.sections = append(r.sections, section{"", notes})
	}
	if code == 3 {
		r.title, r.summary, r.next, r.tone = "Setup partly complete", "The workspace is prepared, but agent setup is pending.", "Fix the agent issue below, then choose Repair setup.", "warning"
	}
	return r
}

func upToDateReport(data map[string]any) report {
	current, latest := value(data, "current", "Unknown"), value(data, "selected", "Unknown")
	return report{title: "Plugin update", summary: "The plugin is up to date.", sections: []section{{"", []string{field("Installed", current), field("Latest", latest)}}}}
}

func auditReport(data map[string]any) report {
	docs := documentMap(data["documents"])
	count := 0
	for _, paths := range docs {
		count += len(paths)
	}
	fileWord := "files"
	if count == 1 {
		fileWord = "file"
	}
	r := report{title: "Project documents", summary: fmt.Sprintf("Found %d candidate %s.", count, fileWord)}
	roles := []struct{ key, label string }{
		{"brief", "Project brief"}, {"specification", "Behavior specifications"},
		{"visual_design", "Visual design"}, {"instructions", "Agent instructions"},
		{"architecture", "Architecture"}, {"commands", "Build and test commands"},
		{"session", "Work session"}, {"release", "Release process"},
	}
	known, missing := map[string]bool{}, []string{}
	for _, role := range roles {
		known[role.key] = true
		if paths := docs[role.key]; len(paths) > 0 {
			r.sections = append(r.sections, section{role.label, paths})
		} else {
			missing = append(missing, strings.ToLower(role.label[:1])+role.label[1:])
		}
	}
	extra := []string{}
	for role := range docs {
		if !known[role] {
			extra = append(extra, role)
		}
	}
	sort.Strings(extra)
	for _, role := range extra {
		r.sections = append(r.sections, section{role, docs[role]})
	}
	if len(missing) > 0 {
		r.sections = append(r.sections, section{"Not found", []string{strings.Join(missing, ", ")}})
	}
	r.sections = append(r.sections, section{"", []string{"Only known locations were checked. Contents have not been reviewed."}})
	return r
}

func errorReport(operation string, err error) report {
	next := "Resolve the issue below, then retry the action."
	if operation == "status" {
		next = "Check the project path and its .spec ownership or installation files."
	}
	if operation == "update" {
		next = "Check Git repository access and stable release tags, then retry."
	}
	return report{title: "Could not complete " + operationName(operation), next: next, tone: "error", sections: []section{{"Reported error", []string{err.Error()}}}}
}
func operationName(operation string) string {
	switch operation {
	case "status":
		return "the status check"
	case "install":
		return "workspace setup"
	case "update":
		return "the plugin update"
	}
	return operation
}

func clientLabel(c spec.Client) string {
	switch c.Status {
	case "installed-by-client":
		return "Installed; verify skills"
	case "installed-in-project":
		return "Skills in project; verify"
	case "pending":
		return "Setup pending"
	case "stale":
		return "Refresh needed"
	case "version-unverified":
		return "Version unknown"
	}
	return "Not configured"
}

// reviewState names the review procedure and, for EGM, why its setup is
// pending in an agent.
func reviewState(review map[string]any) (string, []string) {
	plugins := extensionMap(review["plugins"])
	if value(review, "mode", "spec") != "egm" {
		// A failed removal after switching back still needs a repair.
		if claude := plugins["egm:claude"]; claude.Status == "pending" {
			return "SPEC; EGM removal pending", []string{"EGM in Claude Code: " + claude.Reason}
		}
		return reviewLabel("spec"), nil
	}
	label, reasons := "EGM plugin", []string{}
	if len(plugins) == 0 {
		label = "EGM; setup pending"
	}
	for _, name := range []string{"claude", "codex"} {
		plugin, ok := plugins["egm:"+name]
		if !ok {
			continue
		}
		if plugin.Status == "pending" {
			label = "EGM; setup pending"
			reasons = append(reasons, "EGM in "+agentName(name)+": "+plugin.Reason)
		} else if label == "EGM plugin" && plugin.Ref != "" {
			label += " " + plugin.Ref
		}
	}
	return label, reasons
}

func reviewLabel(mode string) string {
	if mode == "egm" {
		return "EGM plugin"
	}
	return "SPEC, fresh context"
}

// checksLabel names the project check commands agents recorded.
func checksLabel(data any) string {
	names := []string{}
	if checks, ok := data.(map[string][]string); ok {
		for name := range checks {
			names = append(names, name)
		}
	} else {
		for name := range object(data) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "Not recorded"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func extensionMap(data any) map[string]spec.Extension {
	if v, ok := data.(map[string]spec.Extension); ok {
		return v
	}
	result := map[string]spec.Extension{}
	for name, raw := range object(data) {
		d := object(raw)
		result[name] = spec.Extension{Status: value(d, "status", ""), Reason: value(d, "reason", ""), Ref: value(d, "ref", "")}
	}
	return result
}

// An earlier SPEC setup registered Codex's plugin globally; it now loads in
// every Codex session next to the project's own skills.
func globalCodexNote(plugin, cleanup string) string {
	note := "Codex: an earlier setup enabled " + plugin + " in every Codex session."
	if cleanup != "" {
		note += " Remove it with " + cleanup + "."
	}
	return note
}

func foundLabel(paths []string) string {
	if len(paths) == 0 {
		return "No candidate found"
	}
	if len(paths) == 1 {
		return "1 candidate file"
	}
	return fmt.Sprintf("%d candidate files", len(paths))
}
func methodLabel(method string) string {
	switch method {
	case "binary":
		return "Standalone binary"
	case "npm":
		return "npm (Node.js required)"
	case "pnpm":
		return "pnpm (Node.js required)"
	}
	return "Auto (reuse or binary)"
}
func toolNames(tools []string) string {
	names := []string{}
	for _, name := range tools {
		names = append(names, agentName(name))
	}
	return strings.Join(names, ", ")
}
func value(data map[string]any, key, fallback string) string {
	if s, ok := data[key].(string); ok && s != "" {
		return s
	}
	return fallback
}
func object(data any) map[string]any {
	if v, ok := data.(map[string]any); ok {
		return v
	}
	return nil
}
func stringList(data any) []string {
	if v, ok := data.([]string); ok {
		return v
	}
	result := []string{}
	if values, ok := data.([]any); ok {
		for _, v := range values {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}
func documentMap(data any) map[string][]string {
	if v, ok := data.(map[string][]string); ok {
		return v
	}
	result := map[string][]string{}
	for role, paths := range object(data) {
		result[role] = stringList(paths)
	}
	return result
}
func clientMap(data any) map[string]spec.Client {
	if v, ok := data.(map[string]spec.Client); ok {
		return v
	}
	result := map[string]spec.Client{}
	for name, raw := range object(data) {
		d := object(raw)
		result[name] = spec.Client{Status: value(d, "status", ""), Reason: value(d, "reason", "")}
	}
	return result
}
func versionGreater(a, b string) bool {
	aa, bb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	if len(aa) != 3 || len(bb) != 3 {
		return false
	}
	for i := 0; i < 3; i++ {
		x, xe := strconv.ParseUint(aa[i], 10, 64)
		y, ye := strconv.ParseUint(bb[i], 10, 64)
		if xe != nil || ye != nil {
			return false
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// Keep paths and external errors as text: they cannot inject terminal styling,
// control sequences, tabs, or a different font/weight into a report.
func clean(text string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(strings.ToValidUTF8(text, "")))
}
func wrap(text string, width int) string { return ansi.Wrap(clean(text), max(1, width), " ") }

func singleLine(text string) string { return strings.ReplaceAll(clean(text), "\n", " ") }
