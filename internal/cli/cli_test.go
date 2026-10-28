package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/f-tuma/SPEC/internal/spec"
)

func TestCLIJSONAndInputErrors(t *testing.T) {
	project := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(t.TempDir(), "claude"))
	m := spec.New(os.DirFS("../../plugin"))
	for _, test := range []struct {
		args   []string
		code   int
		status string
	}{
		{[]string{"status", "--project", project}, 0, "not-installed"},
		{[]string{"install", "--project", project, "--dry-run", "--cache-dir", filepath.Join(t.TempDir(), "cache")}, 0, "planned"},
		{[]string{"install", "--project", project, "--dry-run", "--openspec-method", "pnpm", "--cache-dir", filepath.Join(t.TempDir(), "cache")}, 0, "planned"},
		{[]string{"install", "--project", project, "--openspec-method", "unknown"}, 2, "error"},
		{[]string{"install", "--project", project, "--dry-run", "--document-language", "Czech", "--cache-dir", filepath.Join(t.TempDir(), "cache")}, 0, "planned"},
		{[]string{"install", "--project", project, "--document-language", "Czech\nrules"}, 2, "error"},
		{[]string{"install", "--project", project, "--dry-run", "--review", "egm", "--cache-dir", filepath.Join(t.TempDir(), "cache")}, 0, "planned"},
		{[]string{"install", "--project", project, "--review", "goldfish"}, 2, "error"},
		{[]string{"status", "--project", project, "--review", "egm"}, 2, "error"},
		{[]string{"install", "--project", project, "--tools", "invalid"}, 2, "error"},
		{[]string{"status"}, 2, "error"},
		{[]string{"status", "--project", project, "unexpected"}, 2, "error"},
		{[]string{"check", "--root", project}, 1, ""},
		{[]string{"fingerprint", "--project", project}, 2, "error"},
		{[]string{"uninstall", "--project", project, "--dry-run"}, 2, "error"},
		{[]string{"uninstall", "--project", project}, 2, "error"},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			var out, stderr bytes.Buffer
			code := Run(context.Background(), test.args, m, strings.NewReader(""), &out, &stderr)
			var result map[string]any
			if err := json.Unmarshal(out.Bytes(), &result); err != nil || code != test.code {
				t.Fatalf("code=%d %s %v", code, out.String(), err)
			}
			if test.status != "" && result["status"] != test.status {
				t.Fatal(result)
			}
		})
	}
}
func TestDocumentLanguageReachesThePlan(t *testing.T) {
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	var out, stderr bytes.Buffer
	args := []string{"install", "--project", t.TempDir(), "--dry-run", "--document-language", "Czech", "--cache-dir", filepath.Join(t.TempDir(), "cache")}
	if code := Run(context.Background(), args, spec.New(os.DirFS("../../plugin")), strings.NewReader(""), &out, &stderr); code != 0 || !strings.Contains(out.String(), `"document_language": "Czech"`) {
		t.Fatal(code, out.String())
	}
}
func TestDesignListsPreviewsAndCreatesOnce(t *testing.T) {
	project := t.TempDir()
	m := spec.New(os.DirFS("../../plugin"))
	run := func(args ...string) (int, string) {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), append([]string{"design", "--project", project}, args...), m, strings.NewReader(""), &out, &stderr)
		return code, out.String()
	}
	if code, out := run(); code != 0 || !strings.Contains(out, `"id": "retro-pixel"`) || !strings.Contains(out, `"value": "Press Start 2P"`) {
		t.Fatal(code, out)
	}
	if code, out := run("--starter", "cozy-warm"); code != 0 || !strings.Contains(out, `"status": "planned"`) {
		t.Fatal(code, out)
	}
	if _, err := os.Stat(filepath.Join(project, "DESIGN.md")); !os.IsNotExist(err) {
		t.Fatal("preview wrote DESIGN.md")
	}
	if code, out := run("--starter", "cozy-warm", "--write"); code != 0 || !strings.Contains(out, `"status": "created"`) {
		t.Fatal(code, out)
	}
	if code, out := run("--starter", "editorial", "--write"); code != 2 || !strings.Contains(out, "nothing was overwritten") {
		t.Fatal(code, out)
	}
}
func TestNonTTYDefaultsToHelpInsteadOfBlocking(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run(context.Background(), nil, spec.New(nil), strings.NewReader(""), &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "spec tui") {
		t.Fatal(code, out.String())
	}
}
