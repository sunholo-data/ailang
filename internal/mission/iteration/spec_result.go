package iteration

import (
	"fmt"
	"io"
	"strings"
)

type CriterionResult struct {
	Outcome  string `json:"outcome"`
	Evidence string `json:"evidence"`
}
type StageResult struct {
	Version          int                        `json:"version"`
	RequestDigest    string                     `json:"request_digest"`
	InputRevision    string                     `json:"input_revision"`
	OutputRevision   string                     `json:"output_revision"`
	ArtifactPaths    []string                   `json:"artifact_paths"`
	Outcome          string                     `json:"outcome"`
	Criteria         map[string]CriterionResult `json:"criteria"`
	BlockingFindings []string                   `json:"blocking_findings"`
}

func DecodeStageResult(r io.Reader, role string, criteria []Criterion) (StageResult, error) {
	var result StageResult
	if err := decodeStrict(r, MaxResultBytes, &result); err != nil {
		return result, err
	}
	return result, result.Validate(role, criteria)
}

// Validate checks protocol semantics. Runtime additionally binds digests, inspects
// commits, verifies artifacts and records provenance before accepting a result.
func (r StageResult) Validate(role string, criteria []Criterion) error {
	if r.Version != 1 || !validHash(r.RequestDigest) || !validRevision(r.InputRevision) || !validRevision(r.OutputRevision) {
		return fmt.Errorf("result requires version 1, SHA-256 request digest and full revisions")
	}
	if err := validatePaths(r.ArtifactPaths, false, false); err != nil {
		return err
	}
	for _, finding := range r.BlockingFindings {
		if strings.TrimSpace(finding) == "" {
			return fmt.Errorf("empty blocking finding")
		}
	}
	switch role {
	case "designer", "planner", "executor":
		if r.Outcome != "produced" && r.Outcome != "fail" && r.Outcome != "needs_decision" {
			return fmt.Errorf("author outcome must be produced, fail or needs_decision")
		}
		if len(r.Criteria) != 0 {
			return fmt.Errorf("only evaluator may report criteria")
		}
		if r.Outcome == "produced" && (len(r.ArtifactPaths) == 0 || r.InputRevision == r.OutputRevision || len(r.BlockingFindings) != 0) {
			return fmt.Errorf("produced author result requires changed artifacts and no blockers")
		}
	case "evaluator":
		if r.Outcome != "pass" && r.Outcome != "fail" && r.Outcome != "needs_decision" {
			return fmt.Errorf("evaluator outcome must be pass, fail or needs_decision")
		}
		if r.InputRevision != r.OutputRevision {
			return fmt.Errorf("evaluator must preserve candidate revision")
		}
		if len(criteria) == 0 || len(r.Criteria) != len(criteria) {
			return fmt.Errorf("evaluator must report every frozen criterion exactly once")
		}
		seen := map[string]bool{}
		for _, criterion := range criteria {
			if !validID(criterion.ID) || seen[criterion.ID] {
				return fmt.Errorf("invalid frozen criterion ID")
			}
			seen[criterion.ID] = true
			c, ok := r.Criteria[criterion.ID]
			if !ok || (c.Outcome != "pass" && c.Outcome != "fail") || strings.TrimSpace(c.Evidence) == "" {
				return fmt.Errorf("criterion %q requires pass/fail and evidence", criterion.ID)
			}
			if r.Outcome == "pass" && c.Outcome != "pass" {
				return fmt.Errorf("passing evaluation has failing criterion")
			}
		}
		if r.Outcome == "pass" && len(r.BlockingFindings) > 0 {
			return fmt.Errorf("passing evaluation has blocking findings")
		}
	default:
		return fmt.Errorf("unknown stage role %q", role)
	}
	return nil
}

// Digest binds result fields; call Validate with the frozen criteria first.
func (r StageResult) Digest() string { return digest(r) }
