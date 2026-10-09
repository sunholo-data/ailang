package coordinator

import (
	"strings"
	"testing"
)

// The evaluator's directive must show EVALUATION_VERDICT in the form
// ParseEvaluationVerdict reads. It used to show every marker as
// "<path-to-file>" with a bold example, and the cloud evaluator copied it
// (2026-10-09: "EVALUATION_VERDICT: <path>"), which parses as no verdict.
func TestSkillDirective_VerdictMarkerShowsAParseableVerdict(t *testing.T) {
	agent := &AgentConfig{ID: "sprint-evaluator", OutputMarkers: []string{EvaluationVerdictMarker}}
	d := buildSkillDirectiveWithConfig(&TaskRecord{Content: "Evaluate the sprint."}, agent, &InvokeConfig{Type: "skill", Name: "sprint-evaluator"})

	if strings.Contains(d, EvaluationVerdictMarker+" <path-to-file>") {
		t.Errorf("directive still asks for a file path after %s:\n%s", EvaluationVerdictMarker, d)
	}
	if strings.Contains(d, "**"+EvaluationVerdictMarker+"**") {
		t.Errorf("directive shows the verdict marker in bold; the parser does not strip markdown:\n%s", d)
	}

	// Every verdict line the directive shows must parse as a real verdict.
	var shown int
	for _, line := range strings.Split(d, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, EvaluationVerdictMarker) || strings.Contains(line, "<") {
			continue // the template lines carry placeholders; the example must parse
		}
		shown++
		if v := ExtractEvaluationVerdict(line); v.Kind != VerdictPass && v.Kind != VerdictFail {
			t.Errorf("example %q parses as %v, not a PASS/FAIL verdict", line, v.Kind)
		}
	}
	if shown == 0 {
		t.Errorf("directive shows no concrete %s example line:\n%s", EvaluationVerdictMarker, d)
	}
}

// Other markers keep the file-path form.
func TestSkillDirective_PathMarkerKeepsPathForm(t *testing.T) {
	agent := &AgentConfig{ID: "design-doc-creator", OutputMarkers: []string{"DESIGN_DOC_CREATED:"}}
	d := buildSkillDirectiveWithConfig(&TaskRecord{Content: "Write a doc."}, agent, &InvokeConfig{Type: "skill", Name: "design-doc-creator"})
	if !strings.Contains(d, "DESIGN_DOC_CREATED: <path-to-file>") {
		t.Errorf("a path marker lost its <path-to-file> form:\n%s", d)
	}
}
