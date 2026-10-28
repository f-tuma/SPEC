package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var fingerprint = regexp.MustCompile(`^[a-f0-9]{64}$`)
var directoryName = regexp.MustCompile(`^\d+\.\d+\.\d+-[a-f0-9]{12}$`)

type Release struct {
	Version   string            `json:"version"`
	Files     map[string]string `json:"files"`
	Digest    string            `json:"digest"`
	Directory string            `json:"directory"`
	SourceRef any               `json:"source_ref"`
}
type State struct {
	SchemaVersion int               `json:"schema_version"`
	Current       Release           `json:"current"`
	History       []Release         `json:"history"`
	Source        string            `json:"source"`
	OpenSpec      map[string]any    `json:"openspec"`
	Clients       map[string]Client `json:"clients"`
	// DocumentLanguage is additive to schema 1; older receipts follow the conversation.
	DocumentLanguage string `json:"document_language,omitempty"`
	// Review is "" for SPEC's own review or "egm" for the official EGM plugin;
	// Extensions records the plugins set up for it. Both additive to schema 1.
	Review     string               `json:"review,omitempty"`
	Extensions map[string]Extension `json:"extensions,omitempty"`
}
type Client struct {
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Output             string `json:"output,omitempty"`
	Registered         bool   `json:"marketplace_registered"`
	InvocationVerified bool   `json:"invocation_verified"`
	PluginVersion      string `json:"plugin_version,omitempty"`
	PluginDigest       string `json:"plugin_digest,omitempty"`
	Materialized       bool   `json:"materialized,omitempty"`
	// Files records the skill files SPEC wrote into the project for Codex,
	// by project path and SHA-256. Additive to schema 1.
	Files map[string]string `json:"project_files,omitempty"`
}

func MarketplaceName(project string) string { return "spec-" + hash([]byte(project))[:10] }

func stateFolder(project string, create bool) (string, error) {
	folder, err := Within(project, ".spec")
	if err != nil {
		return "", err
	}
	marker, err := Within(folder, "owner.json")
	if err != nil {
		return "", err
	}
	if exists(folder) {
		var owner struct {
			Owner string `json:"owner"`
		}
		if err := readJSON(marker, &owner); err != nil || owner.Owner != "SPEC" {
			return "", fmt.Errorf("existing .spec directory is not owned by SPEC; nothing was overwritten")
		}
	} else if create {
		if err := os.Mkdir(folder, 0755); err != nil {
			return "", err
		}
		if err := atomicJSON(marker, map[string]any{"owner": "SPEC", "schema_version": 1}); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(folder, ".gitignore"), []byte("*\n"), 0644); err != nil {
			return "", err
		}
	}
	return folder, nil
}

func LoadState(project string) (*State, error) {
	folder, err := stateFolder(project, false)
	if err != nil {
		return nil, err
	}
	path, err := Within(folder, "installation.json")
	if err != nil {
		return nil, err
	}
	if !exists(path) {
		return nil, nil
	}
	var state State
	if err := readJSON(path, &state); err != nil {
		return nil, fmt.Errorf("invalid SPEC installation state: %w", err)
	}
	if state.SchemaVersion != 1 || state.History == nil || state.Source == "" || state.Clients == nil {
		return nil, fmt.Errorf("invalid SPEC installation state")
	}
	for _, release := range append([]Release{state.Current}, state.History...) {
		if !semver.MatchString(release.Version) || !fingerprint.MatchString(release.Digest) || len(release.Files) == 0 {
			return nil, fmt.Errorf("invalid installed plugin fingerprint/version")
		}
		if _, err := releasePath(folder, release); err != nil {
			return nil, err
		}
		for rel, digest := range release.Files {
			if _, err := Within(folder, rel); err != nil {
				return nil, err
			}
			if !fingerprint.MatchString(digest) {
				return nil, fmt.Errorf("invalid file hash")
			}
		}
	}
	for _, client := range state.Clients {
		for rel, digest := range client.Files {
			if !strings.HasPrefix(rel, projectSkills) || !fingerprint.MatchString(digest) {
				return nil, fmt.Errorf("invalid SPEC installation state: unexpected project file %q", rel)
			}
			if _, err := Within(filepath.Dir(folder), rel); err != nil {
				return nil, err
			}
		}
	}
	return &state, nil
}

func saveState(project string, state *State) error {
	folder, err := stateFolder(project, false)
	if err != nil {
		return err
	}
	path, err := Within(folder, "installation.json")
	if err != nil {
		return err
	}
	return atomicJSON(path, state)
}

