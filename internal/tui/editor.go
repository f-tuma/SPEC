package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// editor wraps the Bubbles textarea with two key styles. Nano keys pass
// straight to the textarea plus save, exit, cut/paste, search and undo. Vim
// keys add normal and visual modes with common motions and operators, ":"
// commands and "/" search. The textarea stays the single owner of the text.
type editor struct {
	area                textarea.Model
	path, digest, saved string
	vim, insert         bool
	prompt, input       string // ":" or "/" while a command or search is typed
	pending             string // vim operator waiting for its motion: d, y, g
	register, search    string
	hasRegister         bool
	registerLines       bool   // the register holds whole lines (dd, yy, V)
	visual              string // "char" or "line" while vim visual mode is active
	anchor, cursor      editState
	undo, redo          []editState
	typing, quitArmed   bool
	message             string
	messageError        bool
}

type editState struct {
	value    string
	row, col int
}

type editorAction int

const (
	editorNone editorAction = iota
	editorSave
	editorForceSave
	editorSaveQuit
	editorQuit
)

const maxUndo = 200

// newEditor refuses content the textarea would change on load (tabs, carriage
// returns, more than its line limit), so a save can never silently rewrite it.
func newEditor(path, content, digest string, vim bool, width, height int, styles textarea.Styles) (editor, error) {
	area := textarea.New()
	area.Prompt = ""
	area.ShowLineNumbers = true
	// The textarea sizes its line-number gutter from MaxHeight, so give it
	// enough digits for this document (and room to grow) to keep text aligned.
	digits := len(strconv.Itoa(max(99, 2*(strings.Count(content, "\n")+1))))
	area.CharLimit, area.MaxWidth = 0, 0
	area.MaxHeight = int(math.Pow10(digits)) - 1
	area.SetStyles(styles)
	area.SetWidth(width)
	area.SetHeight(height)
	area.SetValue(content)
	if area.Value() != content {
		return editor{}, fmt.Errorf("%s contains tabs, carriage returns or more lines than the built-in editor keeps unchanged. Choose Your editor ($EDITOR) in Setup options to edit it", path)
	}
	area.MoveToBegin()
	area.Focus()
	return editor{area: area, path: path, digest: digest, saved: content, vim: vim}, nil
}

func (e editor) dirty() bool { return e.area.Value() != e.saved }

func (e editor) mode() string {
	switch {
	case !e.vim:
		return "NANO"
	case e.insert:
		return "INSERT"
	case e.visual == "char":
		return "VISUAL"
	case e.visual == "line":
		return "VISUAL LINE"
	}
	return "NORMAL"
}

func (e *editor) say(text string, isError bool) { e.message, e.messageError = text, isError }

// Update handles one message and reports what the host should do next.
func (e editor) Update(msg tea.Msg) (editor, editorAction, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		e.area, cmd = e.area.Update(msg)
		return e, editorNone, cmd
	}
	key := press.String()
	if e.prompt != "" {
		return e.promptKey(press)
	}
	if key != "ctrl+x" {
		e.quitArmed = false
	}
	if e.vim && e.visual != "" {
		e.message = ""
		return e.visualKey(press)
	}
	if e.vim && !e.insert {
		e.message = ""
		return e.normalKey(press)
	}
	switch key {
	case "ctrl+s":
		return e, editorSave, nil
	case "esc", "ctrl+c":
		if e.vim {
			e.insert, e.typing = false, false
			return e, editorNone, nil
		}
		if key == "ctrl+c" {
			e.say("Ctrl+X exits the editor.", false)
		}
		return e, editorNone, nil
	}
	if !e.vim {
		switch key {
		case "ctrl+x":
			if e.dirty() && !e.quitArmed {
				e.quitArmed = true
				e.say("Unsaved changes. Ctrl+S saves; Ctrl+X again discards them.", true)
				return e, editorNone, nil
			}
			return e, editorQuit, nil
		case "ctrl+k":
			// Shift+arrows mark text in the textarea; Ctrl+K cuts the mark or the line.
			if e.area.HasSelection() {
				start, end, _ := e.area.Selection()
				e.push(e.snapshot())
				e.cutRange(e.offsetOf(start.Row, start.Col), e.offsetOf(end.Row, end.Col), false)
				return e, editorNone, nil
			}
			e.cutLine()
			return e, editorNone, nil
		case "ctrl+u":
			e.pasteLine(false)
			return e, editorNone, nil
		case "ctrl+z", "alt+u":
			e.undoChange()
			return e, editorNone, nil
		case "ctrl+y", "alt+e":
			e.redoChange()
			return e, editorNone, nil
		case "ctrl+w":
			e.prompt, e.input = "/", ""
			return e, editorNone, nil
		}
	}
	e.message = ""
	return e, editorNone, e.forward(press)
}

