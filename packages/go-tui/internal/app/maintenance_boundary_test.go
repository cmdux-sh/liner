package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCoveredMaintenancePathsHaveNoCanonicalProjectWriter(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	appDir := filepath.Dir(currentFile)
	assertFileOmits(t, filepath.Join(appDir, "sources.go"), "source.AppendToTape", "tape.WriteProject")
	assertFileOmits(t, filepath.Join(appDir, "maintenance.go"), "os.WriteFile", "os.Remove", "tape.WriteProject", "yaml.Marshal")
	assertFileOmits(t, filepath.Join(appDir, "..", "source", "importer.go"), "func AppendToTape", "func AppendActiveToTape")

	compileBody := readTestFile(t, filepath.Join(appDir, "compile.go"))
	start := strings.Index(compileBody, "func (m Model) dropSelectedCompileWarningSource")
	if start < 0 {
		t.Fatal("could not isolate the covered compile Source removal path")
	}
	end := strings.Index(compileBody[start:], "func (m *Model) recordCompileProgress")
	if end < 0 {
		t.Fatal("could not isolate the covered compile Source removal path")
	}
	covered := compileBody[start : start+end]
	for _, forbidden := range []string{"tape.WriteProject", "os.WriteFile", "os.Remove", "dropTapeSourceByID"} {
		if strings.Contains(covered, forbidden) {
			t.Fatalf("covered Source removal path contains independent writer %q:\n%s", forbidden, covered)
		}
	}
}

func TestSourceNoteCleanupNeverImplicitlyApprovesCorePlan(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test source path")
	}
	body := readTestFile(t, filepath.Join(filepath.Dir(currentFile), "audits_source_notes.go"))
	if strings.Contains(body, "ApplyMaintenance(project, plan, plan.ApprovalRequired)") {
		t.Fatal("source-note cleanup must not derive user approval from the Core plan itself")
	}
	for _, expected := range []string{"if plan.ApprovalRequired", "ApplyMaintenance(project, plan, false)"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("source-note cleanup is missing fail-closed approval behavior %q", expected)
		}
	}
}

func assertFileOmits(t *testing.T, path string, forbidden ...string) {
	t.Helper()
	body := readTestFile(t, path)
	for _, value := range forbidden {
		if strings.Contains(body, value) {
			t.Fatalf("%s contains retired maintenance writer %q", path, value)
		}
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
