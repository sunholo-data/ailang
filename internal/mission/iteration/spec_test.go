package iteration

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

func validSpec() Spec {
	limits := Limits{30, 1000, 0.1}
	stages := []Stage{}
	for _, r := range []string{"designer", "planner", "executor", "evaluator"} {
		stages = append(stages, Stage{ID: r, Role: r, Instructions: "Do approved work", RequiredArtifacts: []string{"docs/result.md"}, Limits: limits})
	}
	return Spec{Version: 1, MissionID: "docs", WorkItemID: "item-1", Repository: "github.com/example/project", BaseRevision: strings.Repeat("a", 40), Brief: "Improve docs", AllowedPaths: []string{"docs/"}, Workflow: "full-v1", Stages: stages, Limits: Limits{120, 4000, 0.4}, AcceptanceCriteria: []Criterion{{ID: "clear", Text: "Clear documentation"}}, Verification: []Verification{{ID: "check", Argv: []string{"git", "diff", "--check"}, Cwd: ".", TimeoutSeconds: 30}}}
}
func TestSpecStrictDecode(t *testing.T) {
	s := validSpec()
	b, _ := json.Marshal(s)
	got, err := Decode(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest() != s.Digest() {
		t.Fatal("digest changed")
	}
	for name, input := range map[string]string{"unknown": strings.Replace(string(b), `"version":1`, `"version":1,"extra":1`, 1), "duplicate": strings.Replace(string(b), `"version":1`, `"version":1,"version":1`, 1), "case_alias": strings.Replace(string(b), `"version":1`, `"Version":1`, 1), "null": "null", "trailing": string(b) + " {}", "oversize": strings.Repeat(" ", MaxSpecBytes+1), "nested_duplicate": strings.Replace(string(b), `"max_tokens":1000`, `"max_tokens":1000,"max_tokens":1000`, 1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(strings.NewReader(input)); err == nil {
				t.Fatal("accepted invalid JSON")
			}
		})
	}
}
func TestSpecValidation(t *testing.T) {
	cases := map[string]func(*Spec){
		"version": func(s *Spec) { s.Version = 2 }, "id": func(s *Spec) { s.WorkItemID = "../escape" }, "base": func(s *Spec) { s.BaseRevision = "HEAD" }, "brief": func(s *Spec) { s.Brief = " " }, "workflow": func(s *Spec) { s.Workflow = "fast" }, "missing_role": func(s *Spec) { s.Stages = s.Stages[1:] }, "order": func(s *Spec) { s.Stages[0], s.Stages[1] = s.Stages[1], s.Stages[0] }, "duplicate_id": func(s *Spec) { s.Stages[1].ID = s.Stages[0].ID }, "path": func(s *Spec) { s.AllowedPaths = []string{"docs/../secret"} }, "glob": func(s *Spec) { s.AllowedPaths = []string{"docs/*"} }, "absolute": func(s *Spec) { s.AllowedPaths = []string{"C:/docs"} }, "backslash": func(s *Spec) { s.AllowedPaths = []string{`docs\file`} }, "reserved": func(s *Spec) { s.Stages[0].RequiredArtifacts = []string{"stage-result.json"} }, "scope": func(s *Spec) { s.Stages[0].RequiredArtifacts = []string{"src/code.go"} }, "limits": func(s *Spec) { s.Limits.TimeoutSeconds = 7201 }, "stage_limits": func(s *Spec) { s.Stages[0].Limits.TimeoutSeconds = 1801 }, "nan": func(s *Spec) { s.Limits.MaxCostUSD = math.NaN() }, "check_timeout": func(s *Spec) { s.Verification[0].TimeoutSeconds = 601 }, "check_cwd": func(s *Spec) { s.Verification[0].Cwd = "../" }, "check_argv": func(s *Spec) { s.Verification[0].Argv = nil }, "criteria": func(s *Spec) { s.AcceptanceCriteria = nil }, "duplicate_criterion": func(s *Spec) { s.AcceptanceCriteria = append(s.AcceptanceCriteria, s.AcceptanceCriteria[0]) }, "too_many_paths": func(s *Spec) { s.AllowedPaths = make([]string, 257) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validSpec()
			mutate(&s)
			if s.Validate() == nil {
				t.Fatal("invalid spec accepted")
			}
		})
	}
}
func TestPrerequisitesAndDigest(t *testing.T) {
	s := validSpec()
	sha := strings.Repeat("a", 64)
	for _, role := range []string{"designer", "planner"} {
		s.Prerequisites = append(s.Prerequisites, Prerequisite{Role: role, Artifact: ArtifactRef{Commit: s.BaseRevision, Path: "docs/plan.md", SHA256: sha}, AuthorityRefs: []AuthorityRef{{Revision: s.BaseRevision, Path: "docs/approval.md", Locator: "decision-1", SHA256: sha, ArtifactDigest: sha}}, AuthorModels: []string{"author-model"}})
	}
	s.Stages = s.Stages[2:]
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	digest := s.Digest()
	s.AcceptanceCriteria[0].Text = "Changed"
	if digest == s.Digest() {
		t.Fatal("criterion not bound")
	}
	s.Prerequisites[0].AuthorityRefs = nil
	if s.Validate() == nil {
		t.Fatal("missing authority accepted")
	}
}
func validResult() StageResult {
	return StageResult{Version: 1, RequestDigest: strings.Repeat("b", 64), InputRevision: strings.Repeat("a", 40), OutputRevision: strings.Repeat("a", 40), ArtifactPaths: []string{"docs/result.md"}, Outcome: "pass", Criteria: map[string]CriterionResult{"clear": {Outcome: "pass", Evidence: "docs/result.md explains setup"}}}
}
func TestStageResultContract(t *testing.T) {
	criteria := validSpec().AcceptanceCriteria
	r := validResult()
	b, _ := json.Marshal(r)
	if _, err := DecodeStageResult(strings.NewReader(string(b)), "evaluator", criteria); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*StageResult){"missing": func(r *StageResult) { r.Criteria = nil }, "extra": func(r *StageResult) { r.Criteria["other"] = CriterionResult{Outcome: "pass", Evidence: "yes"} }, "empty_evidence": func(r *StageResult) { r.Criteria["clear"] = CriterionResult{Outcome: "pass"} }, "blocked_pass": func(r *StageResult) { r.BlockingFindings = []string{"bug"} }, "failed_pass": func(r *StageResult) { r.Criteria["clear"] = CriterionResult{Outcome: "fail", Evidence: "bug"} }, "revision": func(r *StageResult) { r.OutputRevision = "HEAD" }, "author_outcome": func(r *StageResult) { r.Outcome = "produced" }} {
		t.Run(name, func(t *testing.T) {
			r := validResult()
			mutate(&r)
			if r.Validate("evaluator", criteria) == nil {
				t.Fatal("invalid result accepted")
			}
		})
	}
	r = validResult()
	r.Outcome = "produced"
	r.Criteria = nil
	r.OutputRevision = strings.Repeat("c", 40)
	if err := r.Validate("executor", criteria); err != nil {
		t.Fatal(err)
	}
	r.OutputRevision = r.InputRevision
	if r.Validate("executor", criteria) == nil {
		t.Fatal("no-change author accepted")
	}
	if _, err := DecodeStageResult(strings.NewReader(strings.Repeat(" ", MaxResultBytes+1)), "evaluator", criteria); err == nil {
		t.Fatal("oversize accepted")
	}
}

