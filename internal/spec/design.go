package spec

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Design starters are DESIGN.md files embedded in the plugin. The TUI and the
// CLI copy one into a project; the spec-guide skill then personalizes it.
const starterDir = "skills/spec-guide/assets/design"

var starterID = regexp.MustCompile(`^[a-z][a-z0-9-]{1,40}$`)
var tokenLine = regexp.MustCompile(`^( *)([A-Za-z0-9_-]+):\s*(.*)$`)
var tokenReference = regexp.MustCompile("`?\\{[a-z]+\\.([A-Za-z0-9_-]+)\\}`?")

type Token struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DesignStarter struct {
	ID, Name, Description string
	Colors, Fonts         []Token
	Overview              string
	Content               string
}

// DesignStarters lists the embedded starters in a stable order.
func DesignStarters(plugin fs.FS) ([]DesignStarter, error) {
	names, err := fs.Glob(plugin, starterDir+"/*.md")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	starters := []DesignStarter{}
	for _, name := range names {
		data, err := fs.ReadFile(plugin, name)
		if err != nil {
			return nil, err
		}
		starter, err := parseStarter(strings.TrimSuffix(path.Base(name), ".md"), string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		starters = append(starters, starter)
	}
	return starters, nil
}

// parseStarter reads the subset of YAML front matter the starters use: flat
// fields plus two-level maps. It is not a general YAML parser.
func parseStarter(id, content string) (DesignStarter, error) {
	s := DesignStarter{ID: id, Content: content}
	if !starterID.MatchString(id) {
		return s, fmt.Errorf("invalid starter id")
	}
	front, body, ok := strings.Cut(strings.TrimPrefix(content, "---\n"), "\n---\n")
	if !strings.HasPrefix(content, "---\n") || !ok {
		return s, fmt.Errorf("missing YAML front matter")
	}
	group, entry := "", ""
	for _, line := range strings.Split(front, "\n") {
		match := tokenLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		indent, key, value := len(match[1]), match[2], unquote(match[3])
		switch indent {
		case 0:
			group = key
			switch key {
			case "name":
				s.Name = value
			case "description":
				s.Description = value
			}
		case 2:
			entry = key
			if group == "colors" {
				s.Colors = append(s.Colors, Token{key, value})
			}
		case 4:
			if group == "typography" && key == "fontFamily" && !containsToken(s.Fonts, value) {
				s.Fonts = append(s.Fonts, Token{entry, value})
			}
		}
	}
	if s.Name == "" || s.Description == "" || len(s.Colors) == 0 || len(s.Fonts) == 0 {
		return s, fmt.Errorf("starter needs a name, description, colors and typography")
	}
	if _, overview, ok := strings.Cut(body, "\n## Overview\n"); ok {
		paragraph, _, _ := strings.Cut(strings.TrimSpace(overview), "\n\n")
		// Previews read as prose: `{colors.primary}` becomes "primary".
		s.Overview = tokenReference.ReplaceAllString(strings.Join(strings.Fields(paragraph), " "), "$1")
	}
	return s, nil
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}

func containsToken(tokens []Token, value string) bool {
	for _, t := range tokens {
		if t.Value == value {
			return true
		}
	}
	return false
}

// CreateDesign copies a starter into the project's DESIGN.md. It never
// replaces an existing file; a preview writes nothing.
func (m *Manager) CreateDesign(project, id string, dry bool) (map[string]any, error) {
	project, err := Project(project)
	if err != nil {
		return nil, err
	}
	starters, err := DesignStarters(m.Plugin)
	if err != nil {
		return nil, err
	}
	var starter *DesignStarter
	for i := range starters {
		if starters[i].ID == id {
			starter = &starters[i]
		}
	}
	if starter == nil {
		return nil, fmt.Errorf("unknown design starter %q; run spec design --project DIR to list them", id)
	}
	target, err := Within(project, "DESIGN.md")
	if err != nil {
		return nil, err
	}
	if exists(target) {
		return nil, fmt.Errorf("DESIGN.md already exists; nothing was overwritten. Personalize it with the spec-guide skill instead")
	}
	result := map[string]any{"status": "planned", "starter": starter.ID, "starter_name": starter.Name, "path": "DESIGN.md", "existing_design_sources": append([]string{}, discover(project)["visual_design"]...)}
	if dry {
		return result, nil
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	_, err = f.WriteString(starter.Content)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(target)
		return nil, err
	}
	result["status"] = "created"
	result["personalized"] = false
	rel, _ := filepath.Rel(project, target)
	result["path"] = filepath.ToSlash(rel)
	return result, nil
}
