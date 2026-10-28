package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/f-tuma/SPEC/internal/spec"
)

// The dashboard shows one screen at a time. Home lists only the actions that
// make sense for the current workspace; every other screen returns with Esc.
type screen int

const (
	homeScreen screen = iota
	projectScreen
	optionsScreen
	docsScreen
	viewScreen
	editScreen
	reportScreen
	planScreen
)

type item struct {
	label, description string
	operation          string
	preview            bool
}

type snapshot struct {
	status, documents map[string]any
	commands          map[string]bool
	err               error
	update            string // a newer stable plugin release, when the quiet check found one
}

type resultMsg struct {
	operation string
	data      map[string]any
	code      int
	err       error
	preview   bool
	options   spec.Options
	workspace snapshot
}
type editorDoneMsg struct{ err error }
type updateMsg struct{ project, current, latest string }

type model struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	manager              *spec.Manager
	options              spec.Options
	project              textinput.Model
	language             textinput.Model
	spinner              spinner.Model
	viewport             viewport.Model
	help                 help.Model
	screen               screen
	width, height        int
	cursor, optionCursor int
	busy, quitting, dark bool
	languageChosen       bool
	reviewChosen         bool
	pending              string
	prefs                preferences
	docs                 []spec.Document
	docCursor            int
	docPath, docContent  string
	docDigest            string
	editor               editor
	pendingOptions       spec.Options
	message              string
	messageError         bool
	workspace            snapshot
	report               report
}

func newModel(ctx context.Context, manager *spec.Manager, options spec.Options) model {
	ctx, cancel := context.WithCancel(ctx)
	if options.Method == "" {
		options.Method = "auto"
	}
	if len(options.Tools) == 0 {
		options.Tools = []string{"claude", "codex"}
	}
	// Setup from the TUI always sends the language shown in its options. It
	// starts from the recorded one unless the command line chose another.
	languageChosen := options.SetDocumentLanguage
	options.SetDocumentLanguage = true
	reviewChosen := options.SetReview
	options.SetReview = true
	if options.Review == "" {
		options.Review = "spec"
	}
	input := textinput.New()
	input.Placeholder = "/path/to/project"
	input.SetValue(options.Project)
	input.CharLimit = 4096
	language := textinput.New()
	language.Placeholder = "empty follows the conversation"
	language.Prompt = ""
	language.CharLimit = 40
	m := model{languageChosen: languageChosen, reviewChosen: reviewChosen, ctx: ctx, cancel: cancel, manager: manager, options: options, project: input, language: language,
		spinner: spinner.New(spinner.WithSpinner(spinner.Line)), viewport: viewport.New(), help: help.New(),
		width: 100, height: 30, dark: true, message: "Reading workspace status..."}
	// Opening a valid workspace goes straight to its read-only overview.
	project, err := spec.Project(options.Project)
	if err != nil {
		m.screen = projectScreen
		m.project.Focus()
		m.message = ""
	} else {
		m.options.Project = project
		m.project.SetValue(project)
		m.busy = true
	}
	m.prefs = loadPreferences()
	m.resize()
	m.setInputStyles()
	m.setHelpStyles()
	return m
}

func (m model) Init() tea.Cmd {
	if m.screen == projectScreen {
		return tea.Batch(textinput.Blink, tea.RequestBackgroundColor)
	}
	return tea.Batch(m.perform("status", false, m.options), m.spinner.Tick, tea.RequestBackgroundColor)
}

func (m model) readWorkspace(project string) snapshot {
	status, err := m.manager.Status(project)
	commands := map[string]bool{}
	for _, name := range []string{"claude", "codex"} {
		_, lookupErr := m.manager.LookPath(name)
		commands[name] = lookupErr == nil
	}
	return snapshot{status: status, documents: spec.Audit(project), commands: commands, err: err}
}

func (m model) perform(operation string, preview bool, options spec.Options) tea.Cmd {
	return func() tea.Msg {
		result := resultMsg{operation: operation, preview: preview, options: options}
		switch operation {
		case "status":
			result.workspace = m.readWorkspace(options.Project)
			result.data, result.err = result.workspace.status, result.workspace.err
		case "audit":
			result.data = spec.Audit(options.Project)
		default:
			options.DryRun = preview
			result.data, result.code, result.err = m.manager.Execute(m.ctx, operation, options)
		}
		// Failed setup can have completed some steps. Refresh off the UI thread,
		// including after errors, so the overview reflects any partial writes.
		if operation != "status" && !preview {
			result.workspace = m.readWorkspace(options.Project)
		}
		return result
	}
}

