package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type evaluationIssueSummary struct {
	DroppedCandidates int
	DroppedYouTube    int
	TranscriptIssues  int
	UnavailableIssues int
}

func readEvaluationIssueSummary(project string) (evaluationIssueSummary, bool) {
	if strings.TrimSpace(project) == "" {
		return evaluationIssueSummary{}, false
	}
	data, err := os.ReadFile(projectAbsPath(project, filepath.Join("working", "03-evaluation.yaml")))
	if err != nil {
		return evaluationIssueSummary{}, false
	}
	var artifact methodologyEvaluationArtifact
	if err := yaml.Unmarshal(data, &artifact); err != nil || len(artifact.Candidates) == 0 {
		return evaluationIssueSummary{}, false
	}
	summary := evaluationIssueSummary{}
	for _, candidate := range artifact.Candidates {
		if strings.ToLower(strings.TrimSpace(candidate.Decision)) != "dropped" {
			continue
		}
		summary.DroppedCandidates++
		reason := evaluationCandidateReason(candidate)
		transcriptIssue := isYouTubeCandidate(candidate.URL) && containsAny(reason, []string{"transcript", "caption", "youtube", "yt-dlp", "rate-limit", "rate limiting", "429", "blocked", "no readable"})
		unavailableIssue := containsAny(reason, []string{"unavailable", "metadata_only", "metadata-only", "failed", "no readable", "empty readable body", "blocked", "rate-limit", "429"})
		if isYouTubeCandidate(candidate.URL) && (transcriptIssue || unavailableIssue) {
			summary.DroppedYouTube++
		}
		if transcriptIssue {
			summary.TranscriptIssues++
		}
		if unavailableIssue {
			summary.UnavailableIssues++
		}
	}
	return summary, summary.HasIssues()
}

func (s evaluationIssueSummary) HasIssues() bool {
	return s.DroppedYouTube > 0 || s.TranscriptIssues > 0 || s.UnavailableIssues > 0
}

func (s evaluationIssueSummary) Display(project string) string {
	if !s.HasIssues() {
		return "none"
	}
	parts := []string{}
	if s.DroppedYouTube > 0 {
		if s.TranscriptIssues > 0 {
			parts = append(parts, "transcript/access: "+intLabel(s.DroppedYouTube, "YouTube candidate")+" dropped")
		} else {
			parts = append(parts, intLabel(s.DroppedYouTube, "YouTube candidate")+" dropped")
		}
	} else if s.UnavailableIssues > 0 {
		parts = append(parts, intLabel(s.UnavailableIssues, "unavailable candidate"))
	}
	rel := displayProjectPath(project, filepath.Join("working", "03-evaluation.yaml"))
	return fmt.Sprintf("%s; see %s", strings.Join(parts, "; "), rel)
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

func isYouTubeCandidate(url string) bool {
	lower := strings.ToLower(strings.TrimSpace(url))
	return strings.Contains(lower, "youtube.com/") || strings.Contains(lower, "youtu.be/")
}
