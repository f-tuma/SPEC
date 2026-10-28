package spec

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Options struct {
	Project                         string
	Tools                           []string
	Cache                           string
	Method                          string
	OpenSpecBin                     string
	DryRun, NoRegister, Check       bool
	Source, To                      string
	ExpectedCurrent, ExpectedTarget string
	// DocumentLanguage replaces the recorded language only when
	// SetDocumentLanguage is true; an empty value follows the conversation.
	DocumentLanguage    string
	SetDocumentLanguage bool
	// Review replaces the recorded review procedure (spec or egm) only when
	// SetReview is true.
	Review    string
	SetReview bool
}

func Tools(value string) ([]string, error) {
	tools := []string{}
	for _, tool := range strings.Split(value, ",") {
		if tool != "claude" && tool != "codex" {
			return nil, fmt.Errorf("tools accepts claude, codex or claude,codex")
		}
		if !slices.Contains(tools, tool) {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func lockProject(project string) (func(), error) {
	folder, err := stateFolder(project, true)
	if err != nil {
		return nil, err
	}
	path, err := Within(folder, "operation.lock")
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot acquire SPEC operation lock; another operation may be active. After an interrupted process, inspect %s: %w", path, err)
	}
	// The token lets unlock remove only its own lock: uninstall moves .spec
	// aside, and another operation may already hold a lock in a new .spec.
	token := make([]byte, 16)
	rand.Read(token)
	owned := fmt.Sprintf("pid=%d\ntoken=%x\n", os.Getpid(), token)
	fmt.Fprint(f, owned)
	f.Close()
	return func() {
		if data, err := readBytes(path, 4096); err == nil && string(data) == owned {
			os.Remove(path)
		}
	}, nil
}

// Execute is shared by the JSON CLI and the TUI. A preview follows the same preflight.
func (m *Manager) Execute(ctx context.Context, operation string, options Options) (map[string]any, int, error) {
	project, err := Project(options.Project)
	if err != nil {
		return nil, 2, err
	}
	options.Project = project
	if operation == "status" {
		result, err := m.Status(project)
		return result, 0, err
	}
	if operation != "install" && operation != "update" && operation != "rollback" {
		return nil, 2, fmt.Errorf("unknown operation: %s", operation)
	}
	if options.Cache == "" {
		options.Cache = DefaultCache()
	}
	if options.Method == "" {
		options.Method = "auto"
	}
	if len(options.Tools) == 0 {
		options.Tools = []string{"claude", "codex"}
	}
	if operation == "install" {
		if _, err = m.initialize(ctx, options, true); err != nil {
			return nil, 2, err
		}
		if _, err = m.installPlugin(project, m.Plugin, nil, true); err != nil {
			return nil, 2, err
		}
	} else {
		state, err := LoadState(project)
		if err != nil {
			return nil, 2, err
		}
		if state == nil {
			return nil, 2, fmt.Errorf("run spec install for this project first")
		}
		if _, err = verifyRelease(project, state.Current); err != nil {
			return nil, 2, err
		}
	}
	if !options.DryRun && !options.Check {
		unlock, err := lockProject(project)
		if err != nil {
			return nil, 2, err
		}
		defer unlock()
		if options.ExpectedCurrent != "" {
			state, err := LoadState(project)
			if err != nil {
				return nil, 2, err
			}
			if baselineDigest(state) != options.ExpectedCurrent {
				return nil, 2, fmt.Errorf("installation changed since preview; create a fresh preview")
			}
		}
	}
	if operation == "install" {
		plugin, err := m.installPlugin(project, m.Plugin, nil, options.DryRun)
		if err != nil {
			return nil, 2, err
		}
		openSpec, err := m.initialize(ctx, options, options.DryRun)
		if err != nil {
			return nil, 2, err
		}
		clients, review := map[string]Client{}, map[string]any{}
		if !options.DryRun {
			if clients, review, err = m.setupAgents(ctx, project, options, options.Tools); err != nil {
				return nil, 2, err
			}
		}
		status := "setup-complete"
		code := pendingCode(clients, review)
		if code == 3 {
			status = "client-install-pending"
		}
		state, _ := LoadState(project)
		if options.DryRun {
			status = "planned"
			review = plannedReview(project, state, options, options.Tools)
		}
		return addCodexGlobal(project, map[string]any{"status": status, "cli_version": Version, "baseline_digest": baselineDigest(state), "plugin": plugin, "openspec": openSpec, "clients": clients, "review": review, "client_registration_requested": !options.NoRegister, "native_invocation_verified": false}), code, nil
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, 2, err
	}
	if state == nil {
		return nil, 2, fmt.Errorf("run spec install for this project first")
	}
	if _, err = verifyRelease(project, state.Current); err != nil {
		return nil, 2, err
	}
	var result map[string]any
	if operation == "rollback" {
		if len(state.History) == 0 {
			return nil, 2, fmt.Errorf("no previous managed plugin version is available")
		}
		previous := state.History[len(state.History)-1]
		if options.ExpectedTarget != "" && options.ExpectedTarget != previous.Digest {
			return nil, 2, fmt.Errorf("rollback target changed since preview; create a fresh preview")
		}
		if _, err = verifyRelease(project, previous); err != nil {
			return nil, 2, err
		}
		if options.DryRun {
			return map[string]any{"status": "planned", "plugin_version": previous.Version, "baseline_digest": state.Current.Digest, "target_digest": previous.Digest, "review": plannedReview(project, state, options, stringList(state.OpenSpec["tools"]))}, 0, nil
		}
		staleClients(state)
		state.History = append(state.History[:len(state.History)-1], state.Current)
		state.Current = previous
		if err = m.commitState(project, state); err != nil {
			return nil, 2, err
		}
		result = map[string]any{"status": "rolled-back", "plugin_version": previous.Version, "project_documents_changed": false}
	} else {
		source := options.Source
		if source == "" {
			source = state.Source
		}
		tags, err := m.Versions(ctx, source)
		if err != nil {
			return nil, 2, err
		}
		if len(tags) == 0 {
			return nil, 2, fmt.Errorf("the source has no stable release tags")
		}
		tag := options.To
		if tag == "" {
			tag = tags[len(tags)-1]
		}
		if !slices.Contains(tags, tag) {
			return nil, 2, fmt.Errorf("the requested release tag was not found")
		}
		if options.Check {
			return map[string]any{"status": "checked", "current": state.Current.Version, "available": tags, "selected": tag}, 0, nil
		}
		if options.DryRun {
			return map[string]any{"status": "planned", "current": state.Current.Version, "selected": tag, "source": source, "baseline_digest": state.Current.Digest, "review": plannedReview(project, state, options, stringList(state.OpenSpec["tools"]))}, 0, nil
		}
		if options.To == "" && compareVersions(tag, state.Current.Version) < 0 {
			result := map[string]any{"status": "already-newer", "current": state.Current.Version, "latest_source_release": tag}
			if !options.SetReview {
				return result, 0, nil
			}
			// The plugin stays, but a chosen review procedure still applies.
			clients, review, err := m.setupAgents(ctx, project, options, stringList(state.OpenSpec["tools"]))
			if err != nil {
				return nil, 2, err
			}
			result["clients"], result["review"] = clients, review
			return result, pendingCode(clients, review), nil
		}
		temp, err := os.MkdirTemp("", "spec-update-")
		if err != nil {
			return nil, 2, err
		}
		defer os.RemoveAll(temp)
		plugin, commit, err := m.fetchPlugin(ctx, source, tag, filepath.Join(temp, "source"))
		if err != nil {
			return nil, 2, err
		}
		result, err = m.installPlugin(project, plugin, map[string]string{"git": source, "tag": tag, "commit": commit}, false)
		if err != nil {
			return nil, 2, err
		}
		state, err = LoadState(project)
		if err != nil {
			return nil, 2, err
		}
		result["openspec_and_project_documents_changed"] = false
	}
	clients, review, err := m.setupAgents(ctx, project, options, stringList(state.OpenSpec["tools"]))
	if err != nil {
		return nil, 2, err
	}
	result["clients"], result["review"] = clients, review
	return addCodexGlobal(project, result), pendingCode(clients, review), nil
}

// setupAgents records the chosen review procedure, then sets up the agents
// and the review plugins unless agent setup is skipped.
func (m *Manager) setupAgents(ctx context.Context, project string, options Options, tools []string) (map[string]Client, map[string]any, error) {
	if options.SetReview {
		if err := setReview(project, options.Review); err != nil {
			return nil, nil, err
		}
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, nil, err
	}
	review := map[string]any{"mode": reviewOf(state), "plugins": map[string]Extension{}, "notes": []string{}}
	if options.NoRegister {
		if review["mode"] == "egm" {
			review["notes"] = []string{"Agent setup was skipped, so the EGM plugin is not set up yet. Run setup without --no-register to install it."}
		}
		return map[string]Client{}, review, nil
	}
	clients, err := m.registerClients(ctx, project, tools)
	if err != nil {
		return nil, nil, err
	}
	plugins, notes, err := m.setupReview(ctx, project, tools)
	if err != nil {
		return nil, nil, err
	}
	review["plugins"] = plugins
	if notes != nil {
		review["notes"] = notes
	}
	return clients, review, nil
}

// plannedReview previews the review procedure an install would set up.
func plannedReview(project string, state *State, options Options, tools []string) map[string]any {
	mode := reviewOf(state)
	if options.SetReview {
		mode = options.Review
	}
	plan := []string{}
	if !options.NoRegister {
		plan = ReviewPlan(project, state, mode, tools)
	}
	return map[string]any{"mode": mode, "planned": plan}
}

func pendingCode(clients map[string]Client, review map[string]any) int {
	for _, client := range clients {
		if client.Status == "pending" {
			return 3
		}
	}
	plugins, _ := review["plugins"].(map[string]Extension)
	for _, plugin := range plugins {
		if plugin.Status == "pending" {
			return 3
		}
	}
	return 0
}

func baselineDigest(state *State) string {
	if state == nil {
		return "not-installed"
	}
	return state.Current.Digest
}
func stringList(value any) []string {
	if list, ok := value.([]string); ok {
		return list
	}
	list := []string{}
	if values, ok := value.([]any); ok {
		for _, v := range values {
			s, ok := v.(string)
			if !ok || s == "" {
				return nil
			}
			list = append(list, s)
		}
	}
	return list
}

func (m *Manager) OpenSpec(ctx context.Context, project string, args []string) (map[string]any, error) {
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("OpenSpec has not been initialized by SPEC")
	}
	argv := stringList(state.OpenSpec["command"])
	if len(argv) == 0 {
		return nil, fmt.Errorf("OpenSpec command is missing")
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("pass OpenSpec arguments after --")
	}
	output, err := m.Run(ctx, project, nil, append(argv, args...)...)
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": "completed", "output": output}, nil
}
