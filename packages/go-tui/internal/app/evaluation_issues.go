package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sourcepkg "github.com/cmdux/liner/packages/go-tui/internal/source"
	"github.com/cmdux/liner/packages/go-tui/internal/tape"
	"gopkg.in/yaml.v3"
)

type evaluationIssueSummary struct {
	DroppedCandidates int
	DroppedYouTube    int
	TranscriptIssues  int
	UnavailableIssues int
	CustomSources     int
	CustomYouTube     int
	MissingCustom     int
	MissingCustomYT   int
	DroppedCustom     int
	DroppedCustomYT   int
	AcceptedIssues    int
	AcceptedIssuesYT  int
}

type localSourceManifest struct {
	Sources []sourcepkg.StagedSource `yaml:"sources"`
}

type candidateIssue struct {
	Dropped     bool
	Transcript  bool
	Unavailable bool
	YouTube     bool
}

var youtubeVideoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func readEvaluationIssueSummary(project string, current tape.Tape) (evaluationIssueSummary, bool) {
	if strings.TrimSpace(project) == "" {
		return evaluationIssueSummary{}, false
	}
	summary := evaluationIssueSummary{}

	candidateIssues := readCandidateIssues(project, &summary)
	accepted := current.Sources
	if len(accepted) == 0 {
		if fromDisk, err := tape.ReadProject(project); err == nil {
			accepted = fromDisk.Sources
		}
	}
	acceptedKeys := sourceKeySet(accepted)
	for _, src := range accepted {
		if sourceHasWarningNote(src) {
			summary.AcceptedIssues++
			if isYouTubeSource(src) {
				summary.AcceptedIssuesYT++
			}
		}
	}

	for _, item := range readLocalSourceManifest(project) {
		if !item.Active {
			continue
		}
		summary.CustomSources++
		if isYouTubeSource(item.Source) {
			summary.CustomYouTube++
		}
		keys := issueKeysForSource(item.Source)
		if keySetContainsAny(acceptedKeys, keys) {
			continue
		}
		summary.MissingCustom++
		if isYouTubeSource(item.Source) {
			summary.MissingCustomYT++
		}
		if issue, ok := candidateIssueForKeys(candidateIssues, keys); ok && issue.Dropped && (issue.Transcript || issue.Unavailable) {
			summary.DroppedCustom++
			if issue.YouTube {
				summary.DroppedCustomYT++
			}
		}
	}

	return summary, summary.HasIssues()
}

func (s evaluationIssueSummary) HasIssues() bool {
	return s.DroppedCustom > 0 ||
		s.MissingCustom > 0 ||
		s.AcceptedIssues > 0 ||
		s.DroppedYouTube > 0 ||
		s.TranscriptIssues > 0 ||
		s.UnavailableIssues > 0
}

func (s evaluationIssueSummary) Display(project string) string {
	if !s.HasIssues() {
		return "none"
	}
	parts := []string{}
	if s.DroppedCustomYT > 0 {
		parts = append(parts, "transcript/access: "+intLabel(s.DroppedCustomYT, "custom YouTube source")+" dropped")
	} else if s.DroppedCustom > 0 {
		parts = append(parts, "source access: "+intLabel(s.DroppedCustom, "custom source")+" dropped")
	}
	if missingOther := s.MissingCustom - s.DroppedCustom; missingOther > 0 {
		missingYouTubeOther := s.MissingCustomYT - s.DroppedCustomYT
		if missingYouTubeOther > 0 {
			parts = append(parts, intLabel(missingYouTubeOther, "custom YouTube source")+" not in tape")
		} else {
			parts = append(parts, intLabel(missingOther, "custom source")+" not in tape")
		}
	}
	if s.AcceptedIssuesYT > 0 {
		parts = append(parts, intLabel(s.AcceptedIssuesYT, "accepted YouTube source")+" needs review")
	} else if s.AcceptedIssues > 0 {
		parts = append(parts, intLabel(s.AcceptedIssues, "accepted source")+" needs review")
	}
	if len(parts) == 0 {
		if s.DroppedYouTube > 0 {
			if s.TranscriptIssues > 0 {
				parts = append(parts, "transcript/access: "+intLabel(s.DroppedYouTube, "YouTube candidate")+" dropped")
			} else {
				parts = append(parts, intLabel(s.DroppedYouTube, "YouTube candidate")+" dropped")
			}
		} else if s.UnavailableIssues > 0 {
			parts = append(parts, intLabel(s.UnavailableIssues, "unavailable candidate"))
		}
	}
	rel := s.evidencePath(project)
	return fmt.Sprintf("%s; see %s", strings.Join(parts, "; "), rel)
}

func (s evaluationIssueSummary) evidencePath(project string) string {
	if s.DroppedCustom > 0 || s.DroppedYouTube > 0 || s.TranscriptIssues > 0 || s.UnavailableIssues > 0 {
		return displayProjectPath(project, filepath.Join("working", "03-evaluation.yaml"))
	}
	if s.MissingCustom > 0 {
		return displayProjectPath(project, filepath.Join("local-sources", "sources-manifest.yaml"))
	}
	return displayProjectPath(project, "tape.yaml")
}

