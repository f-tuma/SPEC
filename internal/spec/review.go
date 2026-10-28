package spec

import (
	"context"
	"fmt"
	"strings"
)

// The official Elephant-Goldfish Model plugin is the optional design review
// procedure. Each client installs it from Google's repository at a pinned
// tag, like OpenSpec, and SPEC records what it added itself so uninstall
// removes only that. Claude Code installs it for the project; Codex has no
// project scope, so its EGM plugin is enabled in every Codex session.
const (
	EGMRepository  = "google/elephant-goldfish-model"
	EGMRef         = "v0.1.0"
	egmPlugin      = "egm@egm"
	egmMarketplace = "egm"
)

// Extension records a third-party plugin set up for the review procedure.
// Status is installed (SPEC added the plugin), present (it was already
// there), pending, or kept: SPEC review is selected again and the record
// only remembers what SPEC added that other projects may share.
// Additive to schema 1.
type Extension struct {
	Status           string `json:"status"`
	Reason           string `json:"reason,omitempty"`
	Ref              string `json:"ref,omitempty"`
	Added            bool   `json:"added,omitempty"`
	MarketplaceAdded bool   `json:"marketplace_added,omitempty"`
}

// ReviewMode validates a CLI or TUI value: spec is SPEC's own fresh-context
// review, egm the official EGM plugin.
func ReviewMode(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "spec":
		return "spec", nil
	case "egm":
		return "egm", nil
	}
	return "", fmt.Errorf("review accepts spec or egm")
}

func reviewOf(state *State) string {
	if state == nil || state.Review == "" {
		return "spec"
	}
	return state.Review
}

// egmState is what one client's own records say about EGM.
type egmState struct {
	installed, marketplace bool
	ref                    string
	// reason explains why SPEC cannot set EGM up: unreadable records, a
	// marketplace named egm from another repository, a disabled plugin.
	reason string
}

// sameRepository compares a marketplace source with EGM's GitHub repository,
// in the forms the clients record: owner/repo, HTTPS or SSH URLs.
func sameRepository(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	for _, prefix := range []string{"https://", "http://", "ssh://", "git@", "www."} {
		source = strings.TrimPrefix(source, prefix)
	}
	source = strings.TrimPrefix(strings.TrimPrefix(source, "github.com/"), "github.com:")
	source = strings.TrimSuffix(strings.TrimSuffix(source, "/"), ".git")
	return source == EGMRepository
}

func text(value any) string {
	s, _ := value.(string)
	return s
}

// egmRecords reads a client's records for EGM. It never writes them.
func egmRecords(project, tool string) egmState {
	var st egmState
	if tool == "claude" {
		entries, unknown := claudeInstalls(egmPlugin)
		if unknown {
			return egmState{reason: "Claude Code's installed_plugins.json could not be read or parsed"}
		}
		// A user-scope install already serves every project.
		for _, entry := range entries {
			st.installed = st.installed || entry.Scope == "user" || (entry.Scope == "project" || entry.Scope == "local") && samePath(entry.ProjectPath, project)
		}
		entry, known, unknown := claudeMarketplace(egmMarketplace)
		if unknown {
			return egmState{reason: "Claude Code's known_marketplaces.json could not be read or parsed"}
		}
		source, _ := entry["source"].(map[string]any)
		st.marketplace, st.ref = known, text(source["ref"])
		if known && !sameRepository(text(source["repo"])) && !sameRepository(text(source["url"])) {
			st.reason = "a different marketplace named egm is configured; SPEC leaves it as it is"
		} else if st.installed && claudeDisabled(project, egmPlugin) {
			st.reason = "Claude Code settings disable egm@egm; enable it in /plugin or the settings file"
		}
		return st
	}
	settings, unknown := codexState()
	if unknown {
		return egmState{reason: "Codex's config.toml could not be read or parsed"}
	}
	plugin, listed := settings.Plugins[egmPlugin]
	marketplace, known := settings.Marketplaces[egmMarketplace]
	st.installed, st.marketplace, st.ref = listed, known, text(marketplace["ref"])
	if known && !sameRepository(text(marketplace["source"])) {
		st.reason = "a different marketplace named egm is configured; SPEC leaves it as it is"
	} else if enabled, ok := plugin["enabled"].(bool); listed && ok && !enabled {
		st.reason = "Codex lists egm@egm but it is disabled; enable it in Codex's config.toml"
	}
	return st
}