// menu lists what the user can do now. The recommended action comes first.
// Agents start with spec run and rollback is a CLI command; the dashboard
// keeps setup, documents and an update offered only when one exists.
func (m model) menu() []item {
	setup := item{label: "Set up workspace", description: "Installs the SPEC plugin, pinned OpenSpec and the selected agent integrations. You review a plan before anything is written.", operation: "install", preview: true}
	documents := item{label: "Project documents", description: "Read and edit the project's brief, instructions, design, specs and OpenSpec changes.", operation: "documents"}
	tail := []item{
		{label: "Setup options", description: "Choose the agents, OpenSpec source, native setup, document language and design review used by the next setup.", operation: "options"},
		{label: "Switch project", description: "Opens another directory. The current project stays unchanged.", operation: "project"},
	}
	status := m.workspace.status
	if m.workspace.err != nil || status == nil || status["status"] == "not-installed" {
		return append([]item{setup, documents}, tail...)
	}
	setup.label = "Repair setup"
	setup.description = "Runs setup again for the plugin, OpenSpec and agents. You review a plan before anything is written."
	items := []item{}
	attention := overviewReport(m.workspace).tone != ""
	if attention {
		items = append(items, setup)
	}
	if m.workspace.update != "" {
		items = append(items, item{label: "Update to " + m.workspace.update, description: "A newer stable plugin release is available. You review a plan before anything changes; project documents and OpenSpec stay as they are.", operation: "update", preview: true})
	}
	items = append(items, documents)
	if !attention {
		items = append(items, setup)
	}
	return append(items, tail...)
}

func (m *model) resize() {
	l := m.layout()
	m.project.SetWidth(max(8, l.main-4))
	m.language.SetWidth(max(8, l.main-24))
	m.viewport.SetWidth(l.main)
	reserved := 2 // title and blank line
	if l.wide {
		reserved++
	}
	m.viewport.SetHeight(max(1, l.body-reserved))
	if m.screen == editScreen {
		m.editor.area.SetWidth(l.main)
		m.editor.area.SetHeight(max(1, l.body-reserved))
	}
	m.help.SetWidth(max(1, m.width-2))
	m.refreshContent(false)
}

func (m *model) refreshContent(top bool) {
	if m.screen == viewScreen {
		m.viewport.SetContent(m.renderDocument(m.docPath, m.docContent, m.viewport.Width()))
	} else {
		m.viewport.SetContent(m.report.render(m.viewport.Width(), m.palette()))
	}
	if top {
		m.viewport.GotoTop()
	}
}

func (m *model) show(s screen, r report) {
	m.screen, m.report = s, r
	m.refreshContent(true)
}