func readCandidateIssues(project string, summary *evaluationIssueSummary) map[string]candidateIssue {
	data, err := os.ReadFile(projectAbsPath(project, filepath.Join("working", "03-evaluation.yaml")))
	if err != nil {
		return map[string]candidateIssue{}
	}
	var artifact methodologyEvaluationArtifact
	if err := yaml.Unmarshal(data, &artifact); err != nil || len(artifact.Candidates) == 0 {
		return map[string]candidateIssue{}
	}
	out := map[string]candidateIssue{}
	for _, candidate := range artifact.Candidates {
		if strings.ToLower(strings.TrimSpace(candidate.Decision)) != "dropped" {
			continue
		}
		summary.DroppedCandidates++
		reason := evaluationCandidateReason(candidate)
		youTube := isYouTubeURL(candidate.URL)
		transcriptIssue := youTube && containsAny(reason, sourceTranscriptIssueTerms())
		unavailableIssue := containsAny(reason, sourceUnavailableIssueTerms())
		issue := candidateIssue{
			Dropped:     true,
			Transcript:  transcriptIssue,
			Unavailable: unavailableIssue,
			YouTube:     youTube,
		}
		if youTube && (transcriptIssue || unavailableIssue) {
			summary.DroppedYouTube++
		}
		if transcriptIssue {
			summary.TranscriptIssues++
		}
		if unavailableIssue {
			summary.UnavailableIssues++
		}
		for _, key := range issueKeysForURL(candidate.URL) {
			out[key] = issue
		}
	}
	return out
}

func readLocalSourceManifest(project string) []sourcepkg.StagedSource {
	data, err := os.ReadFile(projectAbsPath(project, filepath.Join("local-sources", "sources-manifest.yaml")))
	if err != nil {
		return nil
	}
	var manifest localSourceManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	return manifest.Sources
}

func evaluationCandidateReason(candidate methodologyEvaluationCandidate) string {
	return strings.ToLower(strings.Join([]string{
		candidate.URL,
		candidate.Title,
		candidate.Section,
		candidate.Rationale,
		candidate.FetchStatus,
		candidate.ContentQuality,
	}, " "))
}

func sourceHasWarningNote(src tape.Source) bool {
	return containsAny(sourceWarningText(src), append(sourceTranscriptIssueTerms(), sourceUnavailableIssueTerms()...))
}

func sourceWarningText(src tape.Source) string {
	parts := []string{src.Type, src.URL, src.Priority}
	if src.Path != nil {
		parts = append(parts, *src.Path)
	}
	if src.Citation != nil {
		parts = append(parts, *src.Citation)
	}
	if src.Note != nil {
		parts = append(parts, *src.Note)
	}
	if src.Section != nil {
		parts = append(parts, *src.Section)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

func sourceTranscriptIssueTerms() []string {
	return []string{"transcript", "caption", "yt-dlp", "rate-limit", "rate limiting", "429", "no readable"}
}

func sourceUnavailableIssueTerms() []string {
	return []string{"unavailable", "metadata_only", "metadata-only", "failed", "fetch failed", "no readable", "empty readable body", "blocked", "request blocked", "ip blocked", "rate-limit", "rate limiting", "429"}
}

func candidateIssueForKeys(issues map[string]candidateIssue, keys []string) (candidateIssue, bool) {
	for _, key := range keys {
		if issue, ok := issues[key]; ok {
			return issue, true
		}
	}
	return candidateIssue{}, false
}

func sourceKeySet(sources []tape.Source) map[string]bool {
	out := map[string]bool{}
	for _, src := range sources {
		for _, key := range issueKeysForSource(src) {
			out[key] = true
		}
	}
	return out
}

func keySetContainsAny(set map[string]bool, keys []string) bool {
	for _, key := range keys {
		if set[key] {
			return true
		}
	}
	return false
}

func issueKeysForSource(src tape.Source) []string {
	keys := issueKeysForURL(src.URL)
	if src.Path != nil {
		if key := issueKeyForPath(*src.Path); key != "" {
			keys = append(keys, key)
		}
	}
	if src.Citation != nil {
		if key := issueKeyForCitation(*src.Citation); key != "" {
			keys = append(keys, key)
		}
	}
	return dedupeStrings(keys)
}

func issueKeysForURL(value string) []string {
	normalized := normalizeIssueURL(value)
	if normalized == "" {
		return nil
	}
	keys := []string{"url:" + normalized}
	if videoID := youtubeVideoID(value); videoID != "" {
		keys = append(keys, "youtube:"+videoID)
	}
	return keys
}

func issueKeyForPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "path:" + strings.ToLower(filepath.ToSlash(filepath.Clean(value)))
}

func issueKeyForCitation(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "citation:" + strings.ToLower(value)
}

func normalizeIssueURL(value string) string {
	value = normalizeMethodologyURL(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.ToLower(value)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func isYouTubeSource(src tape.Source) bool {
	if strings.EqualFold(strings.TrimSpace(src.Type), "youtube") {
		return true
	}
	return isYouTubeURL(src.URL)
}

func isYouTubeURL(value string) bool {
	if youtubeVideoID(value) != "" {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "youtube.com/") || strings.Contains(lower, "youtu.be/")
}

func youtubeVideoID(value string) string {
	value = strings.TrimSpace(value)
	if youtubeVideoIDPattern.MatchString(value) {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	host := strings.ToLower(parsed.Host)
	if strings.HasSuffix(host, "youtu.be") {
		candidate := strings.Trim(strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")[0], " ")
		if youtubeVideoIDPattern.MatchString(candidate) {
			return candidate
		}
		return ""
	}
	if !strings.Contains(host, "youtube.com") {
		return ""
	}
	if parsed.Path == "/watch" {
		candidate := parsed.Query().Get("v")
		if youtubeVideoIDPattern.MatchString(candidate) {
			return candidate
		}
	}
	for _, prefix := range []string{"/embed/", "/shorts/", "/v/", "/live/"} {
		if strings.HasPrefix(parsed.Path, prefix) {
			candidate := strings.Split(strings.TrimPrefix(parsed.Path, prefix), "/")[0]
			if youtubeVideoIDPattern.MatchString(candidate) {
				return candidate
			}
		}
	}
	return ""
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
