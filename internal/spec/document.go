package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Documents are project files the owner reads and edits in the TUI. SPEC
// writes one only on an explicit save, and never inside its own .spec state.
type Document struct {
	Role string `json:"role"`
	Path string `json:"path"`
}

var documentOrder = []string{"brief", "instructions", "visual_design", "specification", "changes", "architecture", "commands", "session", "release"}

// ProjectDocuments lists discovered documents and active OpenSpec change files.
func ProjectDocuments(project string) []Document {
	docs := discover(project)
	matches, _ := filepath.Glob(filepath.Join(project, "openspec/changes/*/*.md"))
	sort.Strings(matches)
	for _, path := range matches {
		rel, err := filepath.Rel(project, path)
		if err != nil {
			continue
		}
		if _, err = inputBytes(project, filepath.ToSlash(rel)); err == nil {
			docs["changes"] = append(docs["changes"], filepath.ToSlash(rel))
		}
	}
	list := []Document{}
	for _, role := range documentOrder {
		for _, path := range docs[role] {
			list = append(list, Document{role, path})
		}
	}
	return list
}

func documentPath(project, rel string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == ".spec" || strings.HasPrefix(clean, ".spec/") {
		return "", fmt.Errorf("SPEC-managed files cannot be edited as documents")
	}
	return Within(project, clean)
}

// ReadDocument returns a UTF-8 document and the digest used to detect
// concurrent changes when it is saved.
func ReadDocument(project, rel string) (string, string, error) {
	path, err := documentPath(project, rel)
	if err != nil {
		return "", "", err
	}
	data, err := readBytes(path, maxInput)
	if err != nil {
		return "", "", err
	}
	if !utf8.Valid(data) {
		return "", "", fmt.Errorf("%s is not UTF-8 text", rel)
	}
	return string(data), hash(data), nil
}

// WriteDocument saves content atomically when the file still has the digest
// that was read; an empty expected digest is a deliberate overwrite.
func WriteDocument(project, rel, content, expected string) (string, error) {
	path, err := documentPath(project, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", rel)
	}
	if expected != "" {
		current, err := readBytes(path, maxInput)
		if err != nil {
			return "", err
		}
		if hash(current) != expected {
			return "", fmt.Errorf("%s changed on disk since it was opened; nothing was saved", rel)
		}
	}
	if err = atomicFile(path, []byte(content), info.Mode().Perm()); err != nil {
		return "", err
	}
	return hash([]byte(content)), nil
}