func (m *model) start(operation string, preview bool, options spec.Options) tea.Cmd {
	m.busy, m.messageError = true, false
	m.message = operationMessage(operation, preview)
	return tea.Batch(m.perform(operation, preview, options), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil
	case tea.BackgroundColorMsg:
		m.dark = msg.IsDark()
		m.setInputStyles()
		m.setHelpStyles()
		if m.screen == editScreen {
			m.editor.area.SetStyles(m.editorStyles())
		}
		m.refreshContent(false)
		return m, nil
	case resultMsg:
		return m.result(msg)
	case updateMsg:
		if msg.project == m.options.Project && m.workspace.status != nil && versionGreater(msg.latest, msg.current) {
			m.workspace.update = msg.latest
		}
		return m, nil
	case editorDoneMsg:
		m.busy = false
		m.message, m.messageError = "", false
		if msg.err != nil {
			m.message, m.messageError = "Your editor exited with an error: "+msg.err.Error(), true
		}
		return m.openDocument(m.docPath)
	case spinner.TickMsg:
		if m.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	if m.screen == projectScreen {
		var cmd tea.Cmd
		m.project, cmd = m.project.Update(msg)
		return m, cmd
	}
	if m.screen == optionsScreen && m.language.Focused() {
		var cmd tea.Cmd
		m.language, cmd = m.language.Update(msg)
		return m, cmd
	}
	if m.screen == editScreen {
		var cmd tea.Cmd
		m.editor, _, cmd = m.editor.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) result(msg resultMsg) (tea.Model, tea.Cmd) {
	m.busy = false
	if m.quitting {
		return m, tea.Quit
	}
	if msg.workspace.status != nil || msg.workspace.err != nil {
		m.workspace = msg.workspace
		if !m.languageChosen {
			m.options.DocumentLanguage = value(msg.workspace.status, "document_language", "")
		}
		if review, ok := msg.workspace.status["review"].(map[string]any); ok && !m.reviewChosen {
			m.options.Review = value(review, "mode", "spec")
		}
	}
	m.message, m.messageError = "", false
	switch {
	case msg.err != nil:
		m.show(reportScreen, errorReport(msg.operation, msg.err))
	case msg.preview && msg.operation == "update" && !versionGreater(value(msg.data, "selected", ""), value(msg.data, "current", "")):
		m.show(reportScreen, upToDateReport(msg.data))
	case msg.preview:
		m.pending = msg.operation
		m.pendingOptions = msg.options
		m.pendingOptions.ExpectedCurrent, _ = msg.data["baseline_digest"].(string)
		m.pendingOptions.ExpectedTarget, _ = msg.data["target_digest"].(string)
		// Execute the exact source/release displayed in the plan.
		if msg.operation == "update" {
			m.pendingOptions.To, _ = msg.data["selected"].(string)
			m.pendingOptions.Source, _ = msg.data["source"].(string)
		}
		m.show(planScreen, previewReport(msg.operation, msg.data, m.pendingOptions))
	case msg.operation == "status":
		m.cursor = 0
		m.show(homeScreen, report{})
		if m.workspace.err == nil && m.workspace.status["status"] == "installed" {
			return m, m.checkUpdate()
		}
	default:
		m.show(reportScreen, operationReport(msg.operation, msg.data, msg.code))
	}
	return m, nil
}

// checkUpdate asks the configured release source for its newest stable tag
// in the background. Failures stay silent: the offer simply does not appear.
func (m model) checkUpdate() tea.Cmd {
	project := m.options.Project
	return func() tea.Msg {
		current, latest, err := m.manager.LatestRelease(m.ctx, project)
		if err != nil {
			return nil
		}
		return updateMsg{project, current, latest}
	}
}

func (m model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// The editor owns Ctrl+C (vim leaves insert mode with it) so unsaved text
	// is never lost to an accidental quit.
	if m.screen == editScreen && !m.busy {
		return m.editKey(msg)
	}
	if key.Matches(msg, keys.Interrupt) {
		if m.busy {
			m.cancel()
			m.quitting = true
			m.message = "Cancelling; waiting for operation cleanup..."
			return m, nil
		}
		return m, tea.Quit
	}
	if m.busy {
		return m, nil
	}
	switch m.screen {
	case projectScreen:
		return m.projectKey(msg)
	case optionsScreen:
		return m.optionsKey(msg)
	case docsScreen:
		return m.docsKey(msg)
	case viewScreen:
		return m.viewKey(msg)
	case planScreen:
		switch {
		case key.Matches(msg, keys.Apply):
			m.screen = homeScreen
			return m, m.start(m.pending, false, m.pendingOptions)
		case key.Matches(msg, keys.Cancel):
			m.message, m.messageError = "Plan cancelled. Nothing was changed.", false
			m.show(homeScreen, report{})
			return m, nil
		}
		return m.scroll(msg)
	case reportScreen:
		if key.Matches(msg, keys.Back) {
			m.show(homeScreen, report{})
			return m, nil
		}
		return m.scroll(msg)
	}
	items := m.menu()
	m.cursor = min(m.cursor, len(items)-1)
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Refresh):
		return m, m.start("status", false, m.options)
	case key.Matches(msg, keys.Up):
		m.cursor = (m.cursor - 1 + len(items)) % len(items)
		m.message = ""
	case key.Matches(msg, keys.Down):
		m.cursor = (m.cursor + 1) % len(items)
		m.message = ""
	case key.Matches(msg, keys.Select):
		return m.activate(items[m.cursor])
	}
	return m, nil
}

func (m model) activate(selected item) (tea.Model, tea.Cmd) {
	m.message, m.messageError = "", false
	switch {
	case selected.operation == "options":
		m.screen = optionsScreen
		return m, nil
	case selected.operation == "documents":
		m.docs, m.screen = spec.ProjectDocuments(m.options.Project), docsScreen
		m.docCursor = min(m.docCursor, max(0, len(m.docs)-1))
		return m, nil
	case selected.operation == "project":
		m.screen = projectScreen
		return m, m.project.Focus()
	}
	return m, m.start(selected.operation, selected.preview, m.options)
}

