package main

import (
	"encoding/json"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/f-tuma/SPEC/internal/spec"
)

func TestEmbeddedPluginHasCompleteIdentityAndResources(t *testing.T) {
	plugin, _ := fs.Sub(resources, "plugin")
	var portable map[string]any
	for _, name := range []string{"plugin.json", ".claude-plugin/plugin.json", ".codex-plugin/plugin.json"} {
		data, err := fs.ReadFile(plugin, name)
		if err != nil {
			t.Fatal(err)
		}
		var manifest map[string]any
		if err = json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest["name"] != "spec" || manifest["version"] != spec.Version {
			t.Fatal("manifest drift")
		}
		if name == "plugin.json" {
			portable = manifest
		}
	}
	if portable["$schema"] != "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json" {
		t.Fatal("wrong portable manifest")
	}
	skills, _ := fs.Glob(plugin, "skills/*/SKILL.md")
	if len(skills) != 3 {
		t.Fatal("missing skills")
	}
	link := regexp.MustCompile(`\[[^\]]+\]\(([^\s)]+)\)`)
	err := fs.WalkDir(plugin, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(name, ".py") || strings.HasSuffix(name, ".pyc") {
			t.Fatalf("Python runtime resource retained: %s", name)
		}
		if !strings.HasSuffix(name, ".md") {
			return nil
		}
		data, err := fs.ReadFile(plugin, name)
		if err != nil {
			return err
		}
		for _, match := range link.FindAllStringSubmatch(string(data), -1) {
			target := strings.Split(match[1], "#")[0]
			if strings.HasPrefix(target, "http:") || strings.HasPrefix(target, "https:") || target == "" {
				continue
			}
			target = path.Clean(path.Join(path.Dir(name), target))
			if _, err := fs.Stat(plugin, target); err != nil {
				t.Fatalf("broken resource link: %s -> %s", name, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
