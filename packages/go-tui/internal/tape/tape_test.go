package tape

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureLocalFolders(t *testing.T) {
	project := t.TempDir()
	if err := EnsureLocalFolders(project); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"local-sources", "local-sources/captured", "local-sources/skills"} {
		if info, err := os.Stat(filepath.Join(project, rel)); err != nil || !info.IsDir() {
			t.Fatalf("%s was not created", rel)
		}
	}
}

func TestReadWriteProject(t *testing.T) {
	project := t.TempDir()
	srcPath := "local-sources/article.md"
	tape := Tape{
		Title:       "Demo",
		Description: "A useful demo",
		Version:     1,
		Curator:     "Arturo",
		Sources: []Source{{
			Type: "local_file",
			URL:  "",
			Path: &srcPath,
		}},
	}
	if err := WriteProject(project, tape); err != nil {
		t.Fatal(err)
	}
	got, err := ReadProject(project)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != tape.Title || len(got.Sources) != 1 {
		t.Fatalf("unexpected tape: %+v", got)
	}
	if got.Sources[0].Priority != "required" {
		t.Fatalf("expected priority default, got %q", got.Sources[0].Priority)
	}
	raw, err := os.ReadFile(filepath.Join(project, "tape.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "url:") {
		t.Fatalf("local_file source with an empty URL should omit url from tape.yaml:\n%s", raw)
	}
}

func TestProjectAtUsesCanonicalMixtapeLayout(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ProjectAt(project)
	if got.RootPath != project {
		t.Fatalf("unexpected root path: %s", got.RootPath)
	}
	if got.Path != filepath.Join(project, "mixtape") {
		t.Fatalf("expected canonical corpus path, got %s", got.Path)
	}

	tape := Tape{Title: "Demo", Description: "A useful demo", Version: 1, Curator: "Arturo", Sources: []Source{}}
	if err := WriteProject(project, tape); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "mixtape", "tape.yaml")); err != nil {
		t.Fatalf("expected v2 tape path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "tape.yaml")); !os.IsNotExist(err) {
		t.Fatalf("did not expect legacy root tape, stat err=%v", err)
	}
}

func TestProjectAtKeepsLegacyRootLayoutAfterLinerMetadata(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "tape.yaml"), []byte("title: Legacy\nversion: 1\ncurator: Arturo\ndescription: Demo\nsources: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := ProjectAt(project)
	if got.Path != project {
		t.Fatalf("expected legacy corpus path, got %s", got.Path)
	}
	if got.TapePath != filepath.Join(project, "tape.yaml") {
		t.Fatalf("expected legacy tape path, got %s", got.TapePath)
	}
	if _, err := ReadProject(project); err != nil {
		t.Fatalf("expected legacy project with liner.yaml to read: %v", err)
	}
}
