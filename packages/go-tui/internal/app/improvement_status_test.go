package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cmdux/liner/packages/go-tui/internal/tape"
)

const staleQualityChecks = `## Test 6 — Note-quality

Checked: 20 kept/trim notes

## Test 7 — Source-role fit

Required roles:
- [worked workflow cases] — minimum: 2; current: 0; status: gap; evidence: none
- [social-media verification method] — minimum: 1; current: 0; status: gap; evidence: none
- [official oversight guidance] — minimum: 2; current: 1; status: weak; evidence: NIST AI RMF
- [primary research] — minimum: 3; current: 6; status: pass; evidence: several

Operating-fit audit path: status: improvement_recommended
`

func writeImprovementAudits(t *testing.T, project string, files map[string]string, modified map[string]time.Time) {
	t.Helper()
	for rel, body := range files {
		path := projectAbsPath(project, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if at, ok := modified[rel]; ok {
			if err := os.Chtimes(path, at, at); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestImprovementOfferUsesLatestStatusAfterAppliedPass(t *testing.T) {
	project := t.TempDir()
	before := time.Date(2026, 10, 18, 18, 43, 0, 0, time.UTC)
	after := before.Add(40 * time.Minute)
	applied := resolveImprovementAuditBody("# Operating-fit audit\n\nstatus: improvement_recommended\ngap: No complete worked workflow cases.\n")
	writeImprovementAudits(t, project, map[string]string{
		qualityChecksRelPath:     staleQualityChecks,
		operatingFitAuditRelPath: applied,
	}, map[string]time.Time{qualityChecksRelPath: before, operatingFitAuditRelPath: after})

	if _, ok := operatingFitImprovementAudit(project); ok {
		t.Fatal("an applied pass recorded after the quality checks should end the offer")
	}
	if operatingFitImprovementRecommended(project) {
		t.Fatal("Liner should not offer the improvement pass again after it was applied")
	}

	// A newer recommendation (a later quality-check run) offers again.
	writeImprovementAudits(t, project, map[string]string{qualityChecksRelPath: staleQualityChecks},
		map[string]time.Time{qualityChecksRelPath: after.Add(time.Hour)})
	audit, ok := operatingFitImprovementAudit(project)
	if !ok || audit.RelPath != qualityChecksRelPath {
		t.Fatalf("the latest recommendation should be offered, got ok=%v audit=%#v", ok, audit)
	}
}

func TestImprovementOfferShowsGapsInPlainWords(t *testing.T) {
	project := t.TempDir()
	writeImprovementAudits(t, project, map[string]string{qualityChecksRelPath: staleQualityChecks}, nil)

	audit, ok := operatingFitImprovementAudit(project)
	if !ok {
		t.Fatal("quality checks with a recommendation should offer the pass")
	}
	want := "worked workflow cases: none of 2 needed; social-media verification method: none of 1 needed; official oversight guidance: 1 of 2 needed."
	if audit.Summary != want {
		t.Fatalf("gap should be in plain words:\n got %q\nwant %q", audit.Summary, want)
	}

	m := Model{screen: screenImprovementReview, width: 100, height: 30, currentPath: project, currentTape: tape.Tape{Title: "Synthetic"}}
	view := strings.Join(strings.Fields(stripANSICodesForTest(m.viewImprovementReview())), " ")
	if strings.Contains(view, "##") || strings.Contains(view, "Test 7") {
		t.Fatalf("the offer should not show a raw heading:\n%s", view)
	}
	if !strings.Contains(view, "Gap: worked workflow cases: none of 2 needed") {
		t.Fatalf("the offer should show the gaps:\n%s", view)
	}
}

func TestImprovementOfferPrefersOperatingFitGapWhenBothRecommend(t *testing.T) {
	project := t.TempDir()
	at := time.Date(2026, 10, 18, 18, 43, 0, 0, time.UTC)
	writeImprovementAudits(t, project, map[string]string{
		qualityChecksRelPath:     staleQualityChecks,
		operatingFitAuditRelPath: "# Operating-fit audit\n\nstatus: improvement_recommended\ngap: No complete worked workflow cases.\n",
	}, map[string]time.Time{qualityChecksRelPath: at, operatingFitAuditRelPath: at})

	audit, ok := operatingFitImprovementAudit(project)
	if !ok || audit.RelPath != operatingFitAuditRelPath || audit.Summary != "No complete worked workflow cases." {
		t.Fatalf("a tie should go to the operating-fit audit and its gap, got ok=%v %#v", ok, audit)
	}
}