// ReviewPlan describes what setup does for the review procedure, from the
// clients' current records. It only reads them.
func ReviewPlan(project string, state *State, review string, tools []string) []string {
	plan := []string{}
	if review != "egm" {
		if state != nil && changing(state) {
			steps, notes := egmRemoval(project, state)
			for _, s := range steps {
				plan = append(plan, s.label)
			}
			plan = append(plan, notes...)
		}
		return plan
	}
	for _, tool := range tools {
		name, where := "Claude Code", "for this project"
		if tool == "codex" {
			name, where = "Codex", "for every Codex session"
		}
		st := egmRecords(project, tool)
		switch {
		case st.reason != "":
			plan = append(plan, name+": "+st.reason+"; EGM setup stays pending")
		case st.installed:
			plan = append(plan, name+": egm@egm is already installed; it is kept as it is")
		case st.marketplace:
			plan = append(plan, name+": install egm@egm "+where+" from the existing egm marketplace")
		default:
			plan = append(plan, name+": install egm@egm "+where+" from "+EGMRepository+" "+EGMRef)
		}
	}
	return plan
}

// changing reports whether recorded EGM setup still has to be undone after
// SPEC review was selected again.
func changing(state *State) bool {
	for _, record := range state.Extensions {
		if record.Status != "kept" {
			return true
		}
	}
	return false
}

// setReview records the review procedure chosen for the project.
func setReview(project, review string) error {
	state, err := LoadState(project)
	if err != nil || state == nil {
		return err
	}
	if review == "spec" {
		review = ""
	}
	if state.Review == review {
		return nil
	}
	state.Review = review
	return saveState(project, state)
}

// setupReview installs the EGM plugin for the selected agents when the
// project uses EGM review. When it does not, it removes the Claude Code
// project plugin SPEC added earlier and names what stays elsewhere.
func (m *Manager) setupReview(ctx context.Context, project string, tools []string) (map[string]Extension, []string, error) {
	state, err := LoadState(project)
	if err != nil || state == nil {
		return nil, nil, err
	}
	if state.Extensions == nil {
		state.Extensions = map[string]Extension{}
	}
	if state.Review != "egm" {
		if !changing(state) {
			return state.Extensions, nil, nil
		}
		steps, notes := egmRemoval(project, state)
		for _, s := range steps {
			if err := m.runStep(ctx, project, s); err != nil {
				record := state.Extensions["egm:claude"]
				record.Status, record.Reason = "pending", "SPEC review is selected, but removing EGM failed: "+err.Error()
				state.Extensions["egm:claude"] = record
				return state.Extensions, notes, saveState(project, state)
			}
		}
		// Keep what SPEC added that other projects may share, so uninstall
		// still names it and a later EGM setup recognizes it.
		for key, record := range state.Extensions {
			record.Added = record.Added && key == "egm:codex"
			record.Status, record.Reason = "kept", ""
			if !record.Added && !record.MarketplaceAdded {
				delete(state.Extensions, key)
				continue
			}
			state.Extensions[key] = record
		}
		return state.Extensions, notes, saveState(project, state)
	}
	for _, tool := range tools {
		key := "egm:" + tool
		state.Extensions[key] = m.installEGM(ctx, project, tool, state.Extensions[key])
		// Save after each agent so a crash never forgets what SPEC added.
		if err := saveState(project, state); err != nil {
			return nil, nil, err
		}
	}
	return state.Extensions, nil, nil
}

