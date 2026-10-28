package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// Key handling and the help bar share these bindings, so the bar always
// names the keys that actually work on the current screen.
var keys = struct {
	Move, Up, Down, Edit, Select, Change, Open, Apply, Cancel, Back, Scroll, Refresh, Quit, Interrupt key.Binding
}{
	Move:      key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("up/down", "move")),
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "up")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "down")),
	Edit:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Change:    key.NewBinding(key.WithKeys("enter", "space", "left", "right"), key.WithHelp("enter", "change")),
	Open:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Apply:     key.NewBinding(key.WithKeys("enter", "y"), key.WithHelp("enter/y", "apply")),
	Cancel:    key.NewBinding(key.WithKeys("esc", "n", "q"), key.WithHelp("esc/n", "cancel")),
	Back:      key.NewBinding(key.WithKeys("esc", "enter", "q", "backspace"), key.WithHelp("esc", "back")),
	Scroll:    key.NewBinding(key.WithKeys("pgup", "pgdown", "space", "home", "end"), key.WithHelp("pgup/pgdn", "scroll")),
	Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:      key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
	Interrupt: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel and quit")),
}

// shortHelp lists the bindings for the current screen, most important first;
// the help bar drops trailing items when the terminal is narrow.
func (m model) shortHelp() []key.Binding {
	if m.busy {
		return []key.Binding{keys.Interrupt}
	}
	switch m.screen {
	case projectScreen:
		back := keys.Back
		back.SetHelp("esc", "cancel")
		if m.workspace.status == nil && m.workspace.err == nil {
			back.SetHelp("esc", "quit")
		}
		return []key.Binding{keys.Open, back}
	case optionsScreen:
		if m.language.Focused() {
			save, cancel := keys.Select, keys.Back
			save.SetHelp("enter", "save")
			cancel.SetHelp("esc", "cancel")
			return []key.Binding{save, cancel}
		}
		return []key.Binding{keys.Move, keys.Change, keys.Back}
	case docsScreen:
		read := keys.Select
		read.SetHelp("enter", "read")
		return []key.Binding{read, keys.Edit, keys.Back, keys.Move}
	case viewScreen:
		return []key.Binding{keys.Edit, keys.Back, keys.Scroll}
	case editScreen:
		return editorHelp(m.editor)
	case planScreen:
		return []key.Binding{keys.Apply, keys.Cancel, keys.Scroll}
	case reportScreen:
		return []key.Binding{keys.Back, keys.Scroll}
	}
	if m.width < 60 {
		return []key.Binding{keys.Move, keys.Select, keys.Quit}
	}
	return []key.Binding{keys.Move, keys.Select, keys.Refresh, keys.Quit}
}

// The editor's hints are display-only bindings: the editor reads raw keys.
func editorHelp(e editor) []key.Binding {
	hint := func(k, desc string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc)) }
	switch {
	case e.prompt != "":
		return []key.Binding{hint("enter", "run"), hint("esc", "cancel")}
	case !e.vim:
		return []key.Binding{hint("^S", "save"), hint("^X", "exit"), hint("^K", "cut"), hint("^U", "paste"), hint("^W", "search"), hint("^Z", "undo")}
	case e.insert:
		return []key.Binding{hint("esc", "normal mode"), hint("^S", "save")}
	case e.visual != "":
		return []key.Binding{hint("esc", "normal"), hint("y", "copy"), hint("d", "cut"), hint("c", "change"), hint("p", "replace"), hint(">", "indent"), hint("o", "other end")}
	}
	return []key.Binding{hint("i", "insert"), hint(":w", "save"), hint(":q", "quit"), hint("v", "select"), hint("/", "search"), hint("u", "undo")}
}

func (m *model) setHelpStyles() {
	p := m.palette()
	m.help.ShortSeparator = "   "
	m.help.Ellipsis = "..."
	m.help.Styles = help.Styles{ShortKey: p.text, ShortDesc: p.muted, ShortSeparator: p.muted, Ellipsis: p.muted}
}