func inventory(source fs.FS) (map[string]string, error) {
	files := map[string]string{}
	err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == ".git" || entry.Name() == "__pycache__" {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".pyc") {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink in plugin: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 16<<20 {
			return fmt.Errorf("unsupported plugin resource: %s", path)
		}
		data, err := fs.ReadFile(source, path)
		if err != nil {
			return err
		}
		files[path] = hash(data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("plugin has no files")
	}
	return files, nil
}

func pluginInfo(source fs.FS) (Release, error) {
	var release Release
	for _, rel := range []string{"plugin.json", ".claude-plugin/plugin.json", ".codex-plugin/plugin.json"} {
		data, err := fs.ReadFile(source, rel)
		if err != nil {
			return release, err
		}
		var manifest struct{ Name, Version string }
		if err = json.Unmarshal(data, &manifest); err != nil {
			return release, err
		}
		if manifest.Name != "spec" || !semver.MatchString(manifest.Version) {
			return release, fmt.Errorf("expected a stable SPEC plugin")
		}
		if release.Version != "" && release.Version != manifest.Version {
			return release, fmt.Errorf("plugin manifests disagree")
		}
		release.Version = manifest.Version
	}
	skills, err := fs.Glob(source, "skills/*/SKILL.md")
	if err != nil || len(skills) == 0 {
		return release, fmt.Errorf("plugin has no skills")
	}
	files, err := inventory(source)
	if err != nil {
		return release, err
	}
	data, _ := json.Marshal(files)
	release.Files = files
	release.Digest = hash(data)
	release.Directory = release.Version + "-" + release.Digest[:12]
	return release, nil
}

func releasePath(folder string, release Release) (string, error) {
	if len(release.Digest) < 12 || !directoryName.MatchString(release.Directory) || release.Directory != release.Version+"-"+release.Digest[:12] {
		return "", fmt.Errorf("invalid installed release directory")
	}
	return Within(folder, "marketplace/versions/"+release.Directory+"/plugin")
}
func verifyRelease(project string, release Release) (string, error) {
	folder, err := stateFolder(project, false)
	if err != nil {
		return "", err
	}
	path, err := releasePath(folder, release)
	if err != nil {
		return "", err
	}
	files, err := inventory(os.DirFS(path))
	if err != nil || !reflect.DeepEqual(files, release.Files) {
		return "", fmt.Errorf("managed plugin files have local changes or are missing; preserve/resolve them before updating")
	}
	return path, nil
}

func (m *Manager) installPlugin(project string, source fs.FS, sourceRef any, dry bool) (map[string]any, error) {
	candidate, err := pluginInfo(source)
	if err != nil {
		return nil, err
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	folder, err := stateFolder(project, false)
	if err != nil {
		return nil, err
	}
	if state != nil {
		if _, err := verifyRelease(project, state.Current); err != nil {
			return nil, err
		}
	}
	if dry {
		return map[string]any{"status": "planned", "plugin_version": candidate.Version, "managed_directory": folder}, nil
	}
	folder, err = stateFolder(project, true)
	if err != nil {
		return nil, err
	}
	target, err := releasePath(folder, candidate)
	if err != nil {
		return nil, err
	}
	if exists(target) {
		if _, err = verifyRelease(project, candidate); err != nil {
			return nil, err
		}
	} else {
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, err
		}
		stage, err := os.MkdirTemp(filepath.Dir(target), ".spec-stage-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(stage)
		staged := filepath.Join(stage, "plugin")
		if err = os.Mkdir(staged, 0755); err != nil {
			return nil, err
		}
		for rel, digest := range candidate.Files {
			data, err := fs.ReadFile(source, rel)
			if err != nil {
				return nil, err
			}
			if hash(data) != digest {
				return nil, fmt.Errorf("plugin changed during copy")
			}
			path, err := Within(staged, rel)
			if err != nil {
				return nil, err
			}
			if err = atomicFile(path, data, 0644); err != nil {
				return nil, err
			}
		}
		if err = os.Rename(staged, target); err != nil {
			return nil, err
		}
	}
	candidate.SourceRef = sourceRef
	if state == nil {
		state = &State{SchemaVersion: 1, History: []Release{}, Source: Repository, Clients: map[string]Client{}}
	} else if state.Current.Digest != candidate.Digest {
		staleClients(state)
		state.History = append(state.History, state.Current)
	}
	state.Current = candidate
	if ref, ok := sourceRef.(map[string]string); ok && ref["git"] != "" {
		state.Source = ref["git"]
	}
	if err = m.commitState(project, state); err != nil {
		return nil, err
	}
	return map[string]any{"status": "installed", "plugin_version": candidate.Version, "marketplace": filepath.Join(folder, "marketplace")}, nil
}

func (m *Manager) Status(project string) (map[string]any, error) {
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return map[string]any{"status": "not-installed", "cli_version": Version, "project": project}, nil
	}
	managed := "unchanged"
	if _, err = verifyRelease(project, state.Current); err != nil {
		managed = err.Error()
	}
	previous := []string{}
	for _, release := range state.History {
		previous = append(previous, release.Version)
	}
	catalogs := "unchanged"
	status := "installed"
	if err = verifyCatalogs(project, state); err != nil {
		catalogs = err.Error()
		status = "inconsistent"
	}
	clients := map[string]Client{}
	for name, client := range state.Clients {
		if client.Status == "installed-by-client" && client.PluginDigest == "" {
			client.Status = "version-unverified"
			client.Reason = "legacy installation did not record the client's plugin version"
		}
		// Earlier versions registered a global Codex plugin instead of
		// project skills. Once that plugin is gone, Codex has no SPEC skills.
		if name == "codex" && len(client.Files) == 0 && client.Status != "pending" && client.Status != "" {
			if plugin, _, unknown := codexGlobalState(MarketplaceName(project)); !plugin && !unknown {
				client.Status = "pending"
				client.Reason = "the SPEC skills are not in this project and no global Codex plugin was found; run setup again to copy them into .agents/skills"
			}
		}
		clients[name] = client
	}
	return addCodexGlobal(project, map[string]any{"status": status, "cli_version": Version, "plugin_version": state.Current.Version, "project": project, "managed_files": managed, "catalogs": catalogs, "previous_versions": previous, "marketplace_name": MarketplaceName(project), "openspec": state.OpenSpec, "clients": clients, "native_invocation_verified": false, "document_language": state.DocumentLanguage, "config_document_language": configDocumentLanguage(project), "review": map[string]any{"mode": reviewOf(state), "plugins": reviewStatus(project, state)}, "project_checks": projectChecks(project)}), nil
}

// addCodexGlobal names a global Codex plugin an earlier SPEC setup
// registered for this project, which duplicates the project's skills in every
// Codex session, and the exact commands that remove what is there.
func addCodexGlobal(project string, result map[string]any) map[string]any {
	name := MarketplaceName(project)
	plugin, marketplace := codexGlobal(name)
	result["codex_global_plugin"] = ""
	commands := []string{}
	if plugin {
		commands = append(commands, "codex plugin remove spec@"+name)
	}
	if marketplace {
		commands = append(commands, "codex plugin marketplace remove "+name)
	}
	if len(commands) != 0 {
		result["codex_global_plugin"] = "spec@" + name
		result["codex_global_cleanup"] = strings.Join(commands, " && ")
	}
	return result
}

func staleClients(state *State) {
	for name, client := range state.Clients {
		if client.Status == "installed-by-client" {
			client.Registered = true
			client.Materialized = true
		}
		if client.Materialized || client.PluginDigest != "" {
			client.Status = "stale"
			client.Reason = "managed plugin changed; run setup again to refresh this agent"
			client.InvocationVerified = false
		}
		state.Clients[name] = client
	}
}

func (m *Manager) registerClients(ctx context.Context, project string, tools []string) (map[string]Client, error) {
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("plugin is not installed")
	}
	result := map[string]Client{}
	for _, tool := range tools {
		client := state.Clients[tool]
		if tool == "codex" {
			result[tool] = m.codexProjectSkills(project, state, client)
			continue
		}
		executable, err := m.LookPath(tool)
		if err != nil {
			client.Status = "pending"
			client.Reason = tool + " CLI is not installed/on PATH"
			result[tool] = client
			continue
		}
		if client.Status == "installed-by-client" {
			client.Registered = true
			client.Materialized = true
		}
		name := MarketplaceName(project)
		if !client.Registered {
			_, err = m.Run(ctx, project, nil, executable, "plugin", "marketplace", "add", filepath.Join(project, ".spec/marketplace"))
			if err == nil {
				client.Registered = true
			}
		} else if tool == "claude" {
			_, err = m.Run(ctx, project, nil, executable, "plugin", "marketplace", "update", name)
		}
		if err == nil {
			action := "add"
			if tool == "claude" {
				action = "install"
				if client.Materialized || client.PluginDigest != "" {
					action = "update"
				}
			}
			argv := []string{executable, "plugin", action, "spec@" + name}
			if tool == "claude" {
				argv = append(argv, "--scope", "project")
			}
			client.Output, err = m.Run(ctx, project, nil, argv...)
		}
		client.InvocationVerified = false
		client.Reason = ""
		if err != nil {
			client.Status = "pending"
			client.Reason = err.Error()
		} else {
			client.Status = "installed-by-client"
			client.Materialized = true
			client.PluginVersion = state.Current.Version
			client.PluginDigest = state.Current.Digest
		}
		result[tool] = client
	}
	for tool, client := range result {
		state.Clients[tool] = client
	}
	return result, saveState(project, state)
}

