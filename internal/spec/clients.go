package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Codex has no project scope for plugins: a registered plugin is enabled in
// every Codex session. SPEC therefore copies its skills into the project's
// .agents/skills, which Codex reads as repository skills, and records each
// file's hash so updates and uninstall never overwrite or delete local edits.
const projectSkills = ".agents/skills/"

func readExisting(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	return readBytes(path, 16<<20)
}

// skillFiles maps project paths under .agents/skills to the release's skill files.
func skillFiles(project string, release Release) (map[string][]byte, error) {
	plugin, err := verifyRelease(project, release)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for rel := range release.Files {
		name, ok := strings.CutPrefix(rel, "skills/")
		if !ok {
			continue
		}
		data, err := readBytes(filepath.Join(plugin, filepath.FromSlash(rel)), 16<<20)
		if err != nil {
			return nil, err
		}
		files[projectSkills+name] = data
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("the installed plugin has no skills")
	}
	return files, nil
}

func localEdit(rel string) error {
	return fmt.Errorf("%s has local changes or was not written by SPEC; move it aside or merge it, then retry. Nothing was overwritten", rel)
}

// knownSkills maps every project skill path SPEC may have written to the
// contents it may hold there: the recorded copy and the file of every
// installed release, so an interrupted refresh, update or rollback is
// recognized as SPEC's own work. Any other content belongs to the user.
func knownSkills(state *State, recorded map[string]string) map[string]map[string]bool {
	known := map[string]map[string]bool{}
	add := func(rel, digest string) {
		if known[rel] == nil {
			known[rel] = map[string]bool{}
		}
		known[rel][digest] = true
	}
	for _, release := range append([]Release{state.Current}, state.History...) {
		for rel, digest := range release.Files {
			if name, ok := strings.CutPrefix(rel, "skills/"); ok {
				add(projectSkills+name, digest)
			}
		}
	}
	for rel, digest := range recorded {
		add(rel, digest)
	}
	return known
}

// writeProjectSkills preflights every target before writing and stops at
// the first file that holds content SPEC did not write.
func writeProjectSkills(project string, state *State, recorded map[string]string) (map[string]string, error) {
	wanted, err := skillFiles(project, state.Current)
	if err != nil {
		return nil, err
	}
	known := knownSkills(state, recorded)
	digests, obsolete := map[string]string{}, []string{}
	for _, rel := range sortedKeys(known) {
		path, err := Within(project, rel)
		if err != nil {
			return nil, err
		}
		current, err := readExisting(path)
		if err != nil {
			return nil, err
		}
		_, keep := wanted[rel]
		switch {
		case current == nil:
		case !known[rel][hash(current)] && (keep || recorded[rel] != ""):
			return nil, localEdit(rel)
		case !keep && known[rel][hash(current)]:
			obsolete = append(obsolete, rel)
		}
	}
	for _, rel := range sortedKeys(wanted) {
		data := wanted[rel]
		digests[rel] = hash(data)
		path, _ := Within(project, rel)
		if current, _ := readExisting(path); current != nil && hash(current) == digests[rel] {
			continue
		}
		if err := atomicFile(path, data, 0644); err != nil {
			return nil, err
		}
	}
	for _, rel := range obsolete {
		path, _ := Within(project, rel)
		if err := os.Remove(path); err != nil {
			return nil, err
		}
		pruneSkillDirs(project, rel)
	}
	return digests, nil
}

// removeProjectSkills deletes the skill copies that still hold SPEC's
// content and lists the ones it kept. With dry set it only counts them.
func removeProjectSkills(project string, state *State, recorded map[string]string, dry bool) (removed, kept []string, err error) {
	known := knownSkills(state, recorded)
	for _, rel := range sortedKeys(known) {
		path, err := Within(project, rel)
		if err != nil {
			return removed, kept, err
		}
		current, err := readExisting(path)
		if err != nil {
			return removed, kept, err
		}
		switch {
		case current == nil:
		case !known[rel][hash(current)]:
			kept = append(kept, rel)
		default:
			if !dry {
				if err = os.Remove(path); err != nil {
					return removed, kept, err
				}
				pruneSkillDirs(project, rel)
			}
			removed = append(removed, rel)
		}
	}
	return removed, kept, nil
}

