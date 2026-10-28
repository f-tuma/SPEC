package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// withDocument writes a project file and opens it through Project documents.
func withDocument(t *testing.T, m model, rel, content string) model {
	t.Helper()
	put := filepath.Join(m.options.Project, rel)
	if err := os.MkdirAll(filepath.Dir(put), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(put, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ = choose(t, m, "Project documents")
	listed := false
	for i, doc := range m.docs {
		if doc.Path == rel {
			m.docCursor, listed = i, true
		}
	}
	if listed {
		m, _ = press(m, tea.KeyEnter)
	} else {
		// Files outside the known locations open the same way the list does.
		next, _ := m.openDocument(rel)
		m = next.(model)
	}
	if m.screen != viewScreen || m.docPath != rel {
		t.Fatalf("document did not open: %s", m.message)
	}
	return m
}
func typeText(m model, text string) model {
	for _, r := range text {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(model)
	}
	return m
}
func ctrl(m model, code rune) model {
	updated, _ := m.Update(tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl})
	return updated.(model)
}
func fileText(t *testing.T, m model, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(m.options.Project, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDocumentsListRendersMarkdownAndReturns(t *testing.T) {
	m := openFixture(t)
	m = withDocument(t, m, "docs/project-brief.md", "# Brief\n\nA **cozy** game.\n\n- grow plants\n")
	requireText(t, viewText(m), "docs/project-brief.md", "# Brief", "A cozy game.", "- grow plants", "e edit")
	if strings.Contains(viewText(m), "**") {
		t.Fatal("Markdown was not rendered")
	}
	m, _ = press(m, tea.KeyEscape)
	if m.screen != docsScreen {
		t.Fatal("Esc should return to the document list")
	}
	requireText(t, viewText(m), "Project documents", "> Brief", "docs/project-brief.md", "enter read")
	m, _ = press(m, tea.KeyEscape)
	if m.screen != homeScreen {
		t.Fatal("Esc should return home")
	}
}

func TestNanoEditorSavesCutsPastesAndGuardsExit(t *testing.T) {
	m := openFixture(t)
	m = withDocument(t, m, "DESIGN.md", "one\ntwo\nthree")
	m, _ = press(m, 'e')
	requireText(t, viewText(m), "Edit DESIGN.md", "NANO", "^S save", "^X exit")
	m = typeText(m, "zero ")
	requireText(t, viewText(m), "modified")
	m = ctrl(m, 's')
	if got := fileText(t, m, "DESIGN.md"); got != "zero one\ntwo\nthree" {
		t.Fatalf("saved %q", got)
	}
	m = ctrl(m, 'k')
	m = ctrl(m, 'u')
	m = ctrl(m, 'u')
	if got := m.editor.area.Value(); got != "zero one\nzero one\ntwo\nthree" {
		t.Fatalf("cut and paste gave %q", got)
	}
	m = ctrl(m, 'z')
	if got := m.editor.area.Value(); got != "zero one\ntwo\nthree" {
		t.Fatalf("undo gave %q", got)
	}
	m = ctrl(m, 'c')
	if m.screen != editScreen {
		t.Fatal("Ctrl+C must not leave the editor")
	}
	m = ctrl(m, 'k')
	m = ctrl(m, 'x')
	if m.screen != editScreen || !strings.Contains(m.editor.message, "Unsaved changes") {
		t.Fatal("exit with unsaved changes was not guarded")
	}
	m = ctrl(m, 'x')
	if m.screen != viewScreen || fileText(t, m, "DESIGN.md") != "zero one\ntwo\nthree" {
		t.Fatal("discarding changes wrote the file or stayed in the editor")
	}
}

func TestVimEditorMotionsOperatorsAndCommands(t *testing.T) {
	m := openFixture(t)
	m.prefs.Editor = "vim"
	m = withDocument(t, m, "notes.md", "alpha beta\ngamma delta\nepsilon")
	m, _ = press(m, 'e')
	requireText(t, viewText(m), "NORMAL", "i insert", ":w save")
	m = typeText(m, "jdd")
	if got := m.editor.area.Value(); got != "alpha beta\nepsilon" {
		t.Fatalf("dd gave %q", got)
	}
	m = typeText(m, "u")
	if got := m.editor.area.Value(); got != "alpha beta\ngamma delta\nepsilon" {
		t.Fatalf("undo gave %q", got)
	}
	m = typeText(m, "ggwiNEW ")
	requireText(t, viewText(m), "INSERT", "esc normal mode")
	m = ctrl(m, 'c')
	if m.screen != editScreen || m.editor.insert {
		t.Fatal("Ctrl+C should leave insert mode, not the editor")
	}
	if got := m.editor.lines()[0]; got != "alpha NEW beta" {
		t.Fatalf("w then insert gave %q", got)
	}
	m = typeText(m, "/epsilon")
	requireText(t, viewText(m), "/epsilon_")
	m, _ = press(m, tea.KeyEnter)
	if m.editor.area.Line() != 2 {
		t.Fatal("search did not move to the match")
	}
	m = typeText(m, ":q")
	m, _ = press(m, tea.KeyEnter)
	if m.screen != editScreen || !strings.Contains(m.editor.message, ":q!") {
		t.Fatal(":q with changes must refuse and explain")
	}
	m = typeText(m, "yyP:wq")
	m, _ = press(m, tea.KeyEnter)
	if m.screen != viewScreen {
		t.Fatal(":wq did not return to the reader:", m.editor.message)
	}
	if got := fileText(t, m, "notes.md"); got != "alpha NEW beta\ngamma delta\nepsilon\nepsilon" {
		t.Fatalf("saved %q", got)
	}
}

func TestSaveRefusesConcurrentChangesAndLossyContent(t *testing.T) {
	m := openFixture(t)
	m = withDocument(t, m, "DESIGN.md", "mine")
	m, _ = press(m, 'e')
	if err := os.WriteFile(filepath.Join(m.options.Project, "DESIGN.md"), []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = typeText(m, "x")
	m = ctrl(m, 's')
	if fileText(t, m, "DESIGN.md") != "theirs" || !strings.Contains(m.editor.message, "changed on disk") {
		t.Fatal("save overwrote a concurrent change")
	}
	m = ctrl(m, 'x')
	m = ctrl(m, 'x')
	m = withDocument(t, m, "Makefile", "build:\n\tgo build ./...\n")
	m, _ = press(m, 'e')
	if m.screen == editScreen || !strings.Contains(m.message, "$EDITOR") {
		t.Fatal("content with tabs must not open in the lossy built-in editor")
	}
}

func TestExternalEditorAndPreferences(t *testing.T) {
	m := openFixture(t)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if _, err := externalEditor("/tmp/x.md"); err == nil {
		t.Fatal("missing $EDITOR must explain itself")
	}
	t.Setenv("VISUAL", "code --wait")
	cmd, err := externalEditor("/tmp/x.md")
	if err != nil || strings.Join(cmd.Args, " ") != "code --wait /tmp/x.md" {
		t.Fatal(cmd, err)
	}
	m, _ = choose(t, m, "Setup options")
	for range 6 {
		m, _ = press(m, tea.KeyDown)
	}
	requireText(t, viewText(m), "Editor", "Nano keys", "Saved for you on this computer")
	m, _ = press(m, tea.KeyEnter)
	if m.prefs.Editor != "vim" || loadPreferences().Editor != "vim" {
		t.Fatal("editor preference was not saved")
	}
	m, _ = press(m, tea.KeyEnter)
	if loadPreferences().Editor != "external" {
		t.Fatal("external choice was not saved")
	}
	if err := os.WriteFile(preferencesPath(), []byte(`{"editor":"emacs"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if loadPreferences().Editor != "nano" {
		t.Fatal("an unknown preference must fall back to nano")
	}
}

func vimEditor(t *testing.T, content string) editor {
	t.Helper()
	e, err := newEditor("a.md", content, "", true, 60, 20, model{dark: true}.editorStyles())
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func vimKeys(e editor, text string) editor {
	for _, r := range text {
		press := tea.KeyPressMsg{Code: r, Text: string(r)}
		if r == 27 {
			press = tea.KeyPressMsg{Code: tea.KeyEscape}
		}
		e, _, _ = e.Update(press)
	}
	return e
}

func TestVimVisualModeSelectsCopiesCutsAndChanges(t *testing.T) {
	// Character-wise: select "beta" (inclusive of the cursor), copy, paste.
	e := vimKeys(vimEditor(t, "alpha beta gamma"), "wve")
	if e.mode() != "VISUAL" || e.area.SelectedText() != "beta" {
		t.Fatalf("mode %s, highlighted %q", e.mode(), e.area.SelectedText())
	}
	e = vimKeys(e, "y$p")
	if got := e.area.Value(); got != "alpha beta gammabeta" || e.mode() != "NORMAL" {
		t.Fatalf("v y p gave %q in %s", got, e.mode())
	}
	// The other end moves with o; d cuts across lines.
	e = vimKeys(vimEditor(t, "one two\nthree four"), "wvjd")
	if got := e.area.Value(); got != "one  four" {
		t.Fatalf("v j d gave %q", got)
	}
	// Line-wise: V j selects whole lines; d, then p puts them back below.
	e = vimKeys(vimEditor(t, "a\nb\nc\nd"), "jVj")
	if e.mode() != "VISUAL LINE" || e.area.SelectedText() != "b\nc" {
		t.Fatalf("highlighted %q", e.area.SelectedText())
	}
	e = vimKeys(e, "d")
	if got := e.area.Value(); got != "a\nd" || e.area.Line() != 1 {
		t.Fatalf("V j d gave %q at line %d", got, e.area.Line())
	}
	e = vimKeys(e, "p")
	if got := e.area.Value(); got != "a\nd\nb\nc" {
		t.Fatalf("p after V d gave %q", got)
	}
	// Cutting the last lines removes the line break before them.
	if got := vimKeys(vimEditor(t, "a\nb\nc"), "GVkd").area.Value(); got != "a" {
		t.Fatalf("V on the last lines gave %q", got)
	}
	// Indent, change, escape and undo.
	e = vimKeys(vimEditor(t, "x\ny"), "Vj>")
	if got := e.area.Value(); got != "  x\n  y" {
		t.Fatalf("> gave %q", got)
	}
	e = vimKeys(e, "u")
	if got := e.area.Value(); got != "x\ny" {
		t.Fatalf("undo after > gave %q", got)
	}
	e = vimKeys(vimEditor(t, "hello world"), "wvecthere\x1b")
	if got := e.area.Value(); got != "hello there" || e.mode() != "NORMAL" {
		t.Fatalf("v e c gave %q in %s", got, e.mode())
	}
	e = vimKeys(vimEditor(t, "abc"), "vl\x1b")
	if e.mode() != "NORMAL" || e.area.HasSelection() || e.area.Value() != "abc" {
		t.Fatal("Esc did not leave visual mode cleanly")
	}
}

func TestNanoShiftSelectionCutsWithCtrlK(t *testing.T) {
	e, err := newEditor("a.md", "keep cut rest", "", false, 60, 10, model{dark: true}.editorStyles())
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		e, _, _ = e.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	for range 4 {
		e, _, _ = e.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	}
	e, _, _ = e.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	if got := e.area.Value(); got != "keep rest" {
		t.Fatalf("Ctrl+K with a selection gave %q", got)
	}
	e, _, _ = e.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	if got := e.area.Value(); got != "keep cut rest" {
		t.Fatalf("undo gave %q", got)
	}
}