func (m model) projectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		project, err := spec.Project(m.project.Value())
		if err != nil {
			m.message, m.messageError = err.Error(), true
			return m, nil
		}
		m.options.Project = project
		m.project.SetValue(project)
		m.project.Blur()
		m.workspace = snapshot{}
		m.show(homeScreen, report{})
		return m, m.start("status", false, m.options)
	case "esc":
		if m.workspace.status == nil && m.workspace.err == nil {
			return m, tea.Quit
		}
		m.project.SetValue(m.options.Project)
		m.project.Blur()
		m.screen = homeScreen
		m.message, m.messageError = "", false
		return m, nil
	}
	var cmd tea.Cmd
	m.project, cmd = m.project.Update(msg)
	return m, cmd
}

func (m model) optionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.language.Focused() {
		return m.languageKey(msg)
	}
	switch {
	case msg.String() == "esc" || msg.String() == "q":
		m.screen = homeScreen
		m.message, m.messageError = "", false
	case key.Matches(msg, keys.Up):
		m.optionCursor = (m.optionCursor + len(optionLabels) - 1) % len(optionLabels)
		m.message = ""
	case key.Matches(msg, keys.Down):
		m.optionCursor = (m.optionCursor + 1) % len(optionLabels)
		m.message = ""
	case key.Matches(msg, keys.Change) && m.optionCursor == languageOption:
		m.message, m.messageError = "", false
		m.language.SetValue(m.options.DocumentLanguage)
		m.language.CursorEnd()
		return m, m.language.Focus()
	case key.Matches(msg, keys.Change):
		m.toggleOption()
	}
	return m, nil
}

// languageKey edits the document language in place. Any language name is
// accepted; an empty value follows the conversation.
func (m model) languageKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		language, err := spec.DocumentLanguage(m.language.Value())
		if err != nil {
			m.message, m.messageError = "Use a language name such as English, Czech or Brazilian Portuguese, or leave it empty.", true
			return m, nil
		}
		m.options.DocumentLanguage, m.languageChosen = language, true
		m.language.Blur()
		m.message, m.messageError = "", false
		return m, nil
	case "esc":
		m.language.Blur()
		m.message, m.messageError = "", false
		return m, nil
	}
	var cmd tea.Cmd
	m.language, cmd = m.language.Update(msg)
	return m, cmd
}

func (m model) docsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc" || msg.String() == "q":
		m.screen = homeScreen
		m.message, m.messageError = "", false
	case len(m.docs) == 0:
	case key.Matches(msg, keys.Up):
		m.docCursor = (m.docCursor + len(m.docs) - 1) % len(m.docs)
		m.message = ""
	case key.Matches(msg, keys.Down):
		m.docCursor = (m.docCursor + 1) % len(m.docs)
		m.message = ""
	case key.Matches(msg, keys.Select):
		return m.openDocument(m.docs[m.docCursor].Path)
	case key.Matches(msg, keys.Edit):
		next, _ := m.openDocument(m.docs[m.docCursor].Path)
		if m = next.(model); m.screen == viewScreen {
			return m.editDocument()
		}
	}
	return m, nil
}

func (m model) viewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "backspace":
		m.screen = homeScreen
		if m.docs != nil {
			m.screen = docsScreen
		}
		m.message, m.messageError = "", false
		return m, nil
	case "e":
		return m.editDocument()
	case "g":
		m.viewport.GotoTop()
		return m, nil
	case "G":
		m.viewport.GotoBottom()
		return m, nil
	}
	return m.scroll(msg)
}

// openDocument reads a document fresh from disk and shows it rendered.
func (m model) openDocument(path string) (tea.Model, tea.Cmd) {
	content, digest, err := spec.ReadDocument(m.options.Project, path)
	if err != nil {
		m.message, m.messageError = err.Error(), true
		return m, nil
	}
	m.docPath, m.docContent, m.docDigest = path, content, digest
	m.screen = viewScreen
	m.refreshContent(true)
	return m, nil
}

func (m model) editDocument() (tea.Model, tea.Cmd) {
	m.message, m.messageError = "", false
	if m.prefs.Editor == "external" {
		cmd, err := externalEditor(filepath.Join(m.options.Project, filepath.FromSlash(m.docPath)))
		if err != nil {
			m.message, m.messageError = err.Error(), true
			return m, nil
		}
		cmd.Dir = m.options.Project
		m.busy = true
		m.message = "Your editor is using the terminal. Exit it to return."
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDoneMsg{err} })
	}
	l, reserved := m.layout(), 2
	if l.wide {
		reserved++
	}
	e, err := newEditor(m.docPath, m.docContent, m.docDigest, m.prefs.Editor == "vim", l.main, max(1, l.body-reserved), m.editorStyles())
	if err != nil {
		m.message, m.messageError = err.Error(), true
		return m, nil
	}
	m.editor, m.screen = e, editScreen
	return m, textarea.Blink
}

