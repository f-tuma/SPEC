package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/f-tuma/SPEC/internal/spec"
)

type palette struct {
	text, muted, accent, warning, failure lipgloss.Style
	// Sidebar and selection styles carry a background. They are applied to
	// plain text only, so an inner reset can never cut the background short.
	side, sideMuted, selected, inactive lipgloss.Style
}

func (m model) palette() palette {
	choose := lipgloss.LightDark(m.dark)
	style := func(light, dark string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(choose(lipgloss.Color(light), lipgloss.Color(dark)))
	}
	sideBg := choose(lipgloss.Color("#EDF1F4"), lipgloss.Color("#1A232D"))
	return palette{
		text: style("#1F2937", "#E2E8F0"), muted: style("#526172", "#94A3B8"),
		accent: style("#006D60", "#86E6C2"), warning: style("#875600", "#F4CF85"),
		failure:   style("#B42332", "#FFABB3"),
		side:      style("#1F2937", "#E2E8F0").Background(sideBg),
		sideMuted: style("#526172", "#8494A8").Background(sideBg),
		selected:  style("#FFFFFF", "#08261F").Background(choose(lipgloss.Color("#006D60"), lipgloss.Color("#86E6C2"))),
		inactive:  style("#1F2937", "#E2E8F0").Background(choose(lipgloss.Color("#D3DCE3"), lipgloss.Color("#2C3A48"))),
	}
}
func (m *model) setInputStyles() {
	p := m.palette()
	s := textinput.DefaultStyles(m.dark)
	s.Focused.Text, s.Blurred.Text = p.text, p.text
	s.Focused.Prompt, s.Blurred.Prompt = p.accent, p.muted
	s.Focused.Placeholder, s.Blurred.Placeholder = p.muted, p.muted
	m.project.SetStyles(s)
	m.language.SetStyles(s)
	m.spinner.Style = p.accent
}

// editorStyles keeps the textarea in the dashboard palette: muted line
// numbers, a subtle current line and regular-weight text.
func (m model) editorStyles() textarea.Styles {
	p := m.palette()
	s := textarea.DefaultStyles(m.dark)
	for _, state := range []*textarea.StyleState{&s.Focused, &s.Blurred} {
		state.Text, state.Placeholder = p.text, p.muted
		state.LineNumber, state.CursorLineNumber = p.muted, p.accent
		state.CursorLine = p.text
		state.EndOfBuffer = p.muted
		state.Selection = p.inactive
	}
	s.Cursor.Color = lipgloss.LightDark(m.dark)(lipgloss.Color("#006D60"), lipgloss.Color("#86E6C2"))
	return s
}

// At 72 columns and wider the menu is a sidebar on the left and the selected
// content sits on the right. Narrower terminals show one column.
type layout struct {
	wide            bool
	side, gap, main int
	body            int
}

func (m model) layout() layout {
	body := max(1, m.height-4)
	if m.width >= 72 {
		return layout{wide: true, side: 26, gap: 3, main: min(m.width-26-3-2, 80), body: body}
	}
	return layout{main: max(1, min(m.width-4, 76)), body: body}
}

func (m model) View() tea.View {
	if m.width < 44 || m.height < 22 {
		content := []string{"SPEC " + spec.Version, "Use a terminal at least 44 x 22.", "Resize to continue; Ctrl+C quits."}
		for i := range content {
			content[i] = ansi.Truncate(content[i], max(1, m.width), "")
		}
		v := tea.NewView(strings.Join(content[:min(len(content), max(1, m.height))], "\n"))
		v.AltScreen = true
		return v
	}
	p, l := m.palette(), m.layout()
	main, status := m.main(l)
	if l.wide {
		// Content headings line up with the sidebar's "Actions" label.
		main = append([]string{""}, main...)
	}
	main = strings.Split(strings.Join(main, "\n"), "\n")
	main = main[:min(len(main), l.body)]
	for len(main) < l.body {
		main = append(main, "")
	}
	lines := []string{m.header(), ""}
	if l.wide {
		side := m.sidebar(l)
		for i := range l.body {
			lines = append(lines, side[i]+strings.Repeat(" ", l.gap)+ansi.Truncate(main[i], l.main, "..."))
		}
	} else {
		for _, line := range main {
			lines = append(lines, "  "+ansi.Truncate(line, l.main, "..."))
		}
	}
	if m.busy {
		status = m.spinner.View() + " " + p.accent.Render(singleLine(m.message))
	} else if m.message != "" {
		style := p.accent
		if m.messageError {
			style = p.failure
		}
		status = style.Render(singleLine(m.message))
	}
	lines = append(lines, " "+ansi.Truncate(status, m.width-2, "..."), " "+m.help.ShortHelpView(m.shortHelp()))
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width, "")
	}
	v := tea.NewView(p.text.Render(strings.Join(lines, "\n")))
	v.AltScreen = true
	return v
}