func (m *Manager) installEGM(ctx context.Context, project, tool string, record Extension) Extension {
	record.Reason = ""
	executable, err := m.LookPath(tool)
	if err != nil {
		record.Status, record.Reason = "pending", tool+" CLI is not installed/on PATH"
		return record
	}
	st := egmRecords(project, tool)
	if st.reason != "" {
		record.Status, record.Reason = "pending", st.reason
		return record
	}
	if !st.marketplace {
		argv := []string{executable, "plugin", "marketplace", "add", EGMRepository + "#" + EGMRef}
		if tool == "codex" {
			argv = []string{executable, "plugin", "marketplace", "add", EGMRepository, "--ref", EGMRef}
		}
		if _, err = m.Run(ctx, project, nil, argv...); err != nil {
			record.Status, record.Reason = "pending", err.Error()
			return record
		}
		record.MarketplaceAdded, st.ref = true, EGMRef
	}
	record.Ref = st.ref
	if !st.installed {
		argv := []string{executable, "plugin", "install", egmPlugin, "--scope", "project"}
		if tool == "codex" {
			argv = []string{executable, "plugin", "add", egmPlugin}
		}
		if _, err = m.Run(ctx, project, nil, argv...); err != nil {
			record.Status, record.Reason = "pending", err.Error()
			return record
		}
		record.Added = true
	}
	record.Status = "present"
	if record.Added {
		record.Status = "installed"
	}
	return record
}

type step struct {
	tool, label string
	argv        []string
}

func (m *Manager) runStep(ctx context.Context, project string, s step) error {
	executable, err := m.LookPath(s.tool)
	if err != nil {
		return fmt.Errorf("%s CLI is not on PATH; run `%s %s` where it is installed", s.tool, s.tool, strings.Join(s.argv, " "))
	}
	_, err = m.Run(ctx, project, nil, append([]string{executable}, s.argv...)...)
	// The project may enable a plugin Claude Code no longer has installed.
	if err != nil && s.argv[1] == "uninstall" && strings.Contains(err.Error(), "not found in installed plugins") {
		return nil
	}
	return err
}

// egmRemoval plans removing what SPEC added for EGM. Only the Claude Code
// project plugin is removed: the egm marketplace and Codex's plugin are
// shared by every project, so they stay and the notes name the commands.
func egmRemoval(project string, state *State) (steps []step, notes []string) {
	claude, codex := state.Extensions["egm:claude"], state.Extensions["egm:codex"]
	if claude.Added && claudeProjectInstall(project, egmPlugin) {
		steps = append(steps, step{"claude", "Remove the Claude Code plugin egm@egm from this project", []string{"plugin", "uninstall", egmPlugin, "--scope", "project"}})
	}
	if claude.MarketplaceAdded {
		notes = append(notes, "Claude Code keeps the egm marketplace SPEC added, which other projects can use. Remove it with claude plugin marketplace remove egm.")
	}
	commands := []string{}
	if codex.Added {
		commands = append(commands, "codex plugin remove egm@egm")
	}
	if codex.MarketplaceAdded {
		commands = append(commands, "codex plugin marketplace remove egm")
	}
	if len(commands) != 0 {
		notes = append(notes, "Codex keeps the EGM plugin SPEC installed for every session, which other projects can use. Remove it with "+strings.Join(commands, " && ")+".")
	}
	return steps, notes
}

// reviewStatus checks the EGM setup against the clients' records and the
// project's agents. An agent EGM was never set up for is pending too.
func reviewStatus(project string, state *State) map[string]Extension {
	result := map[string]Extension{}
	for key, record := range state.Extensions {
		if state.Review == "egm" && record.Status != "pending" {
			if st := egmRecords(project, strings.TrimPrefix(key, "egm:")); st.reason != "" {
				record.Status, record.Reason = "pending", st.reason
			} else if !st.installed {
				record.Status, record.Reason = "pending", "the EGM plugin is no longer installed; run setup again"
			}
		}
		result[key] = record
	}
	if state.Review == "egm" {
		for _, tool := range stringList(state.OpenSpec["tools"]) {
			if _, ok := result["egm:"+tool]; !ok {
				result["egm:"+tool] = Extension{Status: "pending", Reason: "EGM is not set up for this agent yet; run setup"}
			}
		}
	}
	return result
}

// projectChecks returns the check commands the project mapping records,
// such as fast, full and ui. Agents maintain them; SPEC only reports them.
func projectChecks(project string) map[string][]string {
	config := MappingConfig{Checks: map[string][]string{}}
	if inputJSON(project, Mapping, &config) != nil || validateMapping(config) != nil {
		return map[string][]string{}
	}
	return config.Checks
}
