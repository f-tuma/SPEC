package spec

import (
	"fmt"
	"regexp"
	"strings"
)

// The document language is a project setting. SPEC's skills write project
// documents in it while the conversation keeps the user's language. An empty
// value follows the conversation.
var languageName = regexp.MustCompile(`^\p{L}[\p{L} ()-]{0,39}$`)
var configLanguage = regexp.MustCompile(`(?m)^[ \t]*Document language: ([^.\n]+)\.`)

// DocumentLanguage validates a CLI or TUI value; "conversation" clears it.
func DocumentLanguage(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "conversation") {
		return "", nil
	}
	if !languageName.MatchString(value) {
		return "", fmt.Errorf("document-language accepts a language name such as English or Czech, or conversation")
	}
	return value, nil
}

// LanguageLine is the OpenSpec context line that requests a document language.
func LanguageLine(language string) string {
	return "Document language: " + language + ". Write the brief, proposals, specs, design, tasks, reviews and checkpoints in " + language + "; keep code, identifiers and commands unchanged."
}

func bridgeFor(language string) string {
	if language == "" {
		return bridge
	}
	return strings.Replace(bridge, "context: |\n", "context: |\n  "+LanguageLine(language)+"\n", 1)
}

// configDocumentLanguage reports the language an existing OpenSpec config
// requests. It only reads the file; SPEC never edits an existing config.
func configDocumentLanguage(project string) string {
	for _, rel := range []string{"openspec/config.yaml", "openspec/config.yml"} {
		path, err := Within(project, rel)
		if err != nil || !exists(path) {
			continue
		}
		data, err := readBytes(path, 4<<20)
		if err != nil {
			return ""
		}
		if match := configLanguage.FindSubmatch(data); match != nil {
			return strings.TrimSpace(string(match[1]))
		}
		return ""
	}
	return ""
}
