package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type skillAlignmentFinding struct {
	Skill          string
	Status         string
	Evidence       string
	Recommendation string
}

func writeSkillCorpusAudit(project string) (auditFile, error) {
	skills, err := skillAuditInputs(project)
	if err != nil {
		return auditFile{}, err
	}
	if len(skills) == 0 {
		return auditFile{}, fmt.Errorf("No skills found in skills/. Add Markdown skills before running skill-corpus alignment.")
	}
	corpus, corpusInputs, err := skillAuditCorpus(project)
	if err != nil {
		return auditFile{}, err
	}
	findings, err := skillAlignmentFindings(skills, corpus)
	if err != nil {
		return auditFile{}, err
	}

	dir := filepath.Join(project, "working", "audits")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return auditFile{}, err
	}
	now := time.Now()
	name := now.Format("2006-01-02-150405") + "-skill-corpus-alignment"
	rel := filepath.Join("working", "audits", name+".md")
	path := filepath.Join(project, rel)
	if err := os.WriteFile(path, []byte(renderSkillCorpusAudit(now, skills, corpusInputs, findings)), 0o644); err != nil {
		return auditFile{}, err
	}
	return auditFile{
		Name:    name,
		RelPath: rel,
		Path:    path,
		Type:    "skill alignment",
		Updated: now.Format("2006-01-02"),
	}, nil
}

func skillAuditInputs(project string) ([]auditFile, error) {
	dir := filepath.Join(project, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []auditFile{}, nil
		}
		return nil, err
	}
	items := make([]auditFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		rel := filepath.Join("skills", entry.Name())
		items = append(items, auditFile{Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), RelPath: rel, Path: path, Updated: info.ModTime().Format("2006-01-02")})
	}
	sort.Slice(items, func(i int, j int) bool {
		return strings.ToLower(items[i].RelPath) < strings.ToLower(items[j].RelPath)
	})
	return items, nil
}

func skillAuditCorpus(project string) (string, []string, error) {
	var parts []string
	var inputs []string
	for _, rel := range []string{"LINER.md", "MIXTAPE.md", "synthesis.md", filepath.Join("working", "04-quality-checks.md")} {
		data, err := os.ReadFile(projectAbsPath(project, rel))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", nil, err
		}
		inputs = append(inputs, rel)
		parts = append(parts, string(data))
	}
	return strings.ToLower(strings.Join(parts, "\n")), inputs, nil
}

func skillAlignmentFindings(skills []auditFile, corpus string) ([]skillAlignmentFinding, error) {
	findings := make([]skillAlignmentFinding, 0, len(skills))
	for _, skill := range skills {
		data, err := os.ReadFile(skill.Path)
		if err != nil {
			return nil, err
		}
		body := string(data)
		lower := strings.ToLower(body)
		status := "aligned"
		evidence := "Grounding and boundary markers found."
		recommendation := "Keep this skill; re-audit after changing corpus sources."

		switch {
		case skillBodyDisabled(body):
			status = "disabled"
			evidence = "Managed disabled marker found."
			recommendation = "Keep disabled until the method is fixed, or re-enable through the Skills screen after review."
		case !hasSkillGrounding(lower):
			status = "needs grounding"
			evidence = "Missing Source Grounding or corpus/source reference."
			recommendation = "Add a Source Grounding section that names the corpus files or source notes the skill depends on."
		case !hasSkillBoundary(lower):
			status = "needs boundaries"
			evidence = "Missing boundary, scope, limitation, or abstention language."
			recommendation = "Add boundaries so external agents know when the skill should not be applied."
		case corpus != "" && !skillSharesCorpusTerms(skill.Name, lower, corpus):
			status = "weak corpus signal"
			evidence = "Skill wording has little overlap with the current corpus."
			recommendation = "Confirm this skill is actually supported by the mixtape, or add sources that ground it."
		}

		findings = append(findings, skillAlignmentFinding{
			Skill:          skill.RelPath,
			Status:         status,
			Evidence:       evidence,
			Recommendation: recommendation,
		})
	}
	return findings, nil
}

func hasSkillGrounding(body string) bool {
	return containsAny(body, []string{"source grounding", "grounded in", "mixtape.md", "liner.md", "synthesis.md", "source note", "sources/"})
}

func hasSkillBoundary(body string) bool {
	return containsAny(body, []string{"boundary", "boundaries", "scope", "limitation", "do not", "don't", "unless", "outside", "abstain"})
}

func skillSharesCorpusTerms(name string, body string, corpus string) bool {
	terms := significantTerms(name + " " + body)
	matches := 0
	for _, term := range terms {
		if strings.Contains(corpus, term) {
			matches++
		}
		if matches >= 3 {
			return true
		}
	}
	return false
}

func significantTerms(value string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, raw := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	}) {
		if len(raw) < 5 || skillAuditStopWords[raw] || seen[raw] {
			continue
		}
		seen[raw] = true
		terms = append(terms, raw)
	}
	sort.Strings(terms)
	if len(terms) > 30 {
		return terms[:30]
	}
	return terms
}

var skillAuditStopWords = map[string]bool{
	"about": true, "after": true, "again": true, "against": true, "agent": true,
	"apply": true, "before": true, "being": true, "corpus": true, "every": true,
	"grounding": true, "liner": true, "mixtape": true, "other": true, "rules": true,
	"should": true, "skill": true, "skills": true, "source": true, "sources": true,
	"their": true, "there": true, "these": true, "thing": true, "those": true,
	"under": true, "using": true, "where": true, "which": true, "without": true,
}

func renderSkillCorpusAudit(now time.Time, skills []auditFile, corpusInputs []string, findings []skillAlignmentFinding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Skill-Corpus Alignment Audit\n\nDate: %s\n\n", now.Format("2006-01-02 15:04:05"))
	b.WriteString("## Scope\n\n")
	b.WriteString("This local audit checks whether Markdown skills name their source grounding and boundaries before they are treated as project capabilities. It does not rewrite any project files.\n\n")
	b.WriteString("## Corpus Inputs\n\n")
	if len(corpusInputs) == 0 {
		b.WriteString("- No corpus-facing files were available; alignment is limited to skill self-description.\n")
	} else {
		for _, input := range corpusInputs {
			fmt.Fprintf(&b, "- `%s`\n", input)
		}
	}
	b.WriteString("\n## Skill Inputs\n\n")
	for _, skill := range skills {
		fmt.Fprintf(&b, "- `%s`\n", skill.RelPath)
	}
	b.WriteString("\n## Findings\n\n")
	b.WriteString("| Skill | Status | Evidence | Recommendation |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, finding := range findings {
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n",
			finding.Skill,
			escapeMarkdownTable(finding.Status),
			escapeMarkdownTable(finding.Evidence),
			escapeMarkdownTable(finding.Recommendation),
		)
	}
	b.WriteString("\n## Recommended Review\n\n")
	b.WriteString("- Add or tighten `Source Grounding` sections for every skill marked `needs grounding`.\n")
	b.WriteString("- Add boundaries for any skill that can overreach beyond the corpus.\n")
	b.WriteString("- Add or remove corpus sources before treating weakly grounded skills as authoritative.\n\n")
	b.WriteString("## Decision Log\n\n")
	b.WriteString("- No files were changed by this audit.\n")
	b.WriteString("- Apply decisions only after reviewing the evidence above.\n")
	return b.String()
}