// forward lets the textarea edit and records undo state. Consecutive typed
// characters within a word form one undo step.
func (e *editor) forward(msg tea.KeyPressMsg) tea.Cmd {
	before := e.snapshot()
	var cmd tea.Cmd
	e.area, cmd = e.area.Update(msg)
	typed := len(msg.Text) > 0 && msg.Text != " " && msg.Mod == 0
	if e.area.Value() != before.value {
		if !(e.typing && typed) {
			e.push(before)
		}
		e.typing = typed
	} else if !typed {
		e.typing = false
	}
	return cmd
}

func (e *editor) send(code rune) { e.area, _ = e.area.Update(tea.KeyPressMsg{Code: code}) }

func (e editor) normalKey(press tea.KeyPressMsg) (editor, editorAction, tea.Cmd) {
	key := press.String()
	if e.pending != "" {
		pending := e.pending
		e.pending = ""
		switch pending + key {
		case "dd":
			e.cutLine()
		case "yy":
			e.register, e.hasRegister, e.registerLines = e.lines()[e.area.Line()], true, true
			e.say("Line copied.", false)
		case "gg":
			e.area.MoveToBegin()
		}
		return e, editorNone, nil
	}
	if e.motion(key) {
		e.typing = false
		return e, editorNone, nil
	}
	switch key {
	case "v", "V":
		e.visual = map[string]string{"v": "char", "V": "line"}[key]
		e.anchor, e.cursor = e.snapshot(), e.snapshot()
		e.highlight()
	case "d", "y", "g":
		e.pending = key
	case "i":
		e.insert = true
	case "a":
		if e.area.Column() < len([]rune(e.lines()[e.area.Line()])) {
			e.send(tea.KeyRight)
		}
		e.insert = true
	case "I":
		e.firstNonBlank()
		e.insert = true
	case "A":
		e.area.CursorEnd()
		e.insert = true
	case "o", "O":
		e.openLine(key == "O")
		e.insert = true
	case "x":
		if e.area.Column() < len([]rune(e.lines()[e.area.Line()])) {
			e.push(e.snapshot())
			e.send(tea.KeyDelete)
		}
	case "D":
		e.push(e.snapshot())
		lines, row, col := e.lines(), e.area.Line(), e.area.Column()
		lines[row] = string([]rune(lines[row])[:col])
		e.apply(strings.Join(lines, "\n"), row, col)
	case "p", "P":
		if e.hasRegister && !e.registerLines {
			e.pasteText(key == "p")
		} else {
			e.pasteLine(key == "p")
		}
	case "u":
		e.undoChange()
	case "ctrl+r":
		e.redoChange()
	case "/", ":":
		e.prompt, e.input = key, ""
	case "ctrl+s":
		return e, editorSave, nil
	case "esc", "ctrl+c":
		e.message = ""
	}
	e.typing = false
	return e, editorNone, nil
}

// motion moves the cursor for keys shared by normal and visual mode.
func (e *editor) motion(key string) bool {
	switch key {
	case "h", "left":
		e.send(tea.KeyLeft)
	case "l", "right":
		e.send(tea.KeyRight)
	case "j", "down":
		e.send(tea.KeyDown)
	case "k", "up":
		e.send(tea.KeyUp)
	case "pgdown", "ctrl+d", "ctrl+f":
		e.send(tea.KeyPgDown)
	case "pgup", "ctrl+u", "ctrl+b":
		e.send(tea.KeyPgUp)
	case "w", "b", "e":
		e.moveWord(key)
	case "0", "home":
		e.area.CursorStart()
	case "^":
		e.firstNonBlank()
	case "$", "end":
		e.area.CursorEnd()
	case "G":
		e.area.MoveToEnd()
		e.area.CursorStart()
	case "n", "N":
		e.find(key == "N")
	default:
		return false
	}
	return true
}