func (m *Manager) Versions(ctx context.Context, source string) ([]string, error) {
	return m.versions(ctx, source, nil)
}

// LatestRelease is the dashboard's quiet update check: a read-only tag listing
// that never prompts (no terminal, SSH in batch mode) and gives up quickly.
func (m *Manager) LatestRelease(ctx context.Context, project string) (current, latest string, err error) {
	state, err := LoadState(project)
	if err != nil || state == nil {
		return "", "", fmt.Errorf("workspace is not installed")
	}
	env := []string{}
	if os.Getenv("GIT_SSH_COMMAND") == "" {
		env = append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=10")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	tags, err := m.versions(ctx, state.Source, env)
	if err != nil || len(tags) == 0 {
		return state.Current.Version, "", err
	}
	return state.Current.Version, tags[len(tags)-1], nil
}

func (m *Manager) versions(ctx context.Context, source string, env []string) ([]string, error) {
	output, err := m.Run(ctx, "", env, "git", "ls-remote", "--tags", "--refs", "--", source)
	if err != nil {
		return nil, err
	}
	pattern := regexp.MustCompile(`^[a-f0-9]{40,64}\s+refs/tags/(v?\d+\.\d+\.\d+)$`)
	tags := []string{}
	for _, line := range strings.Split(output, "\n") {
		if match := pattern.FindStringSubmatch(line); match != nil {
			tags = append(tags, match[1])
		}
	}
	sort.Slice(tags, func(i, j int) bool { return compareVersions(tags[i], tags[j]) < 0 })
	return tags, nil
}
func compareVersions(a, b string) int {
	aa, bb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.ParseUint(aa[i], 10, 64)
		y, _ := strconv.ParseUint(bb[i], 10, 64)
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}
func (m *Manager) fetchPlugin(ctx context.Context, source, tag, destination string) (fs.FS, string, error) {
	if !semver.MatchString(strings.TrimPrefix(tag, "v")) {
		return nil, "", fmt.Errorf("use a stable version tag")
	}
	_, err := m.Run(ctx, "", nil, "git", "clone", "--depth", "1", "--branch", tag, "--single-branch", "--", source, destination)
	if err != nil {
		return nil, "", err
	}
	path, err := Within(destination, "plugin")
	if err != nil {
		return nil, "", err
	}
	if !exists(path) {
		path, err = Within(destination, "spec_cli/plugin")
		if err != nil {
			return nil, "", err
		}
	}
	plugin := os.DirFS(path)
	info, err := pluginInfo(plugin)
	if err != nil {
		return nil, "", err
	}
	if info.Version != strings.TrimPrefix(tag, "v") {
		return nil, "", fmt.Errorf("Git tag and plugin version disagree")
	}
	commit, err := m.Run(ctx, destination, nil, "git", "rev-parse", "HEAD")
	return plugin, commit, err
}

func (m *Manager) AgentCommand(ctx context.Context, project, tool string, extra []string) (*exec.Cmd, error) {
	if tool != "claude" && tool != "codex" {
		return nil, fmt.Errorf("agent must be claude or codex")
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	if state == nil || state.OpenSpec == nil {
		return nil, fmt.Errorf("run SPEC setup to prepare OpenSpec first")
	}
	executable, err := m.LookPath(tool)
	if err != nil {
		return nil, fmt.Errorf("%s is not installed/on PATH", tool)
	}
	plugin, err := verifyRelease(project, state.Current)
	if err != nil {
		return nil, err
	}
	bin, err := Within(project, ".spec/bin")
	if err != nil {
		return nil, err
	}
	shim, err := Within(bin, shimName())
	if err != nil {
		return nil, err
	}
	if !exists(shim) {
		return nil, fmt.Errorf("private OpenSpec launcher is missing")
	}
	argv := []string{}
	if tool == "claude" {
		argv = append(argv, "--plugin-dir", plugin)
	}
	argv = append(argv, extra...)
	self, err := executablePath()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, executable, argv...)
	cmd.Dir = project
	cmd.Env = environment("PATH="+bin+string(os.PathListSeparator)+filepath.Dir(self)+string(os.PathListSeparator)+os.Getenv("PATH"), "SPEC_EXECUTABLE="+self)
	return cmd, nil
}