// externalEditor follows the usual convention: $VISUAL, then $EDITOR. The
// value may carry arguments, such as "code --wait".
func externalEditor(path string) (*exec.Cmd, error) {
	command := strings.TrimSpace(os.Getenv("VISUAL"))
	if command == "" {
		command = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if command == "" {
		return nil, fmt.Errorf("set $EDITOR or $VISUAL, or choose a built-in editor in Setup options")
	}
	fields := strings.Fields(command)
	return exec.Command(fields[0], append(fields[1:], path)...), nil
}

func (m model) editKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var action editorAction
	var cmd tea.Cmd
	m.editor, action, cmd = m.editor.Update(msg)
	switch action {
	case editorSave, editorForceSave:
		m.saveDocument(action == editorForceSave)
	case editorSaveQuit:
		if m.saveDocument(false) {
			return m.openDocument(m.docPath)
		}
	case editorQuit:
		return m.openDocument(m.docPath)
	}
	return m, cmd
}

// saveDocument writes only if the file is unchanged since it was opened,
// unless the user forced it with :w!.
func (m *model) saveDocument(force bool) bool {
	expected := m.editor.digest
	if force {
		expected = ""
	}
	value := m.editor.area.Value()
	digest, err := spec.WriteDocument(m.options.Project, m.editor.path, value, expected)
	if err != nil {
		hint := ""
		if m.editor.vim && !force {
			hint = " Use :w! to overwrite."
		}
		m.editor.say(err.Error()+hint, true)
		return false
	}
	m.editor.digest, m.editor.saved = digest, value
	m.docContent, m.docDigest = value, digest
	m.editor.say("Saved "+m.editor.path+".", false)
	return true
}

func (m model) scroll(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.viewport.ScrollUp(1)
	case "down", "j":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.PageUp()
	case "pgdown", "space":
		m.viewport.PageDown()
	case "home":
		m.viewport.GotoTop()
	case "end":
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m *model) toggleOption() {
	m.message, m.messageError = "", false
	switch m.optionCursor {
	case 0, 1:
		name := []string{"claude", "codex"}[m.optionCursor]
		tools := []string{}
		for _, current := range m.options.Tools {
			if current != name {
				tools = append(tools, current)
			}
		}
		if len(tools) == len(m.options.Tools) {
			tools = append(tools, name)
		}
		if len(tools) == 0 {
			m.message, m.messageError = "Keep at least one agent selected.", true
			return
		}
		m.options.Tools = tools
	case 2:
		if m.options.OpenSpecBin != "" {
			m.message = "Using --openspec-bin. Change this executable through the CLI."
			return
		}
		switch m.options.Method {
		case "auto":
			m.options.Method = "binary"
		case "binary":
			m.options.Method = "pnpm"
		case "pnpm":
			m.options.Method = "npm"
		default:
			m.options.Method = "auto"
		}
	case 3:
		m.options.NoRegister = !m.options.NoRegister
	case reviewOption:
		m.options.Review, m.reviewChosen = map[string]string{"spec": "egm", "egm": "spec"}[m.options.Review], true
	default:
		for i, choice := range editorChoices {
			if choice == m.prefs.Editor {
				m.prefs.Editor = editorChoices[(i+1)%len(editorChoices)]
				break
			}
		}
		if err := savePreferences(m.prefs); err != nil {
			m.message, m.messageError = "Editor changed for this session; it could not be saved: "+err.Error(), true
		}
	}
}

var optionLabels = []string{"Claude Code", "Codex", "OpenSpec source", "Native plugin setup", "Document language", "Design review", "Editor"}

const (
	languageOption = 4
	reviewOption   = 5
)

func operationMessage(operation string, preview bool) string {
	if preview {
		if operation == "update" {
			return "Checking stable plugin releases..."
		}
		return "Preparing a change plan..."
	}
	switch operation {
	case "status":
		return "Reading workspace status..."
	case "audit":
		return "Looking for project documents..."
	case "install":
		return "Setting up plugin, OpenSpec and agents..."
	case "update":
		return "Updating the managed plugin..."
	}
	return "Working..."
}

func Run(ctx context.Context, manager *spec.Manager, options spec.Options, in io.Reader, out io.Writer) error {
	model := newModel(ctx, manager, options)
	defer model.cancel()
	_, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}

func selectedTool(tools []string, name string) bool {
	for _, tool := range tools {
		if tool == name {
			return true
		}
	}
	return false
}
func agentName(name string) string {
	if name == "claude" {
		return "Claude Code"
	}
	if name == "codex" {
		return "Codex"
	}
	return strings.TrimSpace(name)
}