// visualKey handles vim visual mode. The editor keeps the anchor and the vim
// cursor itself; highlight mirrors the range into the textarea selection.
func (e editor) visualKey(press tea.KeyPressMsg) (editor, editorAction, tea.Cmd) {
	key := press.String()
	e.setCursor(e.cursor.row, e.cursor.col)
	if e.pending == "g" {
		e.pending = ""
		if key == "g" {
			e.area.MoveToBegin()
		}
	} else if !e.motion(key) {
		start, end := e.visualRange()
		switch key {
		case "esc", "ctrl+c":
			e.leaveVisual(e.cursor)
			return e, editorNone, nil
		case "v", "V":
			mode := map[string]string{"v": "char", "V": "line"}[key]
			if e.visual == mode {
				e.leaveVisual(e.cursor)
				return e, editorNone, nil
			}
			e.visual = mode
		case "o":
			e.anchor, e.cursor = e.cursor, e.anchor
			e.setCursor(e.cursor.row, e.cursor.col)
		case "g":
			e.pending = "g"
		case "y":
			e.register, e.hasRegister, e.registerLines = string([]rune(e.area.Value())[start:end]), true, e.visual == "line"
			if e.registerLines {
				e.register = strings.TrimSuffix(e.register, "\n")
			}
			e.leaveVisual(e.startState(start))
			e.say("Selection copied.", false)
			return e, editorNone, nil
		case "d", "x", "c":
			e.push(e.snapshot())
			linewise := e.visual == "line"
			e.cutRange(start, end, linewise)
			e.visual = ""
			e.area.ClearSelection()
			e.insert = key == "c"
			if e.insert && linewise {
				e.openAt(e.area.Line())
			}
			return e, editorNone, nil
		case ">", "<":
			e.push(e.snapshot())
			e.indent(key == ">")
			e.leaveVisual(e.snapshot())
			return e, editorNone, nil
		case "p":
			if e.hasRegister {
				e.push(e.snapshot())
				text := []rune(e.area.Value())
				replacement := e.register
				if e.visual == "line" && end > 0 && end <= len(text) && text[end-1] == '\n' {
					replacement += "\n"
				}
				e.apply(string(text[:start])+replacement+string(text[end:]), 0, 0)
				e.moveTo(start)
			}
			e.leaveVisual(e.snapshot())
			return e, editorNone, nil
		case "/":
			e.prompt, e.input = "/", ""
		}
	}
	e.cursor = e.snapshot()
	e.highlight()
	return e, editorNone, nil
}

// visualRange returns the selected rune offsets [start, end). Character mode
// includes the character under the cursor; line mode covers whole lines and
// their line break.
func (e *editor) visualRange() (int, int) {
	text := []rune(e.area.Value())
	if e.visual == "line" {
		first, last := min(e.anchor.row, e.cursor.row), max(e.anchor.row, e.cursor.row)
		start, end := e.offsetOf(first, 0), e.offsetOf(last, len([]rune(e.lines()[last])))
		if end < len(text) {
			end++
		}
		return start, end
	}
	a, c := e.offsetOf(e.anchor.row, e.anchor.col), e.offsetOf(e.cursor.row, e.cursor.col)
	return min(a, c), min(max(a, c)+1, len(text))
}

func (e *editor) startState(offset int) editState {
	e.moveTo(offset)
	return e.snapshot()
}

func (e *editor) leaveVisual(at editState) {
	e.visual, e.pending = "", ""
	e.area.ClearSelection()
	e.setCursor(at.row, at.col)
}

// highlight selects the visual range in the textarea through its own
// keyboard selection (Shift+motions), which also renders the highlight.
func (e *editor) highlight() {
	start, end := e.visualRange()
	if e.visual == "line" {
		if text := []rune(e.area.Value()); end > start && end <= len(text) && text[end-1] == '\n' {
			end--
		}
	}
	e.area.ClearSelection()
	e.moveTo(start)
	row, col := e.positionOf(end)
	for i := 0; e.area.Line() < row && i < 1_000_000; i++ {
		e.area, _ = e.area.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	}
	for i := 0; e.area.Column() < col && i < 1_000_000; i++ {
		e.area, _ = e.area.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	}
	for i := 0; e.area.Column() > col && i < 1_000_000; i++ {
		e.area, _ = e.area.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	}
}