// pruneSkillDirs removes directories left empty below .agents/skills.
func pruneSkillDirs(project, rel string) {
	for dir := filepath.ToSlash(filepath.Dir(rel)); strings.HasPrefix(dir, projectSkills); dir = filepath.ToSlash(filepath.Dir(dir)) {
		path, err := Within(project, dir)
		if err != nil || os.Remove(path) != nil {
			return
		}
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (m *Manager) codexProjectSkills(project string, state *State, client Client) Client {
	files, err := writeProjectSkills(project, state, client.Files)
	client.InvocationVerified, client.Output = false, ""
	if err != nil {
		client.Status, client.Reason = "pending", err.Error()
		return client
	}
	client.Status, client.Reason = "installed-in-project", ""
	// A global registration from an earlier version is detected from
	// Codex's configuration, not from this receipt.
	client.Files, client.Materialized, client.Registered = files, true, false
	client.PluginVersion, client.PluginDigest = state.Current.Version, state.Current.Digest
	return client
}

// readFollowed reads a client's own configuration file, following symlinks.
func readFollowed(path string) ([]byte, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	return readBytes(resolved, 4<<20)
}

// codexGlobal reports what an earlier SPEC setup registered globally in
// Codex for this project. It only reads Codex's configuration.
func codexGlobal(name string) (plugin, marketplace bool) {
	plugin, marketplace, _ = codexGlobalState(name)
	return plugin, marketplace
}

// codexGlobalState also reports a configuration that exists but cannot be read.
func codexGlobalState(name string) (plugin, marketplace, unknown bool) {
	settings, unknown := codexState()
	_, plugin = settings.Plugins["spec@"+name]
	_, marketplace = settings.Marketplaces[name]
	return plugin, marketplace, unknown
}

// codexConfig reads Codex's user configuration. Dotfile managers often link
// config.toml, so the read follows a symlink.
func codexConfig() ([]byte, error) {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		home = filepath.Join(dir, ".codex")
	}
	return readFollowed(filepath.Join(home, "config.toml"))
}

// codexSettings is the part of Codex's config.toml SPEC reads.
type codexSettings struct {
	Plugins      map[string]map[string]any `toml:"plugins"`
	Marketplaces map[string]map[string]any `toml:"marketplaces"`
}

// codexState parses Codex's configuration; it never writes it. A file that
// exists but cannot be read or parsed is unknown, so nothing is assumed.
func codexState() (settings codexSettings, unknown bool) {
	data, err := codexConfig()
	if err != nil {
		return settings, !os.IsNotExist(err)
	}
	if _, err = toml.Decode(string(data), &settings); err != nil {
		return codexSettings{}, true
	}
	return settings, false
}

func claudeHome() string {
	if home := os.Getenv("CLAUDE_CONFIG_DIR"); home != "" {
		return home
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, ".claude")
}

type claudeInstall struct {
	Scope       string `json:"scope"`
	ProjectPath string `json:"projectPath"`
}

// claudeInstalls reads Claude Code's own record of a plugin's installs.
func claudeInstalls(plugin string) (entries []claudeInstall, unknown bool) {
	home := claudeHome()
	if home == "" {
		return nil, true
	}
	var plugins struct {
		Plugins map[string][]claudeInstall `json:"plugins"`
	}
	data, err := readFollowed(filepath.Join(home, "plugins", "installed_plugins.json"))
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil || json.Unmarshal(data, &plugins) != nil {
		return nil, true
	}
	return plugins.Plugins[plugin], false
}

// claudeMarketplace reads Claude Code's record of a known marketplace.
func claudeMarketplace(name string) (entry map[string]any, known, unknown bool) {
	home := claudeHome()
	if home == "" {
		return nil, false, true
	}
	var marketplaces map[string]map[string]any
	data, err := readFollowed(filepath.Join(home, "plugins", "known_marketplaces.json"))
	if os.IsNotExist(err) {
		return nil, false, false
	}
	if err != nil || json.Unmarshal(data, &marketplaces) != nil {
		return nil, false, true
	}
	entry, known = marketplaces[name]
	return entry, known, false
}

// claudeProjectInstall reports whether Claude Code has a plugin installed
// with project scope for this project. A record that cannot be read counts
// as present, so uninstall still asks Claude Code to remove it.
func claudeProjectInstall(project, plugin string) bool {
	entries, unknown := claudeInstalls(plugin)
	installed := unknown
	for _, entry := range entries {
		installed = installed || entry.Scope == "project" && samePath(entry.ProjectPath, project)
	}
	return installed
}

// claudeRegistration reports whether Claude Code has the SPEC plugin
// installed for this project and knows its marketplace.
func claudeRegistration(project, name string) (installed, marketplace bool) {
	_, known, unknown := claudeMarketplace(name)
	return claudeProjectInstall(project, "spec@"+name), known || unknown
}

// samePath compares two directories, also through symlinks such as macOS's
// /var -> /private/var: a client may record either form.
func samePath(a, b string) bool {
	if resolved, err := filepath.EvalSymlinks(a); err == nil {
		a = resolved
	}
	if resolved, err := filepath.EvalSymlinks(b); err == nil {
		b = resolved
	}
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// claudeEnabled reports whether the project's own Claude Code settings
// enable the SPEC plugin.
func claudeEnabled(project, name string) bool {
	path, err := Within(project, ".claude/settings.json")
	if err != nil {
		return false
	}
	var settings struct {
		EnabledPlugins map[string]any `json:"enabledPlugins"`
	}
	if readJSON(path, &settings) != nil {
		return false
	}
	_, enabled := settings.EnabledPlugins["spec@"+name]
	return enabled
}

// claudeDisabled reports whether Claude Code settings turn a plugin off. Local
// project settings win over project settings, which win over user settings.
func claudeDisabled(project, plugin string) bool {
	paths := []string{filepath.Join(project, ".claude", "settings.local.json"), filepath.Join(project, ".claude", "settings.json")}
	if home := claudeHome(); home != "" {
		paths = append(paths, filepath.Join(home, "settings.json"))
	}
	for _, path := range paths {
		var settings struct {
			EnabledPlugins map[string]any `json:"enabledPlugins"`
		}
		data, err := readFollowed(path)
		if err != nil || json.Unmarshal(data, &settings) != nil {
			continue
		}
		if enabled, ok := settings.EnabledPlugins[plugin]; ok {
			return enabled == false
		}
	}
	return false
}

// Uninstall removes what SPEC set up for its agents and then its own .spec
// state. Project documents, OpenSpec configuration and OpenSpec's agent files
// are kept. A failed client step keeps the state so the command can be retried.
func (m *Manager) Uninstall(ctx context.Context, options Options) (map[string]any, int, error) {
	project, err := Project(options.Project)
	if err != nil {
		return nil, 2, err
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, 2, err
	}
	if state == nil {
		return nil, 2, fmt.Errorf("SPEC is not installed in this project")
	}
	if !options.DryRun {
		unlock, err := lockProject(project)
		if err != nil {
			return nil, 2, err
		}
		defer unlock()
		// Read the receipt again under the lock; an update may have finished.
		if state, err = LoadState(project); err != nil || state == nil {
			return nil, 2, fmt.Errorf("SPEC state changed while waiting; run uninstall again")
		}
	}
	name := MarketplaceName(project)
	steps, egmNotes := egmRemoval(project, state)
	installed, marketplace := claudeRegistration(project, name)
	if installed {
		steps = append(steps, step{"claude", "Remove the Claude Code plugin spec@" + name + " from this project", []string{"plugin", "uninstall", "spec@" + name, "--scope", "project"}})
	}
	if marketplace {
		steps = append(steps, step{"claude", "Remove the Claude Code marketplace " + name, []string{"plugin", "marketplace", "remove", name}})
	}
	plugin, codexMarketplace, codexUnknown := codexGlobalState(name)
	if plugin {
		steps = append(steps, step{"codex", "Remove the global Codex plugin spec@" + name, []string{"plugin", "remove", "spec@" + name}})
	}
	if codexMarketplace {
		steps = append(steps, step{"codex", "Remove the Codex marketplace " + name, []string{"plugin", "marketplace", "remove", name}})
	}
	skills := state.Clients["codex"].Files
	kept := []string{"Project documents, openspec/ and OpenSpec's own agent skills and commands are kept."}
	kept = append(kept, egmNotes...)
	if codexUnknown {
		kept = append(kept, "Codex's config.toml could not be read. If it lists spec@"+name+", remove it with codex plugin remove spec@"+name+" and codex plugin marketplace remove "+name+".")
	}
	if !installed && claudeEnabled(project, name) {
		kept = append(kept, ".claude/settings.json still enables spec@"+name+"; it is your file, so SPEC leaves it. Remove that entry if you share the file.")
	}
	if options.DryRun {
		planned := []string{}
		for _, s := range steps {
			planned = append(planned, s.label)
		}
		removable, edited, err := removeProjectSkills(project, state, skills, true)
		if err != nil {
			return nil, 2, err
		}
		if len(removable) != 0 {
			planned = append(planned, fmt.Sprintf("Remove %d SPEC skill files from .agents/skills", len(removable)))
		}
		for _, rel := range edited {
			kept = append(kept, rel+" has local changes or was not written by SPEC and will be kept")
		}
		planned = append(planned, "Remove .spec (plugin versions, catalogs, private OpenSpec launcher and receipt)")
		return map[string]any{"status": "planned", "project": project, "actions": planned, "kept": kept}, 0, nil
	}
	done, failures := []string{}, []string{}
	for _, s := range steps {
		if err := m.runStep(ctx, project, s); err != nil {
			failures = append(failures, s.label+": "+err.Error())
			continue
		}
		done = append(done, s.label)
	}
	removed, edited, err := removeProjectSkills(project, state, skills, false)
	if err != nil {
		failures = append(failures, "Remove SPEC skills from .agents/skills: "+err.Error())
	}
	if len(removed) != 0 {
		done = append(done, fmt.Sprintf("Removed %d SPEC skill files from .agents/skills", len(removed)))
	}
	for _, rel := range edited {
		kept = append(kept, rel+" has local changes or was not written by SPEC and was kept")
	}
	result := map[string]any{"project": project, "done": done, "kept": kept}
	if len(failures) != 0 {
		// A retry recognizes SPEC's remaining files by their recorded and
		// released hashes; the receipt only stops claiming the skills are set up.
		if len(removed) != 0 {
			client := state.Clients["codex"]
			client.Status, client.Reason = "pending", "uninstall did not finish; run spec uninstall again"
			state.Clients["codex"] = client
			if err = saveState(project, state); err != nil {
				failures = append(failures, "Update the SPEC receipt: "+err.Error())
			}
		}
		result["status"], result["failures"] = "partial", failures
		return result, 3, nil
	}
	if err = removeState(project); err != nil {
		return nil, 2, err
	}
	result["status"] = "uninstalled"
	result["done"] = append(done, "Removed .spec")
	if home := claudeHome(); home != "" {
		if cache := filepath.Join(home, "plugins", "cache", name); exists(cache) {
			result["client_cache"] = "Claude Code kept an unused cache copy at " + cache
		}
	}
	return result, 0, nil
}

// removeState moves .spec aside in one rename before deleting it, so a
// failed deletion never leaves a half-removed receipt behind.
func removeState(project string) error {
	folder, err := stateFolder(project, false)
	if err != nil {
		return err
	}
	aside, err := os.MkdirTemp(project, ".spec-removed-")
	if err != nil {
		return err
	}
	if err = os.Remove(aside); err != nil {
		return err
	}
	if err = os.Rename(folder, aside); err != nil {
		return err
	}
	if err = os.RemoveAll(aside); err != nil {
		return fmt.Errorf("SPEC was uninstalled, but %s could not be deleted: %w", aside, err)
	}
	return nil
}