func (m model) header() string {
	p := m.palette()
	badge := p.selected.Render(" SPEC ") + p.muted.Render(" "+spec.Version) + "   "
	if m.options.Project == "" {
		return " " + badge + "No project open"
	}
	name, path := singleLine(filepath.Base(m.options.Project)), singleLine(m.options.Project)
	room := m.width - 2 - ansi.StringWidth(badge) - ansi.StringWidth(name) - 3
	if room < 8 {
		return " " + badge + name
	}
	if over := ansi.StringWidth(path) - room; over > 0 {
		path = ansi.TruncateLeft(path, over+3, "...")
	}
	return " " + badge + name + "   " + p.muted.Render(path)
}

// sidebar renders the menu as a filled column. On other screens the item
// that opened them stays marked, with a quieter highlight: focus is on the
// content column.
func (m model) sidebar(l layout) []string {
	p := m.palette()
	row := func(style lipgloss.Style, text string) string {
		return style.Width(l.side).Render(ansi.Truncate(text, l.side, "..."))
	}
	lines := []string{row(p.side, ""), row(p.sideMuted, "  Actions"), row(p.side, "")}
	items := m.menu()
	cursor := min(m.cursor, len(items)-1)
	for i, it := range items {
		switch {
		case i == cursor && m.screen == homeScreen:
			lines = append(lines, row(p.selected, " > "+it.label))
		case i == cursor:
			lines = append(lines, row(p.inactive, " > "+it.label))
		default:
			lines = append(lines, row(p.side, "   "+it.label))
		}
	}
	// The options used by the next setup stay visible on every screen.
	footer := []string{row(p.sideMuted, "  Next setup"), row(p.side, "  "+toolNames(m.options.Tools)), row(p.side, "  OpenSpec: "+methodShort(m.options)), row(p.side, "  Docs: "+languageShort(m.options.DocumentLanguage)), row(p.side, "  Review: "+strings.ToUpper(m.options.Review))}
	if m.options.NoRegister {
		footer = append(footer, row(p.side, "  Native setup skipped"))
	}
	footer = append(footer, row(p.side, ""))
	if len(lines)+len(footer)+1 > l.body {
		footer = nil
	}
	for len(lines)+len(footer) < l.body {
		lines = append(lines, row(p.side, ""))
	}
	return append(lines, footer...)[:l.body]
}