// cutRange removes [start, end) into the register. A linewise cut of the last
// line removes the line break before it instead.
func (e *editor) cutRange(start, end int, linewise bool) {
	text := []rune(e.area.Value())
	cut := string(text[start:end])
	e.register, e.hasRegister, e.registerLines = cut, true, linewise
	if linewise {
		e.register = strings.TrimSuffix(cut, "\n")
		if end == len(text) && start > 0 && !strings.HasSuffix(cut, "\n") {
			start--
		}
	}
	e.apply(string(text[:start])+string(text[end:]), 0, 0)
	e.moveTo(min(start, len([]rune(e.area.Value()))))
	if linewise {
		e.area.CursorStart()
	}
}

func (e *editor) indent(right bool) {
	first, last := min(e.anchor.row, e.cursor.row), max(e.anchor.row, e.cursor.row)
	lines := e.lines()
	for i := first; i <= last; i++ {
		if right {
			lines[i] = "  " + lines[i]
		} else {
			lines[i] = strings.TrimPrefix(strings.TrimPrefix(lines[i], " "), " ")
		}
	}
	e.apply(strings.Join(lines, "\n"), first, 0)
}

// pasteText inserts a character register after (or at) the cursor.
func (e *editor) pasteText(after bool) {
	e.push(e.snapshot())
	text, at := []rune(e.area.Value()), e.offset()
	if after && at < len(text) && text[at] != '\n' {
		at++
	}
	e.apply(string(text[:at])+e.register+string(text[at:]), 0, 0)
	e.moveTo(max(at, at+len([]rune(e.register))-1))
}

func (e *editor) openAt(row int) {
	lines := e.lines()
	lines = append(lines[:row], append([]string{""}, lines[row:]...)...)
	e.apply(strings.Join(lines, "\n"), row, 0)
}

func (e editor) promptKey(press tea.KeyPressMsg) (editor, editorAction, tea.Cmd) {
	switch press.String() {
	case "esc", "ctrl+c":
		e.prompt = ""
		return e, editorNone, nil
	case "backspace":
		if e.input == "" {
			e.prompt = ""
		} else {
			runes := []rune(e.input)
			e.input = string(runes[:len(runes)-1])
		}
		return e, editorNone, nil
	case "enter":
		prompt, input := e.prompt, strings.TrimSpace(e.input)
		e.prompt = ""
		if prompt == "/" {
			if input != "" {
				e.search = input
			}
			if e.visual != "" {
				e.setCursor(e.cursor.row, e.cursor.col)
			}
			e.find(false)
			if e.visual != "" {
				e.cursor = e.snapshot()
				e.highlight()
			}
			return e, editorNone, nil
		}
		return e.command(input)
	}
	if press.Text != "" {
		e.input += press.Text
	}
	return e, editorNone, nil
}

func (e editor) command(input string) (editor, editorAction, tea.Cmd) {
	switch input {
	case "w":
		return e, editorSave, nil
	case "w!":
		return e, editorForceSave, nil
	case "wq", "x":
		return e, editorSaveQuit, nil
	case "q":
		if e.dirty() {
			e.say("Unsaved changes. Use :w to save or :q! to discard them.", true)
			return e, editorNone, nil
		}
		return e, editorQuit, nil
	case "q!":
		return e, editorQuit, nil
	case "":
		return e, editorNone, nil
	}
	if line, err := strconv.Atoi(input); err == nil {
		e.setCursor(min(max(line-1, 0), len(e.lines())-1), 0)
		return e, editorNone, nil
	}
	e.say("Unknown command :"+input+". Try :w, :q, :wq or :q!", true)
	return e, editorNone, nil
}

func (e *editor) lines() []string { return strings.Split(e.area.Value(), "\n") }

func (e *editor) snapshot() editState {
	return editState{e.area.Value(), e.area.Line(), e.area.Column()}
}

func (e *editor) push(s editState) {
	e.undo = append(e.undo, s)
	if len(e.undo) > maxUndo {
		e.undo = e.undo[1:]
	}
	e.redo = nil
}

func (e *editor) undoChange() {
	if len(e.undo) == 0 {
		e.say("Nothing to undo.", false)
		return
	}
	e.redo = append(e.redo, e.snapshot())
	s := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.apply(s.value, s.row, s.col)
	e.typing = false
}

func (e *editor) redoChange() {
	if len(e.redo) == 0 {
		e.say("Nothing to redo.", false)
		return
	}
	e.undo = append(e.undo, e.snapshot())
	s := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.apply(s.value, s.row, s.col)
}

func (e *editor) apply(value string, row, col int) {
	e.area.SetValue(value)
	e.setCursor(row, col)
}