func TestJSONFixtures(t *testing.T) {
	for _, name := range []string{"work-item", "malformed"} {
		body, err := os.ReadFile("../../../cmd/ailang/testdata/mission-iteration/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		_, err = Decode(bytes.NewReader(body))
		if (err != nil) != (name == "malformed") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
func TestPathsBoundaries(t *testing.T) {
	for _, p := range []string{"docs", "docs-other/a", "docs/../secret", "/docs/a", "stage-result.json", "docs//a", "docs/a/"} {
		if PathAllowed(p, []string{"docs/"}) {
			t.Fatalf("unexpected scope acceptance %q", p)
		}
	}
	if !PathAllowed("docs/a", []string{"docs/"}) || !PathAllowed("README.md", []string{"README.md"}) {
		t.Fatal("valid path rejected")
	}
}
func TestResultStrictDecode(t *testing.T) {
	b, _ := json.Marshal(validResult())
	base := string(b)
	for _, input := range []string{base + " {}", strings.Replace(base, `"outcome":"pass"`, `"outcome":"pass","outcome":"fail"`, 1), strings.Replace(base, `"evidence":`, `"Evidence":`, 1), strings.Replace(base, `"version":1`, `"version":null`, 1)} {
		if _, err := DecodeStageResult(strings.NewReader(input), "evaluator", validSpec().AcceptanceCriteria); err == nil {
			t.Fatal("ambiguous result accepted")
		}
	}
}

func TestRequiredJSONFields(t *testing.T) {
	b, _ := json.Marshal(validSpec())
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "verification")
	b, _ = json.Marshal(fields)
	if _, err := Decode(bytes.NewReader(b)); err == nil {
		t.Fatal("missing required verification field accepted")
	}
}
