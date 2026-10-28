package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const Mapping = ".spec-pilot.json"
const maxInput = 2 << 20

var phases = map[string][]string{
	"orient":    {"instructions", "brief", "commands"},
	"design":    {"instructions", "brief", "architecture", "specification"},
	"implement": {"instructions", "specification", "technical_design", "tasks", "commands"},
	"verify":    {"instructions", "specification", "commands"},
	"release":   {"instructions", "specification", "commands", "release"},
}
var candidates = map[string][]string{
	"instructions":  {"AGENTS.md", "CLAUDE.md"},
	"brief":         {"docs/project-brief.md", "PROJECT.md", "README.md"},
	"architecture":  {"docs/architecture.md", "ARCHITECTURE.md", "packages/docs/src/content/docs/architecture/overview.mdx"},
	"specification": {"docs/specification.md", "docs/sync/README.md"},
	"commands":      {"package.json", "go.mod", "pyproject.toml", "Makefile", "Taskfile.yml", "Taskfile.yaml", "docs/testing/behavior-scenarios.md"},
	"visual_design": {"DESIGN.md", "docs/ui/design-system.md"},
	"session":       {"docs/work-session.md"},
	"release":       {"docs/releasing.md", "RELEASING.md"},
}

type MappingConfig struct {
	SchemaVersion int                 `json:"schema_version"`
	Mode          string              `json:"mode"`
	Goal          string              `json:"goal"`
	StopCondition string              `json:"stop_condition"`
	Documents     map[string][]string `json:"documents"`
	Checks        map[string][]string `json:"checks"`
}
type Finding struct {
	ID       string `json:"id"`
	Rule     string `json:"rule"`
	Subject  string `json:"subject"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Severity string `json:"severity"`
}

func finding(rule, subject, expected, actual string) Finding {
	return Finding{rule + ":" + subject, rule, subject, expected, actual, "error"}
}
func inputBytes(root, rel string) ([]byte, error) {
	path, err := Within(root, rel)
	if err != nil {
		return nil, err
	}
	return readBytes(path, maxInput)
}
func inputJSON(root, rel string, value any) error {
	data, err := inputBytes(root, rel)
	if err != nil {
		return err
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("input is not valid UTF-8")
	}
	if strings.TrimSpace(string(data)) == "null" {
		return fmt.Errorf("expected a JSON object")
	}
	return json.Unmarshal(data, value)
}
func Digest(root, rel string) (string, error) {
	data, err := inputBytes(root, rel)
	if err != nil {
		return "", err
	}
	return hash(data), nil
}

func discover(root string) map[string][]string {
	docs := map[string][]string{}
	for role, paths := range candidates {
		for _, path := range paths {
			if _, err := inputBytes(root, path); err == nil {
				docs[role] = append(docs[role], path)
				if role == "brief" {
					break
				}
			}
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, "openspec/specs/*/spec.md"))
	specs := []string{}
	for _, path := range matches {
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if _, err := inputBytes(root, rel); err == nil {
			specs = append(specs, rel)
		}
	}
	if len(specs) > 0 {
		sort.Strings(specs)
		docs["specification"] = specs
	}
	return docs
}
func Audit(root string) map[string]any {
	docs := discover(root)
	result := map[string]any{"schema_version": 1, "operation": "audit", "documents": docs, "semantic_result": "unknown", "limitations": []string{"Candidates are file mappings, not accepted requirements.", "Only documented candidate locations were inspected; this is not a full repository scan.", "No project command, model review or browser was executed."}}
	for _, path := range docs["commands"] {
		if path == "package.json" {
			var pkg map[string]any
			if err := inputJSON(root, path, &pkg); err != nil {
				result["limitations"] = append(result["limitations"].([]string), err.Error())
				continue
			}
			names := []string{}
			if scripts, ok := pkg["scripts"].(map[string]any); ok {
				for name := range scripts {
					names = append(names, name)
				}
			}
			sort.Strings(names)
			result["package_manager"] = pkg["packageManager"]
			result["script_names"] = names
		}
	}
	return result
}
func Adopt(root string, write bool) (MappingConfig, error) {
	config := MappingConfig{1, "pilot", "Map existing project sources and identify the next bounded step", "Finish orientation; do not start product delivery", discover(root), map[string][]string{}}
	if write {
		path, err := Within(root, Mapping)
		if err != nil {
			return config, err
		}
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return config, err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return config, err
		}
		defer file.Close()
		_, err = file.Write(append(data, '\n'))
		if err != nil {
			return config, err
		}
	}
	return config, nil
}
func validateMapping(config MappingConfig) error {
	if config.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be integer 1")
	}
	if config.Mode != "methodology" && config.Mode != "pilot" && config.Mode != "delivery" {
		return fmt.Errorf("mode must be methodology, pilot or delivery")
	}
	if strings.TrimSpace(config.Goal) == "" || strings.TrimSpace(config.StopCondition) == "" {
		return fmt.Errorf("goal and stop_condition must be nonempty strings")
	}
	if config.Documents == nil {
		return fmt.Errorf("documents must map roles to nonempty lists of relative file paths")
	}
	for role, paths := range config.Documents {
		if strings.TrimSpace(role) == "" || len(paths) == 0 {
			return fmt.Errorf("each document role needs a nonempty list")
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				return fmt.Errorf("empty mapped path")
			}
		}
	}
	if config.Checks == nil {
		return fmt.Errorf("checks must be an object when provided")
	}
	for name, args := range config.Checks {
		if name == "" || len(args) == 0 {
			return fmt.Errorf("each check needs a nonempty argument list")
		}
		for _, arg := range args {
			if arg == "" {
				return fmt.Errorf("empty command argument")
			}
		}
	}
	return nil
}

var placeholder = regexp.MustCompile(`(?i)^(?:[-*]\s*)?(?:<[^>]+>|\[TODO[^\]]*\]|TODO(?::.*)?|TBD|FILL-IN)$`)

func emptyDocument(data []byte) bool {
	lines := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return true
	}
	for _, line := range lines {
		if !placeholder.MatchString(line) {
			return false
		}
	}
	return true
}

type Evidence struct {
	SchemaVersion int               `json:"schema_version"`
	Subject       map[string]string `json:"subject"`
	Artifacts     map[string]string `json:"artifacts"`
}

func CheckEvidence(root string, receipt Evidence) ([]Finding, error) {
	if receipt.SchemaVersion != 1 {
		return nil, fmt.Errorf("evidence schema_version must be integer 1")
	}
	issues := []Finding{}
	for _, field := range []string{"subject", "artifacts"} {
		values := receipt.Subject
		if field == "artifacts" {
			values = receipt.Artifacts
		}
		if len(values) == 0 {
			issues = append(issues, finding("SP-EVIDENCE-MISSING", field, "Nonempty path-to-SHA256 map", "Missing or empty"))
			continue
		}
		paths := []string{}
		for path := range values {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			expected := values[path]
			label := field + "/" + path
			if !fingerprint.MatchString(expected) {
				issues = append(issues, finding("SP-EVIDENCE-HASH", label, "Lowercase SHA-256", "Invalid hash"))
				continue
			}
			current, err := Digest(root, path)
			if err != nil {
				issues = append(issues, finding("SP-EVIDENCE-PATH", label, "Readable contained file", err.Error()))
			} else if current != expected {
				issues = append(issues, finding("SP-EVIDENCE-STALE", label, expected, current))
			}
		}
	}
	return issues, nil
}
func Check(root, phase string, ui bool, evidence string) (map[string]any, error) {
	required, ok := phases[phase]
	if !ok {
		return nil, fmt.Errorf("unknown phase %q", phase)
	}
	required = append([]string{}, required...)
	if ui {
		required = append(required, "visual_design")
	}
	issues := []Finding{}
	config := MappingConfig{Checks: map[string][]string{}}
	err := inputJSON(root, Mapping, &config)
	if err == nil {
		err = validateMapping(config)
	}
	if err != nil {
		issues = append(issues, finding("SP-CONFIG", Mapping, "Valid explicit project mapping", err.Error()))
	} else {
		for _, role := range required {
			if _, ok := config.Documents[role]; !ok {
				issues = append(issues, finding("SP-DOCUMENT-ROLE", role, "Mapped relevant document", "Role is not mapped"))
			}
		}
		roles := []string{}
		for role := range config.Documents {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		for _, role := range roles {
			for _, rel := range config.Documents[role] {
				data, err := inputBytes(root, rel)
				if err == nil && !utf8.Valid(data) {
					err = fmt.Errorf("input is not valid UTF-8")
				}
				if err != nil {
					issues = append(issues, finding("SP-DOCUMENT-PATH", role+"/"+rel, "Readable contained UTF-8 file", err.Error()))
				} else if emptyDocument(data) {
					issues = append(issues, finding("SP-DOCUMENT-EMPTY", role+"/"+rel, "Substantive content", "Empty, headings only or placeholder only"))
				}
			}
		}
	}
	if evidence != "" {
		var receipt Evidence
		err := inputJSON(root, evidence, &receipt)
		if err == nil {
			var findings []Finding
			findings, err = CheckEvidence(root, receipt)
			issues = append(issues, findings...)
		}
		if err != nil {
			issues = append(issues, finding("SP-EVIDENCE", evidence, "Valid evidence object", err.Error()))
		}
	}
	mechanical := "pass"
	if len(issues) > 0 {
		mechanical = "fail"
	}
	return map[string]any{"schema_version": 1, "operation": "check", "phase": phase, "mechanical_result": mechanical, "semantic_result": "unknown", "findings": issues, "limitations": []string{"Existence and simple placeholders do not establish document quality or requirement completeness.", "Supplied hashes check only declared files; omitted dependencies are not detected.", "Execution, test counts, outcome interpretation, provenance and human approvals are not authenticated.", "This result does not authorize implementation, merge, publication or deployment."}}, nil
}