// setCursor moves to a logical line and column. CursorDown moves by wrapped
// screen line, so it repeats until the logical line is reached.
func (e *editor) setCursor(row, col int) {
	e.area.MoveToBegin()
	for i := 0; e.area.Line() < row && i < 1_000_000; i++ {
		e.area.CursorDown()
	}
	e.area.SetCursorColumn(col)
}

func (e *editor) cutLine() {
	e.push(e.snapshot())
	lines, row := e.lines(), e.area.Line()
	e.register, e.hasRegister, e.registerLines = lines[row], true, true
	if len(lines) == 1 {
		e.apply("", 0, 0)
		return
	}
	lines = append(lines[:row], lines[row+1:]...)
	e.apply(strings.Join(lines, "\n"), min(row, len(lines)-1), 0)
}

// pasteLine inserts the register as a whole line above the cursor line, or
// below it for vim's "p".
func (e *editor) pasteLine(below bool) {
	if !e.hasRegister {
		e.say("Nothing to paste.", false)
		return
	}
	e.push(e.snapshot())
	lines, row := e.lines(), e.area.Line()
	at := row
	if below {
		at = row + 1
	}
	lines = append(lines[:at], append([]string{e.register}, lines[at:]...)...)
	e.apply(strings.Join(lines, "\n"), at, 0)
}

func (e *editor) openLine(above bool) {
	e.push(e.snapshot())
	lines, row := e.lines(), e.area.Line()
	at := row + 1
	if above {
		at = row
	}
	lines = append(lines[:at], append([]string{""}, lines[at:]...)...)
	e.apply(strings.Join(lines, "\n"), at, 0)
}

func (e *editor) firstNonBlank() {
	line := []rune(e.lines()[e.area.Line()])
	col := 0
	for col < len(line) && unicode.IsSpace(line[col]) {
		col++
	}
	e.area.SetCursorColumn(col)
}

// offset converts between the cursor and a rune offset in the whole text.
func (e *editor) offset() int { return e.offsetOf(e.area.Line(), e.area.Column()) }

func (e *editor) offsetOf(row, col int) int {
	lines, offset := e.lines(), 0
	for i := 0; i < row && i < len(lines); i++ {
		offset += len([]rune(lines[i])) + 1
	}
	return offset + col
}

func (e *editor) positionOf(offset int) (int, int) {
	lines := e.lines()
	for row, line := range lines {
		length := len([]rune(line))
		if offset <= length || row == len(lines)-1 {
			return row, min(offset, length)
		}
		offset -= length + 1
	}
	return 0, 0
}

func (e *editor) moveTo(offset int) { e.setCursor(e.positionOf(offset)) }

func class(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return 1
	}
	return 2
}

func (e *editor) moveWord(motion string) {
	text, at := []rune(e.area.Value()), e.offset()
	switch motion {
	case "w":
		if at < len(text) {
			start := class(text[at])
			for at < len(text) && class(text[at]) == start && start != 0 {
				at++
			}
		}
		for at < len(text) && class(text[at]) == 0 {
			at++
		}
	case "b":
		for at > 0 && class(text[at-1]) == 0 {
			at--
		}
		if at > 0 {
			start := class(text[at-1])
			for at > 0 && class(text[at-1]) == start {
				at--
			}
		}
	case "e":
		at++
		for at < len(text) && class(text[at]) == 0 {
			at++
		}
		if at < len(text) {
			start := class(text[at])
			for at+1 < len(text) && class(text[at+1]) == start {
				at++
			}
		}
	}
	e.moveTo(min(at, len(text)))
}

// find moves to the next (or previous) match of the last search, wrapping
// around the document.
func (e *editor) find(backward bool) {
	if e.search == "" {
		e.say("No search yet. Press / to search.", false)
		return
	}
	text, needle, at := []rune(e.area.Value()), []rune(e.search), e.offset()
	match := func(i int) bool {
		return i >= 0 && i+len(needle) <= len(text) && string(text[i:i+len(needle)]) == e.search
	}
	for step := 1; step <= len(text); step++ {
		i := (at + step) % (len(text) + 1)
		if backward {
			i = ((at-step)%(len(text)+1) + len(text) + 1) % (len(text) + 1)
		}
		if match(i) {
			e.moveTo(i)
			e.say("", false)
			return
		}
	}
	e.say("Not found: "+e.search, true)
}