// main returns the content column and the status line for the current screen.
func (m model) main(l layout) ([]string, string) {
	p, width := m.palette(), l.main
	switch m.screen {
	case projectScreen:
		return []string{p.accent.Render("Open a project"), "", wrap("Enter an existing directory. An empty folder is fine; opening it only reads local files.", width), "", m.project.View()}, ""
	case optionsScreen:
		lines := []string{p.accent.Render("Setup options"), "", wrap("Used by the next setup in this session. Nothing is written now.", width), ""}
		values := []string{check(selectedTool(m.options.Tools, "claude")), check(selectedTool(m.options.Tools, "codex")), methodLabel(m.options.Method), "Enabled"}
		if m.options.OpenSpecBin != "" {
			values[2] = "Existing executable"
		}
		if m.options.NoRegister {
			values[3] = "Skipped"
		}
		values = append(values, languageLabel(m.options.DocumentLanguage), reviewLabel(m.options.Review), editorLabel(m.prefs.Editor))
		for i := range optionLabels {
			if i == languageOption && m.language.Focused() {
				lines = append(lines, p.accent.Render(fmt.Sprintf("> %-20s ", optionLabels[i]))+m.language.View())
				continue
			}
			lines = append(lines, m.choice(fmt.Sprintf("%-20s %s", optionLabels[i], values[i]), i == m.optionCursor, width))
		}
		descriptions := []string{
			"Prepare Claude Code during setup.",
			"Prepare Codex during setup.",
			"Auto reuses a pinned CLI or downloads the official binary. pnpm and npm need Node.js.",
			"Install the Claude Code plugin for this project and copy the Codex skills into .agents/skills.",
			"Language for the brief, specs, design, tasks and reviews. Type any language name; leave it empty to follow the conversation. The conversation keeps your language.",
			"SPEC review starts a reviewer without the conversation. EGM installs Google's Elephant-Goldfish Model plugin (v0.1.0) for design reviews: for this project in Claude Code, for every session in Codex.",
			"How documents open for editing: built-in Nano or Vim keys, or your own $VISUAL/$EDITOR. Saved for you on this computer.",
		}
		return append(lines, "", p.muted.Render(wrap(descriptions[m.optionCursor], width))), ""
	case docsScreen:
		return m.docsView(width, l.body), ""
	case viewScreen:
		lines := append([]string{p.accent.Render(singleLine(m.docPath)), ""}, strings.Split(m.viewport.View(), "\n")...)
		status := ""
		if m.viewport.TotalLineCount() > m.viewport.Height() {
			start := m.viewport.YOffset() + 1
			end := min(m.viewport.TotalLineCount(), m.viewport.YOffset()+m.viewport.Height())
			status = p.muted.Render(fmt.Sprintf("Lines %d-%d of %d", start, end, m.viewport.TotalLineCount()))
		}
		return lines, status
	case editScreen:
		title := p.accent.Render("Edit "+singleLine(m.editor.path)) + p.muted.Render("  "+m.editor.mode())
		if m.editor.dirty() {
			title += p.warning.Render("  modified")
		}
		status := ""
		switch {
		case m.editor.prompt != "":
			status = p.text.Render(m.editor.prompt + singleLine(m.editor.input) + "_")
		case m.editor.message != "":
			style := p.muted
			if m.editor.messageError {
				style = p.failure
			}
			status = style.Render(singleLine(m.editor.message))
		}
		return append([]string{title, ""}, strings.Split(m.editor.area.View(), "\n")...), status
	case reportScreen, planScreen:
		style := p.accent
		if m.report.tone == "error" {
			style = p.failure
		} else if m.report.tone == "warning" {
			style = p.warning
		}
		lines := append([]string{style.Render(singleLine(m.report.title)), ""}, strings.Split(m.viewport.View(), "\n")...)
		status := ""
		if m.viewport.TotalLineCount() > m.viewport.Height() {
			start := m.viewport.YOffset() + 1
			end := min(m.viewport.TotalLineCount(), m.viewport.YOffset()+m.viewport.Height())
			status = p.muted.Render(fmt.Sprintf("Lines %d-%d of %d", start, end, m.viewport.TotalLineCount()))
		} else if m.screen == planScreen {
			status = p.muted.Render("Nothing changes until you press Enter.")
		}
		return lines, status
	}
	items := m.menu()
	cursor := min(m.cursor, len(items)-1)
	overview := strings.Split(overviewReport(m.workspace).render(width, p), "\n")
	if !l.wide {
		menu := []string{""}
		for i, it := range items {
			menu = append(menu, m.choice(it.label, i == cursor, width))
		}
		overview = overview[:min(len(overview), max(0, l.body-len(menu)))]
		return append(overview, menu...), p.muted.Render(items[cursor].description)
	}
	// The selected action is explained below the status; the status is cut
	// first when the terminal is short.
	detail := append([]string{"", "", p.accent.Render(items[cursor].label)}, strings.Split(wrap(items[cursor].description, width), "\n")...)
	overview = overview[:min(len(overview), max(0, l.body-1-len(detail)))]
	return append(overview, detail...), ""
}

var documentRoles = map[string]string{"brief": "Brief", "instructions": "Instructions", "visual_design": "Design", "specification": "Spec", "changes": "Change", "architecture": "Architecture", "commands": "Commands", "session": "Session", "release": "Release"}

// docsView lists project documents in a window that follows the cursor.
func (m model) docsView(width, height int) []string {
	p := m.palette()
	lines := []string{p.accent.Render("Project documents"), "", p.muted.Render(wrap("Known locations only. Enter reads a document; e edits it with "+strings.ToLower(editorLabel(m.prefs.Editor))+".", width)), ""}
	if len(m.docs) == 0 {
		return append(lines, "No documents found yet. Your agent can create a brief with you.")
	}
	room := max(1, height-len(strings.Split(strings.Join(lines, "\n"), "\n"))-1)
	start := min(max(0, m.docCursor-room/2), max(0, len(m.docs)-room))
	for i := start; i < min(len(m.docs), start+room); i++ {
		lines = append(lines, m.choice(fmt.Sprintf("%-13s %s", documentRoles[m.docs[i].Role], m.docs[i].Path), i == m.docCursor, width))
	}
	return lines
}

func (m model) choice(label string, selected bool, width int) string {
	if selected {
		return m.palette().selected.Width(width).Render(ansi.Truncate("> "+label, width, "..."))
	}
	return "  " + label
}

func languageShort(language string) string {
	if language == "" {
		return "conversation"
	}
	return language
}

func methodShort(options spec.Options) string {
	if options.OpenSpecBin != "" {
		return "existing"
	}
	if options.Method == "" {
		return "auto"
	}
	return options.Method
}

func check(selected bool) string {
	if selected {
		return "[x]"
	}
	return "[ ]"
}
