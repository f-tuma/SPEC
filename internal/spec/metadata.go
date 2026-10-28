package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type metadataFile struct {
	relative string
	value    any
}

func metadataFiles(project string, state *State) []metadataFile {
	relative := "./versions/" + state.Current.Directory + "/plugin"
	name := MarketplaceName(project)
	return []metadataFile{
		{".spec/marketplace/.agents/plugins/marketplace.json", map[string]any{"name": name, "interface": map[string]string{"displayName": "SPEC local"}, "plugins": []any{map[string]any{"name": "spec", "source": map[string]string{"source": "local", "path": relative}, "policy": map[string]string{"installation": "AVAILABLE", "authentication": "ON_INSTALL"}, "category": "Productivity"}}}},
		{".spec/marketplace/.claude-plugin/marketplace.json", map[string]any{"name": name, "owner": map[string]string{"name": "f-tuma"}, "plugins": []any{map[string]string{"name": "spec", "source": relative, "description": "SPEC coding workflow"}}}},
		{".spec/installation.json", state},
	}
}

// Prepare all targets and backups before changing any active catalog or receipt.
// A failed rename restores committed files; crash-time drift is detected by status.
func (m *Manager) commitState(project string, state *State) error {
	type pending struct {
		target         string
		data           []byte
		old            []byte
		existed        bool
		staged, backup string
		keepBackup     bool
	}
	files := []*pending{}
	defer func() {
		for _, f := range files {
			if f.staged != "" {
				os.Remove(f.staged)
			}
			if f.backup != "" && !f.keepBackup {
				os.Remove(f.backup)
			}
		}
	}()
	for _, item := range metadataFiles(project, state) {
		target, err := Within(project, item.relative)
		if err != nil {
			return err
		}
		data, err := json.MarshalIndent(item.value, "", "  ")
		if err != nil {
			return err
		}
		f := &pending{target: target, data: append(data, '\n'), existed: exists(target)}
		if f.existed {
			f.old, err = readBytes(target, 4<<20)
			if err != nil {
				return fmt.Errorf("metadata target conflict: %w", err)
			}
		}
		files = append(files, f)
	}
	stage := func(target string, data []byte) (string, error) {
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return "", err
		}
		f, err := os.CreateTemp(filepath.Dir(target), ".spec-metadata-")
		if err != nil {
			return "", err
		}
		name := f.Name()
		if _, err = f.Write(data); err == nil {
			err = f.Chmod(0644)
		}
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(name)
			return "", err
		}
		return name, nil
	}
	for _, f := range files {
		var err error
		f.staged, err = stage(f.target, f.data)
		if err != nil {
			return err
		}
		if f.existed {
			f.backup, err = stage(f.target, f.old)
			if err != nil {
				return err
			}
		}
	}
	for i, f := range files {
		if err := m.Rename(f.staged, f.target); err != nil {
			failures := []error{fmt.Errorf("metadata commit failed: %w", err)}
			for j := i - 1; j >= 0; j-- {
				old := files[j]
				var restore error
				if old.existed {
					restore = m.Rename(old.backup, old.target)
				} else {
					restore = os.Remove(old.target)
				}
				if restore != nil {
					old.keepBackup = true
					failures = append(failures, fmt.Errorf("recovery failed for %s; backup %s: %w", old.target, old.backup, restore))
				}
			}
			return errors.Join(failures...)
		}
	}
	return nil
}

func verifyCatalogs(project string, state *State) error {
	for _, file := range metadataFiles(project, state)[:2] {
		path, err := Within(project, file.relative)
		if err != nil {
			return err
		}
		var actual, expected any
		if err = readJSON(path, &actual); err != nil {
			return err
		}
		data, _ := json.Marshal(file.value)
		json.Unmarshal(data, &expected)
		if !reflect.DeepEqual(actual, expected) {
			return fmt.Errorf("catalog differs from installed receipt: %s", file.relative)
		}
	}
	return nil
}
