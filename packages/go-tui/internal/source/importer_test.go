package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportURLsAndSkills(t *testing.T) {
	project := t.TempDir()
	preview, err := Import("https://example.com https://youtu.be/abc terminal-ui https://github.com/user/repo/tree/main/skills/writing", project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 4 {
		t.Fatalf("expected 4 sources, got %d", len(preview.Sources))
	}
	if preview.WebURLs != 1 || preview.YouTubeURLs != 1 || preview.Skills != 2 {
		t.Fatalf("unexpected counts: %+v", preview)
	}
}

func TestImportDefaultsCuratorSourceKind(t *testing.T) {
	sourceDir := t.TempDir()
	localFile := filepath.Join(sourceDir, "paper.md")
	if err := os.WriteFile(localFile, []byte("# Paper\n\nUseful."), 0o644); err != nil {
		t.Fatal(err)
	}
	article := "Captured Article\n\nThis pasted source has enough prose to be captured as local evidence. It covers design judgment, concrete examples, and working constraints with enough substance to pass the pasted-content detector."
	project := t.TempDir()
	preview, err := Import(strings.Join([]string{
		"https://example.com",
		"https://youtu.be/abc",
		localFile,
		"terminal-ui",
		article,
	}, "\n--- source ---\n"), project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 5 {
		t.Fatalf("expected 5 sources, got %d", len(preview.Sources))
	}
	for _, src := range preview.Sources {
		if src.Kind == nil || *src.Kind != "principle" {
			t.Fatalf("expected default principle kind for %#v", src)
		}
	}
}

func TestImportPreviewDoesNotCreateLocalSourceFolders(t *testing.T) {
	sourceDir := t.TempDir()
	localFile := filepath.Join(sourceDir, "paper.md")
	if err := os.WriteFile(localFile, []byte("# Paper\n\nUseful."), 0o644); err != nil {
		t.Fatal(err)
	}
	pastedArticle := "First Article\n\nThis article has enough prose to be treated as a captured source preview. It covers interaction timing, source boundaries, local evidence, and product judgment with enough words to pass the pasted-content detector."
	for name, input := range map[string]string{
		"urls":    "https://example.com https://youtu.be/abc",
		"article": pastedArticle,
		"file":    localFile,
	} {
		t.Run(name, func(t *testing.T) {
			project := t.TempDir()
			preview, err := Import(input, project, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(preview.Sources) == 0 {
				t.Fatalf("expected preview sources for %s", name)
			}
			if _, err := os.Stat(filepath.Join(project, "local-sources")); !os.IsNotExist(err) {
				t.Fatalf("preview should not create local-sources, stat err=%v", err)
			}
		})
	}
}

func TestImportCapturesMultipleArticleBlocks(t *testing.T) {
	project := t.TempDir()
	input := "First Article\n\nThis first article has enough prose to be captured as its own source. It talks about design systems, interaction timing, and product judgment.\n--- source ---\nSecond Article\n\nThis second article has enough prose to be captured as its own source. It talks about writing voice, examples, and editorial constraints."
	preview, err := Import(input, project, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(preview.Sources))
	}
	if preview.CapturedArticles != 2 || preview.LocalFiles != 2 {
		t.Fatalf("unexpected counts: %+v", preview)
	}
	for _, src := range preview.Sources {
		if src.Path == nil {
			t.Fatalf("captured source missing path: %+v", src)
		}
		if _, err := os.Stat(filepath.Join(project, *src.Path)); err != nil {
			t.Fatalf("captured file missing: %v", err)
		}
	}
}

func TestImportCopiesLocalFiles(t *testing.T) {
	project := t.TempDir()
	sourceDir := t.TempDir()
	file := filepath.Join(sourceDir, "paper.md")
	if err := os.WriteFile(file, []byte("# Paper\n\nUseful."), 0o644); err != nil {
		t.Fatal(err)
	}
	preview, err := Import(file, project, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(preview.Sources))
	}
	if preview.Sources[0].Path == nil || *preview.Sources[0].Path != "local-sources/paper.md" {
		t.Fatalf("unexpected local path: %+v", preview.Sources[0])
	}
	if _, err := os.Stat(filepath.Join(project, "local-sources", "paper.md")); err != nil {
		t.Fatal(err)
	}
}

func TestImportCopiesLocalFilesIntoCanonicalMixtapeLayout(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "liner.yaml"), []byte("version: 2\nartifact: liner\nmixtape: mixtape\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceDir := t.TempDir()
	file := filepath.Join(sourceDir, "paper.md")
	if err := os.WriteFile(file, []byte("# Paper\n\nUseful."), 0o644); err != nil {
		t.Fatal(err)
	}
	preview, err := Import(file, project, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Sources) != 1 {
		t.Fatalf("expected one source, got %d", len(preview.Sources))
	}
	if preview.Sources[0].Path == nil || *preview.Sources[0].Path != "local-sources/paper.md" {
		t.Fatalf("unexpected local path: %+v", preview.Sources[0])
	}
	if _, err := os.Stat(filepath.Join(project, "mixtape", "local-sources", "paper.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "local-sources", "paper.md")); !os.IsNotExist(err) {
		t.Fatalf("did not expect legacy local source path, stat err=%v", err)
	}
}

func TestIngestWritesManifestsAndSkillSnapshot(t *testing.T) {
	project := t.TempDir()
	items, warnings, err := Ingest("https://example.com terminal-ui", project)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 staged sources, got %d", len(items))
	}
	if err := WriteManifests(project, items); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"local-sources/sources-manifest.yaml",
		"local-sources/links.yaml",
		"local-sources/skills.yaml",
		"local-sources/skills/terminal-ui.md",
	} {
		if _, err := os.Stat(filepath.Join(project, rel)); err != nil {
			t.Fatalf("expected %s: %v", rel, err)
		}
	}
	links, err := os.ReadFile(filepath.Join(project, "local-sources", "links.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(links), "https://example.com") {
		t.Fatalf("links manifest missing URL:\n%s", string(links))
	}
}

func TestActiveSourcesFiltersInactiveStagedItems(t *testing.T) {
	project := t.TempDir()
	items, _, err := Ingest("https://example.com https://example.org", project)
	if err != nil {
		t.Fatal(err)
	}
	items[1].Active = false
	active := ActiveSources(items)
	if len(active) != 1 || active[0].URL != "https://example.com" {
		t.Fatalf("unexpected active sources: %+v", active)
	}
}
