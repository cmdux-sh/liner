package tape

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Source struct {
	ID          *string `yaml:"id,omitempty"`
	Type        string  `yaml:"type"`
	URL         string  `yaml:"url,omitempty"`
	Path        *string `yaml:"path,omitempty"`
	Citation    *string `yaml:"citation,omitempty"`
	Note        *string `yaml:"note,omitempty"`
	Section     *string `yaml:"section,omitempty"`
	Priority    string  `yaml:"priority"`
	Render      *string `yaml:"render,omitempty"`
	Kind        *string `yaml:"kind,omitempty"`
	ContentHash *string `yaml:"content_hash,omitempty"`
}

type Tape struct {
	Title                   string            `yaml:"title"`
	Description             string            `yaml:"description"`
	Version                 int               `yaml:"version"`
	Curator                 string            `yaml:"curator"`
	Sources                 []Source          `yaml:"sources"`
	Tags                    []string          `yaml:"tags,omitempty"`
	Created                 *string           `yaml:"created,omitempty"`
	Updated                 *string           `yaml:"updated,omitempty"`
	License                 *string           `yaml:"license,omitempty"`
	Homepage                *string           `yaml:"homepage,omitempty"`
	Mode                    *string           `yaml:"mode,omitempty"`
	JTBD                    *string           `yaml:"jtbd,omitempty"`
	JTBDClarifications      []Clarification   `yaml:"jtbd_clarifications,omitempty"`
	JTBDClarificationStatus *string           `yaml:"jtbd_clarification_status,omitempty"`
	MethodologyVersion      *string           `yaml:"methodology_version,omitempty"`
	Extra                   map[string]string `yaml:",inline,omitempty"`
}

type Clarification struct {
	Question string `yaml:"question"`
	Answer   string `yaml:"answer"`
}

type Project struct {
	Path        string
	RootPath    string
	TapePath    string
	SourcesDir  string
	WorkingDir  string
	LocalDir    string
	CapturedDir string
	SkillsDir   string
}

func ProjectAt(path string) Project {
	corpus := CorpusPath(path)
	return Project{
		Path:        corpus,
		RootPath:    path,
		TapePath:    filepath.Join(corpus, "tape.yaml"),
		SourcesDir:  filepath.Join(corpus, "sources"),
		WorkingDir:  filepath.Join(corpus, "working"),
		LocalDir:    filepath.Join(corpus, "local-sources"),
		CapturedDir: filepath.Join(corpus, "local-sources", "captured"),
		SkillsDir:   filepath.Join(corpus, "local-sources", "skills"),
	}
}

func CorpusPath(path string) string {
	canonical := filepath.Join(path, "mixtape")
	legacyTape := filepath.Join(path, "tape.yaml")
	canonicalTape := filepath.Join(canonical, "tape.yaml")
	if fileExists(canonicalTape) {
		return canonical
	}
	if fileExists(legacyTape) {
		return path
	}
	if fileExists(filepath.Join(path, "liner.yaml")) {
		return canonical
	}
	if dirExists(canonical) {
		return canonical
	}
	return path
}

func ReadProject(path string) (Tape, error) {
	data, err := os.ReadFile(ProjectAt(path).TapePath)
	if err != nil {
		return Tape{}, err
	}
	var t Tape
	if err := yaml.Unmarshal(data, &t); err != nil {
		return Tape{}, err
	}
	if t.Version == 0 {
		t.Version = 1
	}
	if t.PriorityDefaults(); t.Sources == nil {
		t.Sources = []Source{}
	}
	return t, nil
}

func WriteProject(path string, t Tape) error {
	if t.Title == "" {
		return errors.New("tape title is required")
	}
	if t.Version == 0 {
		t.Version = 1
	}
	t.PriorityDefaults()
	out, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	project := ProjectAt(path)
	if err := os.MkdirAll(filepath.Dir(project.TapePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(project.TapePath, out, 0o644)
}

func (t *Tape) PriorityDefaults() {
	for i := range t.Sources {
		if t.Sources[i].Priority == "" {
			t.Sources[i].Priority = "required"
		}
	}
}

func EnsureLocalFolders(path string) error {
	p := ProjectAt(path)
	for _, dir := range []string{p.LocalDir, p.CapturedDir, p.SkillsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
