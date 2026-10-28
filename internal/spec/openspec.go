package spec

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const OpenSpecVersion = "1.15.0"
const openSpecRelease = "https://github.com/Fission-AI/OpenSpec/releases/download/v" + OpenSpecVersion

//go:embed bridge.txt
var bridge string

func DefaultCache() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "SPEC-cache")
	}
	return filepath.Join(base, "SPEC")
}

func resolveMissing(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if exists(path) {
		return filepath.EvalSymlinks(path)
	}
	parent, err := resolveMissing(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}
func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func legacyPreflight(project string) error {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		base, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		home = filepath.Join(base, ".codex")
	}
	for _, name := range []string{"propose", "explore", "new", "continue", "apply", "update", "ff", "sync", "archive", "bulk-archive", "verify", "onboard"} {
		path := filepath.Join(home, "prompts", "opsx-"+name+".md")
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("OpenSpec would migrate existing global legacy prompts; resolve that migration separately: %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	for _, rel := range []string{".claude/commands/openspec", ".codex/skills/openspec-propose"} {
		path, err := Within(project, rel)
		if err != nil {
			return err
		}
		if exists(path) {
			return fmt.Errorf("legacy OpenSpec layout needs a separate migration: %s", rel)
		}
	}
	matches, err := filepath.Glob(filepath.Join(project, ".codex/prompts/openspec-*.md"))
	if err != nil {
		return err
	}
	if len(matches) > 0 {
		return fmt.Errorf("legacy project OpenSpec prompts need a separate migration")
	}
	return nil
}

func (m *Manager) ensureCLI(ctx context.Context, cache, method, provided string) ([]string, string, error) {
	versionOK := func(argv []string) bool {
		out, err := m.Run(ctx, "", nil, append(append([]string{}, argv...), "--version")...)
		return err == nil && strings.TrimPrefix(strings.TrimSpace(out), "v") == OpenSpecVersion
	}
	if provided != "" {
		path, err := filepath.Abs(provided)
		if err != nil {
			return nil, "", err
		}
		if !versionOK([]string{path}) {
			return nil, "", fmt.Errorf("expected OpenSpec %s at --openspec-bin", OpenSpecVersion)
		}
		return []string{path}, "provided", nil
	}
	if path, err := m.LookPath("openspec"); err == nil && versionOK([]string{path}) {
		return []string{path}, "existing", nil
	}
	if method == "auto" {
		method = "binary"
	}
	if method != "binary" && method != "npm" && method != "pnpm" {
		return nil, "", fmt.Errorf("openspec-method must be auto, binary, pnpm or npm")
	}
	base := filepath.Join(cache, "openspec", OpenSpecVersion, method)
	var argv []string
	if method == "npm" || method == "pnpm" {
		node, err := m.LookPath("node")
		if err != nil {
			return nil, "", fmt.Errorf("%s installation requires Node.js >=20.19.0", method)
		}
		installer, err := m.LookPath(method)
		if err != nil {
			return nil, "", fmt.Errorf("%s installation requires %s on PATH: %w", method, method, err)
		}
		version, err := m.Run(ctx, "", nil, node, "--version")
		if err != nil {
			return nil, "", err
		}
		version = strings.Split(strings.TrimPrefix(version, "v"), "-")[0]
		if !semver.MatchString(version) || compareVersions(version, "20.19.0") < 0 {
			return nil, "", fmt.Errorf("%s installation requires Node.js >=20.19.0", method)
		}
		entry := filepath.Join(base, "node_modules/@fission-ai/openspec/bin/openspec.js")
		argv = []string{node, entry}
		if !exists(entry) {
			if err = os.MkdirAll(base, 0755); err != nil {
				return nil, "", err
			}
			install := []string{installer, "install", "--prefix", base, "--cache", filepath.Join(cache, "npm-cache"), "--save-exact", "--ignore-scripts", "--no-audit", "--no-fund", "@fission-ai/openspec@" + OpenSpecVersion}
			if method == "pnpm" {
				install = []string{installer, "add", "--dir", base, "--ignore-workspace", "--store-dir", filepath.Join(cache, "pnpm-store"), "--config.cache-dir=" + filepath.Join(cache, "pnpm-cache"), "--save-exact", "--ignore-scripts", "--ignore-pnpmfile", "@fission-ai/openspec@" + OpenSpecVersion}
			}
			_, err = m.Run(ctx, base, nil, install...)
			if err != nil {
				return nil, "", err
			}
		}
	} else {
		name, err := binaryName()
		if err != nil {
			return nil, "", err
		}
		entry := filepath.Join(base, "openspec")
		if runtime.GOOS == "windows" {
			entry += ".exe"
		}
		argv = []string{entry}
		if !exists(entry) {
			if err = os.MkdirAll(base, 0755); err != nil {
				return nil, "", err
			}
			temp, err := os.MkdirTemp(base, "download-")
			if err != nil {
				return nil, "", err
			}
			defer os.RemoveAll(temp)
			checksums := filepath.Join(temp, "SHA256SUMS")
			if err = m.Download(ctx, openSpecRelease+"/SHA256SUMS", checksums, 65536); err != nil {
				return nil, "", err
			}
			data, err := readBytes(checksums, 65536)
			if err != nil {
				return nil, "", err
			}
			expected := ""
			for _, line := range strings.Split(string(data), "\n") {
				parts := strings.Fields(line)
				if len(parts) == 2 && strings.TrimPrefix(parts[1], "*") == name {
					expected = parts[0]
				}
			}
			if !fingerprint.MatchString(expected) {
				return nil, "", fmt.Errorf("official checksums do not contain the selected binary")
			}
			file := filepath.Join(temp, name)
			if err = m.Download(ctx, openSpecRelease+"/"+name, file, 150<<20); err != nil {
				return nil, "", err
			}
			data, err = readBytes(file, 150<<20)
			if err != nil {
				return nil, "", err
			}
			if hash(data) != expected {
				return nil, "", fmt.Errorf("OpenSpec binary checksum mismatch")
			}
			if err = os.Chmod(file, 0755); err != nil {
				return nil, "", err
			}
			if err = os.Rename(file, entry); err != nil {
				return nil, "", err
			}
		}
	}
	if !versionOK(argv) {
		return nil, "", fmt.Errorf("expected OpenSpec %s in %s; installed executable returned an unexpected version or could not run. Inspect this backend cache or retry with a fresh external --cache-dir", OpenSpecVersion, base)
	}
	return argv, method, nil
}

func binaryName() (string, error) {
	system := runtime.GOOS
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	if (system != "darwin" && system != "linux" && system != "windows") || arch == "" {
		return "", fmt.Errorf("no supported standalone OpenSpec binary for this platform")
	}
	suffix := ""
	if system == "linux" && exists("/etc/alpine-release") {
		suffix = "-musl"
	}
	if system == "windows" {
		suffix = ".exe"
	}
	return "openspec-" + system + "-" + arch + suffix, nil
}

func (m *Manager) initialize(ctx context.Context, options Options, dry bool) (map[string]any, error) {
	project := options.Project
	cache, err := resolveMissing(options.Cache)
	if err != nil {
		return nil, err
	}
	if inside(project, cache) {
		return nil, fmt.Errorf("choose an OpenSpec cache outside the project for isolated staging")
	}
	if err = legacyPreflight(project); err != nil {
		return nil, err
	}
	config, err := Within(project, "openspec/config.yaml")
	if err != nil {
		return nil, err
	}
	if !exists(config) {
		alternate, err := Within(project, "openspec/config.yml")
		if err != nil {
			return nil, err
		}
		if exists(alternate) {
			config = alternate
		}
	}
	var original []byte
	hasConfig := exists(config)
	configRel := ""
	if hasConfig {
		original, err = readBytes(config, 4<<20)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(project, config)
		configRel = filepath.ToSlash(rel)
	}
	priorState, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	var priorCommand []string
	language := options.DocumentLanguage
	if priorState != nil {
		priorCommand = stringList(priorState.OpenSpec["command"])
		if !options.SetDocumentLanguage {
			language = priorState.DocumentLanguage
		}
	}
	// A new config requests the language itself. An existing config is never
	// edited, so the plan names the line its owner can add.
	languageInfo := map[string]any{"document_language": language}
	if hasConfig {
		configLanguage := configDocumentLanguage(project)
		languageInfo["config_document_language"] = configLanguage
		if language != "" && configLanguage != language {
			languageInfo["config_language_line"] = LanguageLine(language)
		}
	}
	bin, err := Within(project, ".spec/bin")
	if err != nil {
		return nil, err
	}
	shim, err := Within(bin, shimName())
	if err != nil {
		return nil, err
	}
	var priorShim []byte
	if exists(shim) {
		priorShim, err = readBytes(shim, 65536)
		if err != nil {
			return nil, err
		}
		if len(priorCommand) == 0 || !ownedShim(string(priorShim), priorCommand) {
			return nil, fmt.Errorf("existing OpenSpec launcher differs from its receipt; preserve/resolve it before reinstalling")
		}
	}
	if dry {
		result := map[string]any{"status": "planned", "version": OpenSpecVersion, "tools": options.Tools, "cache": cache, "method": options.Method, "existing_config_preserved": hasConfig, "existing_config_path": configRel}
		for k, v := range languageInfo {
			result[k] = v
		}
		return result, nil
	}
	var argv []string
	method := ""
	if options.Method == "auto" && options.OpenSpecBin == "" && len(priorCommand) > 0 {
		version, checkErr := m.Run(ctx, "", nil, append(append([]string{}, priorCommand...), "--version")...)
		if checkErr == nil && strings.TrimPrefix(strings.TrimSpace(version), "v") == OpenSpecVersion {
			argv = priorCommand
			method = "managed-existing"
		}
	}
	if len(argv) == 0 {
		argv, method, err = m.ensureCLI(ctx, cache, options.Method, options.OpenSpecBin)
		if err != nil {
			return nil, err
		}
	}
	if err = os.MkdirAll(cache, 0755); err != nil {
		return nil, err
	}
	stage, err := os.MkdirTemp(cache, "init-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stage)
	output, err := m.Run(ctx, stage, []string{"XDG_CONFIG_HOME=" + filepath.Join(cache, "config")}, append(append([]string{}, argv...), "init", stage, "--tools", strings.Join(options.Tools, ","), "--profile", "core", "--no-animation")...)
	if err != nil {
		return nil, err
	}
	generated := os.DirFS(stage)
	configData, err := fs.ReadFile(generated, "openspec/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("OpenSpec init did not create its staging config: %w", err)
	}
	for _, tool := range options.Tools {
		base := ".agents"
		if tool == "claude" {
			base = ".claude"
		}
		matches, err := fs.Glob(generated, base+"/skills/openspec-*/SKILL.md")
		if err != nil || len(matches) == 0 {
			return nil, fmt.Errorf("OpenSpec did not generate skills for %s; check its delivery setting", tool)
		}
	}
	type write struct {
		path string
		data []byte
		rel  string
	}
	writes := []write{}
	created, preserved := []string{}, []string{}
	err = fs.WalkDir(generated, ".", func(rel string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("unexpected symlink in OpenSpec output: %s", rel)
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasPrefix(rel, "openspec/") && !strings.HasPrefix(rel, ".claude/") && !strings.HasPrefix(rel, ".agents/") {
			return fmt.Errorf("unexpected OpenSpec output: %s", rel)
		}
		if hasConfig && strings.HasPrefix(rel, "openspec/") {
			preserved = append(preserved, rel)
			return nil
		}
		target, err := Within(project, rel)
		if err != nil {
			return err
		}
		if exists(target) {
			info, err := os.Stat(target)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("OpenSpec output conflicts with a directory: %s", rel)
			}
			preserved = append(preserved, rel)
			return nil
		}
		data, err := fs.ReadFile(generated, rel)
		if err != nil {
			return err
		}
		if rel == "openspec/config.yaml" {
			data = append(configData, []byte(bridgeFor(language))...)
		}
		writes = append(writes, write{target, data, rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	content, err := shimContent(argv)
	if err != nil {
		return nil, err
	}
	for _, w := range writes {
		if err = os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(w.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(w.data)
		closeErr := f.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		created = append(created, w.rel)
	}
	if !hasConfig {
		for _, rel := range []string{"openspec/specs", "openspec/changes/archive"} {
			path, err := Within(project, rel)
			if err != nil {
				return nil, err
			}
			if err = os.MkdirAll(path, 0755); err != nil {
				return nil, err
			}
		}
	}
	if hasConfig {
		data, err := readBytes(config, 4<<20)
		if err != nil {
			return nil, err
		}
		if string(data) != string(original) {
			return nil, fmt.Errorf("existing OpenSpec config unexpectedly changed")
		}
	}
	if string(priorShim) != content {
		if err = atomicFile(shim, []byte(content), 0755); err != nil {
			return nil, err
		}
	}
	result := map[string]any{"status": "initialized", "version": OpenSpecVersion, "installed_by": method, "command": argv, "tools": options.Tools, "created_files": created, "preserved_files": preserved, "bin_directory": bin, "existing_config_preserved": hasConfig, "existing_config_path": configRel, "planning_shape_preserved": hasConfig, "init_output": output, "native_host_invocation_verified": false}
	for k, v := range languageInfo {
		result[k] = v
	}
	state, err := LoadState(project)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("plugin setup must precede OpenSpec initialization")
	}
	state.OpenSpec = result
	state.DocumentLanguage = language
	if err = saveState(project, state); err != nil {
		return nil, err
	}
	return result, nil
}

func ownedShim(content string, argv []string) bool {
	expected, err := shimContent(argv)
	if err == nil && content == expected {
		return true
	}
	return runtime.GOOS == "windows" && content == legacyWindowsShim(argv)
}

// Python 0.2 used subprocess.list2cmdline before appending batch forwarding.
func legacyWindowsShim(argv []string) string {
	args := []string{}
	for _, arg := range argv {
		quote := arg == "" || strings.ContainsAny(arg, " \t")
		var out strings.Builder
		if quote {
			out.WriteByte('"')
		}
		slashes := 0
		for _, r := range arg {
			if r == '\\' {
				slashes++
				continue
			}
			if r == '"' {
				out.WriteString(strings.Repeat("\\", slashes*2+1))
				out.WriteRune(r)
			} else {
				out.WriteString(strings.Repeat("\\", slashes))
				out.WriteRune(r)
			}
			slashes = 0
		}
		out.WriteString(strings.Repeat("\\", slashes))
		if quote {
			out.WriteString(strings.Repeat("\\", slashes))
			out.WriteByte('"')
		}
		args = append(args, out.String())
	}
	return "@echo off\r\n" + strings.ReplaceAll(strings.Join(args, " "), "%", "%%") + " %*\r\n"
}

func shimName() string {
	if runtime.GOOS == "windows" {
		return "openspec.cmd"
	}
	return "openspec"
}

var shellSafe = regexp.MustCompile(`^[a-zA-Z0-9_@%+=:,./-]+$`)

func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
func shimContent(argv []string) (string, error) {
	quoted := []string{}
	if runtime.GOOS == "windows" {
		for _, arg := range argv {
			if strings.ContainsAny(arg, "\r\n\"") {
				return "", fmt.Errorf("unsupported Windows launcher argument")
			}
			quoted = append(quoted, "\""+strings.ReplaceAll(arg, "%", "%%")+"\"")
		}
		return "@echo off\r\nsetlocal DisableDelayedExpansion\r\n" + strings.Join(quoted, " ") + " %*\r\n", nil
	}
	for _, arg := range argv {
		quoted = append(quoted, shellQuote(arg))
	}
	return "#!/bin/sh\nexec " + strings.Join(quoted, " ") + " \"$@\"\n", nil
}
