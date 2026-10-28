package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences belong to the person, not the project, so they live in the
// user's configuration directory. A missing or unreadable file means defaults.
type preferences struct {
	Editor string `json:"editor"`
}

var editorChoices = []string{"nano", "vim", "external"}

func preferencesPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "SPEC", "tui.json")
}

func loadPreferences() preferences {
	prefs := preferences{Editor: "nano"}
	if path := preferencesPath(); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(data, &prefs)
		}
	}
	valid := false
	for _, choice := range editorChoices {
		valid = valid || prefs.Editor == choice
	}
	if !valid {
		prefs.Editor = "nano"
	}
	return prefs
}

func savePreferences(prefs preferences) error {
	path := preferencesPath()
	if path == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(prefs, "", "  ")
	temp := path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func editorLabel(choice string) string {
	switch choice {
	case "vim":
		return "Vim keys"
	case "external":
		return "Your editor ($EDITOR)"
	}
	return "Nano keys"
}
