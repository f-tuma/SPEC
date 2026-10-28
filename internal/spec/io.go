package spec

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const Version = "0.4.0"
const Repository = "https://github.com/f-tuma/SPEC.git"

type Manager struct {
	Plugin   fs.FS
	Run      func(context.Context, string, []string, ...string) (string, error)
	LookPath func(string) (string, error)
	Download func(context.Context, string, string, int64) error
	Rename   func(string, string) error
}

func New(plugin fs.FS) *Manager {
	return &Manager{Plugin: plugin, Run: run, LookPath: exec.LookPath, Download: download, Rename: os.Rename}
}

func executablePath() (string, error) {
	path, err := os.Executable()
	if err == nil {
		return path, nil
	}
	// Some containers do not mount /proc/self/exe. Resolve the original argv.
	path, err = exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

func run(ctx context.Context, cwd string, env []string, argv ...string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("empty command")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = environment(append([]string{"GIT_TERMINAL_PROMPT=0", "CI=1"}, env...)...)
	var stdout, stderr tailBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s failed: %w\n%s\n%s", filepath.Base(argv[0]), err, stderr.String(), stdout.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Keep command diagnostics bounded while still draining both output streams.
type tailBuffer struct{ data []byte }

func (b *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > 65536 {
		p = p[len(p)-65536:]
	}
	b.data = append(b.data, p...)
	if len(b.data) > 65536 {
		b.data = b.data[len(b.data)-65536:]
	}
	return n, nil
}
func (b *tailBuffer) String() string { return string(b.data) }

func environment(values ...string) []string {
	env := os.Environ()
	for _, value := range values {
		key, _, ok := strings.Cut(value, "=")
		if !ok {
			continue
		}
		prefix := key + "="
		filtered := env[:0]
		for _, old := range env {
			if !strings.HasPrefix(strings.ToUpper(old), strings.ToUpper(prefix)) {
				filtered = append(filtered, old)
			}
		}
		env = append(filtered, value)
	}
	return env
}

func Project(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("an explicit project directory is required")
	}
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		value = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(value, "~"), "/"))
	}
	root, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("invalid project: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	home, _ := os.UserHomeDir()
	home, _ = filepath.EvalSymlinks(home)
	if !info.IsDir() || filepath.Dir(root) == root || root == home {
		return "", fmt.Errorf("choose an existing project directory, not a home or filesystem root")
	}
	if _, err := Within(root, ".spec"); err != nil {
		return "", err
	}
	return root, nil
}

// Within rejects symlinks in every existing component, including the leaf.
func Within(root, relative string) (string, error) {
	if relative == "." || relative == "" || strings.ContainsAny(relative, "\\\x00:") || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("invalid repository-relative path: %q", relative)
	}
	target := root
	for _, part := range strings.Split(relative, "/") {
		if part == ".." || part == "" || part == "." {
			return "", fmt.Errorf("invalid path component in %q", relative)
		}
		target = filepath.Join(target, part)
		info, err := os.Lstat(target)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink in managed/input path: %s", relative)
		}
	}
	return target, nil
}

func readBytes(path string, max int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > max {
		return nil, fmt.Errorf("unsupported file or size: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, max+1))
	if int64(len(data)) > max {
		return nil, fmt.Errorf("file exceeds size limit: %s", path)
	}
	return data, err
}

func readJSON(path string, value any) error {
	data, err := readBytes(path, 4<<20)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func atomicJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicFile(path, append(data, '\n'), 0644)
}

func atomicFile(path string, data []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".spec-write-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func exists(path string) bool { _, err := os.Lstat(path); return err == nil }

func download(ctx context.Context, url, target string, max int64) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SPEC/"+Version)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(response.Body, max+1))
	if n > max {
		return fmt.Errorf("download exceeds size limit")
	}
	return err
}
