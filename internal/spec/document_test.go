package spec

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestProjectDocumentsIncludeActiveChanges(t *testing.T) {
	_, project, _ := managerFixture(t)
	put(t, filepath.Join(project, "docs/project-brief.md"), "# Brief")
	put(t, filepath.Join(project, "DESIGN.md"), "# Design")
	put(t, filepath.Join(project, "openspec/changes/add-garden/proposal.md"), "# Proposal")
	put(t, filepath.Join(project, "openspec/changes/archive/old/proposal.md"), "# Old")
	got := ProjectDocuments(project)
	want := []Document{{"brief", "docs/project-brief.md"}, {"visual_design", "DESIGN.md"}, {"changes", "openspec/changes/add-garden/proposal.md"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
}

func TestDocumentWriteIsGuardedAndPreservesMode(t *testing.T) {
	_, project, _ := managerFixture(t)
	path := filepath.Join(project, "DESIGN.md")
	put(t, path, "first")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	content, digest, err := ReadDocument(project, "DESIGN.md")
	if err != nil || content != "first" {
		t.Fatal(content, err)
	}
	next, err := WriteDocument(project, "DESIGN.md", "second", digest)
	if err != nil || next == digest {
		t.Fatal(next, err)
	}
	if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatal("file mode changed:", info.Mode())
	}
	if _, err = WriteDocument(project, "DESIGN.md", "third", digest); err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatal("stale digest was accepted:", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "second" {
		t.Fatal("refused save changed the file")
	}
	for _, rel := range []string{".spec/state.json", "../outside.md", "missing.md"} {
		if _, err = WriteDocument(project, rel, "x", ""); err == nil {
			t.Fatalf("%s was writable", rel)
		}
	}
	put(t, filepath.Join(project, "binary.md"), "\xff\xfe")
	if _, _, err = ReadDocument(project, "binary.md"); err == nil {
		t.Fatal("non-UTF-8 content opened as text")
	}
}
