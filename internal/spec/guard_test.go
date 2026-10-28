package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func guardFixture(t *testing.T) (string, MappingConfig) {
	t.Helper()
	root := t.TempDir()
	for _, path := range []string{"AGENTS.md", "README.md", "package.json"} {
		data := "# Source\nActual accepted purpose and commands.\n"
		if path == "package.json" {
			data = `{"scripts":{"test":"touch SHOULD_NOT_EXIST"}}`
		}
		put(t, filepath.Join(root, path), data)
	}
	config, err := Adopt(root, true)
	if err != nil {
		t.Fatal(err)
	}
	return root, config
}
func hasRule(report map[string]any, rule string) bool {
	for _, finding := range report["findings"].([]Finding) {
		if finding.Rule == rule {
			return true
		}
	}
	return false
}
func TestAuditReadsCandidatesWithoutExecutingCommands(t *testing.T) {
	root, _ := guardFixture(t)
	before, _ := Digest(root, Mapping)
	audit := Audit(root)
	after, _ := Digest(root, Mapping)
	if before != after || audit["semantic_result"] != "unknown" || !reflect.DeepEqual(audit["script_names"], []string{"test"}) || exists(filepath.Join(root, "SHOULD_NOT_EXIST")) {
		t.Fatal(audit)
	}
}
func TestDiscoveryUsesEquivalentSourcesAndPrefersExplicitBrief(t *testing.T) {
	root, _ := guardFixture(t)
	put(t, filepath.Join(root, "docs/project-brief.md"), "user intent")
	put(t, filepath.Join(root, "docs/ui/design-system.md"), "canonical tokens")
	put(t, filepath.Join(root, "openspec/specs/rules/spec.md"), "canonical behavior")
	docs := discover(root)
	for role, path := range map[string]string{"brief": "docs/project-brief.md", "visual_design": "docs/ui/design-system.md", "specification": "openspec/specs/rules/spec.md"} {
		if !reflect.DeepEqual(docs[role], []string{path}) {
			t.Fatal(docs)
		}
	}
	if err := os.Symlink("AGENTS.md", filepath.Join(root, "CLAUDE.md")); err == nil {
		if !reflect.DeepEqual(discover(root)["instructions"], []string{"AGENTS.md"}) {
			t.Fatal("instruction alias followed")
		}
	}
}
func TestAdoptIsDraftAndNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	if _, err := Adopt(root, false); err != nil || exists(filepath.Join(root, Mapping)) {
		t.Fatal("draft wrote file")
	}
	if _, err := Adopt(root, true); err != nil {
		t.Fatal(err)
	}
	before, _ := Digest(root, Mapping)
	if _, err := Adopt(root, true); err == nil {
		t.Fatal("mapping overwritten")
	}
	after, _ := Digest(root, Mapping)
	if before != after {
		t.Fatal("mapping changed")
	}
}
func TestCheckIsMechanicalAndPhaseSpecific(t *testing.T) {
	root, config := guardFixture(t)
	report, err := Check(root, "orient", false, "")
	if err != nil || report["mechanical_result"] != "pass" || report["semantic_result"] != "unknown" {
		t.Fatalf("%v %v", report, err)
	}
	report, _ = Check(root, "implement", false, "")
	if !hasRule(report, "SP-DOCUMENT-ROLE") {
		t.Fatal("orientation treated as implementation readiness")
	}
	report, _ = Check(root, "orient", true, "")
	if !hasRule(report, "SP-DOCUMENT-ROLE") {
		t.Fatal("UI needs design mapping")
	}
	put(t, filepath.Join(root, "style-rules.txt"), "Equivalent design rules")
	config.Documents["visual_design"] = []string{"style-rules.txt"}
	atomicJSON(filepath.Join(root, Mapping), config)
	report, _ = Check(root, "orient", true, "")
	if report["mechanical_result"] != "pass" {
		t.Fatal(report)
	}
}
func TestEmptyAndPlaceholderDocumentsFail(t *testing.T) {
	for _, content := range []string{"", " \n", "# Heading\n## Only heading", "# Intent\n<fill in>", "TODO: write brief", "[TODO fill in]", "- TBD"} {
		t.Run(fmt.Sprintf("%q", content), func(t *testing.T) {
			root, _ := guardFixture(t)
			put(t, filepath.Join(root, "README.md"), content)
			report, _ := Check(root, "orient", false, "")
			if !hasRule(report, "SP-DOCUMENT-EMPTY") {
				t.Fatal(report)
			}
		})
	}
	root, _ := guardFixture(t)
	put(t, filepath.Join(root, "README.md"), "Actual accepted purpose.\nTODO: optional detail")
	report, _ := Check(root, "orient", false, "")
	if report["mechanical_result"] != "pass" {
		t.Fatal(report)
	}
}
func TestMalformedMappingsCannotPass(t *testing.T) {
	for _, raw := range []string{"{", "[]", "null", `{"schema_version":true}`, `{"schema_version":1,"mode":"bad"}`, `{"schema_version":1,"mode":"pilot","goal":"g","stop_condition":"s","documents":{"brief":[]}}`, `{"schema_version":1,"mode":"pilot","goal":"g","stop_condition":"s","documents":{},"checks":{"test":"sh command"}}`} {
		t.Run(raw, func(t *testing.T) {
			root, _ := guardFixture(t)
			put(t, filepath.Join(root, Mapping), raw)
			report, _ := Check(root, "orient", false, "")
			if !hasRule(report, "SP-CONFIG") {
				t.Fatal(report)
			}
		})
	}
}
func TestAllMappedPathsAreChecked(t *testing.T) {
	for _, path := range []string{"missing", "../outside", "bad\x00path", "/absolute"} {
		t.Run(fmt.Sprintf("%q", path), func(t *testing.T) {
			root, config := guardFixture(t)
			config.Documents["extra"] = []string{path}
			atomicJSON(filepath.Join(root, Mapping), config)
			report, _ := Check(root, "orient", false, "")
			if !hasRule(report, "SP-DOCUMENT-PATH") {
				t.Fatal(report)
			}
		})
	}
	root, _ := guardFixture(t)
	os.WriteFile(filepath.Join(root, "README.md"), []byte{255}, 0644)
	report, _ := Check(root, "orient", false, "")
	if !hasRule(report, "SP-DOCUMENT-PATH") {
		t.Fatal("invalid UTF-8 accepted")
	}
}
func TestOversizedAndSymlinkInputsRejected(t *testing.T) {
	root, _ := guardFixture(t)
	put(t, filepath.Join(root, "large"), strings.Repeat("x", maxInput+1))
	if _, err := Digest(root, "large"); err == nil {
		t.Fatal("large input accepted")
	}
	if err := os.Symlink("README.md", filepath.Join(root, "link")); err == nil {
		if _, err := Digest(root, "link"); err == nil {
			t.Fatal("symlink input accepted")
		}
	}
}
func TestEvidenceHashesInvalidateChangedSubjectAndArtifact(t *testing.T) {
	for _, changed := range []string{"README.md", "output.txt"} {
		t.Run(changed, func(t *testing.T) {
			root, _ := guardFixture(t)
			put(t, filepath.Join(root, "output.txt"), "actual fixture output")
			subject, _ := Digest(root, "README.md")
			artifact, _ := Digest(root, "output.txt")
			receipt := Evidence{1, map[string]string{"README.md": subject}, map[string]string{"output.txt": artifact}}
			findings, err := CheckEvidence(root, receipt)
			if err != nil || len(findings) != 0 {
				t.Fatalf("%v %v", findings, err)
			}
			put(t, filepath.Join(root, changed), "changed bytes")
			findings, err = CheckEvidence(root, receipt)
			if err != nil || len(findings) != 1 || findings[0].Rule != "SP-EVIDENCE-STALE" {
				t.Fatalf("%v %v", findings, err)
			}
		})
	}
}
func TestEvidenceMissingInvalidAndTraversalStayFailures(t *testing.T) {
	root, _ := guardFixture(t)
	for _, receipt := range []Evidence{{1, nil, nil}, {1, map[string]string{"README.md": "fake"}, map[string]string{"../outside": strings.Repeat("0", 64)}}} {
		findings, err := CheckEvidence(root, receipt)
		if err != nil || len(findings) != 2 {
			t.Fatalf("%v %v", findings, err)
		}
	}
	put(t, filepath.Join(root, "receipt.json"), `{"schema_version":1,"subject":{},"artifacts":{},"passed":true,"tests_executed":100}`)
	report, _ := Check(root, "orient", false, "receipt.json")
	if report["mechanical_result"] != "fail" || report["semantic_result"] != "unknown" {
		t.Fatal(report)
	}
}
func TestFindingIDsAreStable(t *testing.T) {
	root, _ := guardFixture(t)
	os.Remove(filepath.Join(root, "README.md"))
	first, _ := Check(root, "orient", false, "")
	second, _ := Check(root, "orient", false, "")
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("unstable findings")
	}
}
