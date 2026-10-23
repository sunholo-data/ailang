package eval_harness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// MaintenanceNativeGradeOptions is controller configuration, never model input.
// A digest is a binding, not proof: protected grading also requires admission of
// the actual wrapper/policy/runtime boundary. Development mode is no-model only.
type MaintenanceNativeGradeOptions struct {
	EvidenceRoot          string
	CommandPrefix         []string
	ContainmentDigest     string
	DevelopmentUnconfined bool
	Admission             func(context.Context) error
}

type maintenanceGradeContainment struct {
	Mode                string  `json:"mode"`
	Qualified           bool    `json:"qualified"`
	QualificationDigest *string `json:"qualificationDigest"`
	WrapperDigest       string  `json:"wrapperDigest"`
	QualificationStatus string  `json:"qualificationStatus"`
}
type maintenanceGradeCase struct {
	ID                 string   `json:"id"`
	InputDigest        string   `json:"inputDigest"`
	ExpectedDigest     string   `json:"expectedDigest"`
	ExpectedAssertions int      `json:"expectedAssertions"`
	OutputDigest       string   `json:"outputDigest"`
	MismatchIDs        []string `json:"mismatchIds"`
	Assertions         *int     `json:"assertions"`
	Passed             bool     `json:"passed"`
	Failure            string   `json:"failure"`
}
type maintenanceGradeProtocol struct {
	Schema               string                      `json:"schema"`
	Family               string                      `json:"family"`
	Language             string                      `json:"language"`
	Stage                int                         `json:"stage"`
	Status               string                      `json:"status"`
	Compiled             bool                        `json:"compiled"`
	Cases                []maintenanceGradeCase      `json:"cases"`
	PublicCheckPassed    *bool                       `json:"publicCheckPassed"`
	SourceDigest         string                      `json:"sourceDigest"`
	CandidateDigestAfter *string                     `json:"candidateDigestAfter"`
	Containment          maintenanceGradeContainment `json:"containment"`
	ModelRun             bool                        `json:"modelRun"`
	TrustedInputs        map[string]string           `json:"trustedInputs"`
	OracleDigest         string                      `json:"oracleDigest"`
	ProtocolDigest       string                      `json:"protocolDigest"`
	RuntimeDigest        string                      `json:"runtimeDigest"`
	RuntimePath          string                      `json:"runtimePath"`
	Failure              string                      `json:"failure"`
	ProcessExit          *int                        `json:"processExit"`
	DiagnosticDigest     string                      `json:"diagnosticDigest"`
}

func NewMaintenanceNativeGrader(e *MaintenanceExecution, o MaintenanceNativeGradeOptions) (MaintenanceGrader, error) {
	if e == nil || o.EvidenceRoot == "" {
		return nil, fmt.Errorf("explicit execution and controller grade root required")
	}
	if o.DevelopmentUnconfined {
		if o.Admission != nil || len(o.CommandPrefix) > 0 || o.ContainmentDigest != "" {
			return nil, fmt.Errorf("development grading conflicts with protected configuration")
		}
		if e.Plan.Role != "instrument_control" {
			return nil, fmt.Errorf("unconfined grading only admits instrument controls")
		}
	} else if o.Admission == nil || len(o.CommandPrefix) == 0 || !maintenanceSHA(o.ContainmentDigest) {
		return nil, fmt.Errorf("protected grading requires actual admission, trusted prefix and policy digest")
	}
	return func(ctx context.Context, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
		return maintenanceInvokeGrade(ctx, e, o, tr, stage, cp)
	}, nil
}

func maintenanceInvokeGrade(ctx context.Context, e *MaintenanceExecution, o MaintenanceNativeGradeOptions, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
	empty := MaintenanceGrade{}
	if o.Admission != nil {
		if err := o.Admission(ctx); err != nil {
			return empty, fmt.Errorf("candidate containment refused: %w", err)
		}
	}
	binding, err := maintenanceGradeBinding(e, tr, stage)
	if err != nil {
		return empty, err
	}
	if err = maintenanceVerifyGradeBinding(binding); err != nil {
		return empty, err
	}
	if err = os.MkdirAll(o.EvidenceRoot, 0700); err != nil {
		return empty, err
	}
	dir, err := os.MkdirTemp(o.EvidenceRoot, "candidate-grade-")
	if err != nil {
		return empty, err
	}
	candidate := filepath.Join(dir, "retained")
	if err = RestoreMaintenanceCheckpoint(cp, candidate); err != nil {
		return empty, err
	}
	output := filepath.Join(dir, "grade.json")
	argv, err := maintenanceGradeArgv(binding.Oracle, candidate, output, cp.Digest, o)
	if err != nil {
		return empty, err
	}
	seconds := e.Manifest.Limits.MaxStageSeconds
	if seconds <= 0 {
		return empty, fmt.Errorf("grading time bound unavailable")
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(seconds)*time.Second)
	defer cancel()
	command := exec.CommandContext(bounded, argv[0], argv[1:]...)
	command.Dir = e.Root
	proctree.Configure(command)
	raw, commandErr := command.CombinedOutput()
	proctree.Kill(command)
	evidence := struct {
		Argv   []string `json:"argv"`
		Output string   `json:"output"`
		Error  string   `json:"error"`
	}{Argv: argv, Output: string(raw)}
	if commandErr != nil {
		evidence.Error = commandErr.Error()
	}
	if err = maintenanceWriteNewJSON(filepath.Join(dir, "invocation.json"), evidence); err != nil {
		return empty, err
	}
	if err = maintenanceVerifyGradeBinding(binding); err != nil {
		return empty, err
	}
	bytes, err := os.ReadFile(output)
	if err != nil {
		return empty, fmt.Errorf("candidate grade did not emit evidence: %w: %v", err, commandErr)
	}
	var report maintenanceGradeProtocol
	if err = maintenanceGradeRequired(bytes); err != nil {
		return empty, err
	}
	if err = maintenanceStrictJSON(bytes, &report); err != nil {
		return empty, fmt.Errorf("invalid typed candidate report: %w", err)
	}
	if commandErr != nil && report.Status != "invalid_setup" {
		return empty, fmt.Errorf("candidate controller process failed: %w", commandErr)
	}
	if err = maintenanceGradeReceipts(report, binding, o); err != nil {
		return empty, err
	}
	return maintenanceGradeVerdict(report, tr, stage, cp)
}

func maintenanceGradeBinding(e *MaintenanceExecution, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan) (MaintenanceStageBinding, error) {
	for _, binding := range e.Manifest.Trajectories {
		if binding.ID != tr.ID {
			continue
		}
		for _, s := range binding.Stages {
			if s.ID == stage.ID {
				if len(s.Oracle.Files) == 0 || s.Oracle.Files[0].SHA256 != stage.OracleDigest {
					return s, fmt.Errorf("oracle identity is not frozen stage binding")
				}
				return s, nil
			}
		}
	}
	return MaintenanceStageBinding{}, fmt.Errorf("missing trajectory/stage grading binding")
}

func maintenanceVerifyGradeBinding(binding MaintenanceStageBinding) error {
	for _, command := range []MaintenanceCommandBinding{binding.PublicCheck, binding.Oracle} {
		if len(command.Command) < 2 || len(command.Files) == 0 {
			return fmt.Errorf("missing native grading command or source")
		}
		for _, ref := range command.Files {
			if _, err := maintenanceReadBoundFile(ref); err != nil {
				return err
			}
		}
	}
	return nil
}

func maintenanceGradeArgv(binding MaintenanceCommandBinding, candidate, output, hash string, o MaintenanceNativeGradeOptions) ([]string, error) {
	prefix, err := json.Marshal(o.CommandPrefix)
	if err != nil {
		return nil, err
	}
	substitutions := map[string]string{"{candidate}": candidate, "{output}": output, "{candidateDigest}": hash, "{commandPrefixJSON}": string(prefix), "{containmentDigest}": o.ContainmentDigest}
	argv := []string{}
	for i, part := range binding.Command {
		if o.DevelopmentUnconfined && (part == "--command-prefix-json" || part == "--containment-qualified" || i > 0 && (binding.Command[i-1] == "--command-prefix-json" || binding.Command[i-1] == "--containment-qualified")) {
			continue
		}
		if replacement, ok := substitutions[part]; ok {
			part = replacement
		} else if strings.ContainsAny(part, "{}") {
			return nil, fmt.Errorf("unknown grade command substitution: %s", part)
		}
		argv = append(argv, part)
	}
	if len(argv) < 2 || len(binding.Files) == 0 {
		return nil, fmt.Errorf("native grader command missing")
	}
	// The manifest's first bound controller file is authoritative, independent of
	// a relative argv spelling in a movable example manifest.
	argv[1] = binding.Files[0].Path
	if o.DevelopmentUnconfined {
		argv = append(argv, "--development-unconfined")
	}
	for flag, value := range map[string]string{"--candidate": candidate, "--output": output, "--expected-source-digest": hash} {
		found := false
		for i := 0; i+1 < len(argv); i++ {
			if argv[i] == flag && argv[i+1] == value {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("grade command lacks bound %s", flag)
		}
	}
	return argv, nil
}

func maintenanceGradeReceipts(r maintenanceGradeProtocol, b MaintenanceStageBinding, o MaintenanceNativeGradeOptions) error {
	if r.ModelRun || r.Containment.Qualified {
		return fmt.Errorf("grade report claims unsupported model/containment attestation")
	}
	want := "controller_wrapper"
	if o.DevelopmentUnconfined {
		want = "development_unconfined"
	}
	if r.Containment.Mode != want {
		return fmt.Errorf("grade containment mode drift")
	}
	if !o.DevelopmentUnconfined && (r.Containment.QualificationDigest == nil || *r.Containment.QualificationDigest != o.ContainmentDigest) {
		return fmt.Errorf("grade policy identity drift")
	}
	bound := map[string]string{}
	for _, ref := range b.Oracle.Files {
		bound[filepath.Clean(ref.Path)] = ref.SHA256
	}
	for path, hash := range r.TrustedInputs {
		if bound[filepath.Clean(path)] != hash {
			return fmt.Errorf("unbound trusted grading source: %s", path)
		}
	}
	if r.Status != "invalid_setup" {
		if len(r.TrustedInputs) == 0 || r.TrustedInputs[filepath.Clean(b.Oracle.Files[0].Path)] != b.Oracle.Files[0].SHA256 {
			return fmt.Errorf("grade report omitted trusted source receipts")
		}
		values := map[string]bool{}
		for _, hash := range r.TrustedInputs {
			values[hash] = true
		}
		if !values[r.OracleDigest] || !values[r.ProtocolDigest] {
			return fmt.Errorf("unbound oracle/protocol receipt")
		}
		runtimeHash, err := maintenanceFileSHA(r.RuntimePath)
		if err != nil || runtimeHash != r.RuntimeDigest {
			return fmt.Errorf("native runtime receipt mismatch")
		}
		for _, row := range r.Cases {
			if !values[row.InputDigest] || row.ExpectedDigest != "" && !values[row.ExpectedDigest] {
				return fmt.Errorf("unbound native fixture receipt")
			}
		}
	}
	return nil
}

func maintenanceGradeVerdict(r maintenanceGradeProtocol, tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, cp MaintenanceCheckpoint) (MaintenanceGrade, error) {
	g := MaintenanceGrade{ArtifactDigest: cp.Digest, Provenance: "native controller candidate report; private oracle withheld from model"}
	number, err := strconv.Atoi(strings.TrimPrefix(stage.ID, "S"))
	if err != nil || number < 1 || number > 3 || r.Schema != "maintenance.candidate-grade/1" || r.Family != tr.Family || r.Language != tr.Language || r.Stage != number || r.SourceDigest != cp.Digest {
		return g, fmt.Errorf("candidate report identity mismatch")
	}
	if r.Status == "invalid_setup" {
		return g, fmt.Errorf("native grading setup invalid: %s", r.Failure)
	}
	kind := "parser"
	if tr.Language == "ailang" {
		kind = "type"
	}
	g.CheckKinds = []string{kind}
	if r.Status == "artifact_failure" {
		g.Visible = "fail"
		g.Oracle = "not_run"
		g.Provenance += "; model artifact/check failure: " + r.Failure
		return g, nil
	}
	if (r.Status != "passed" && r.Status != "semantic_failure") || !r.Compiled || r.CandidateDigestAfter == nil || *r.CandidateDigestAfter != cp.Digest || !maintenanceSHA(r.RuntimeDigest) || !maintenanceSHA(r.OracleDigest) || !maintenanceSHA(r.ProtocolDigest) {
		return g, fmt.Errorf("invalid native semantic evidence")
	}
	count := 1
	if tr.Family == "resource-ownership" {
		count = 2
	} else if tr.Family != "text-fidelity" {
		return g, fmt.Errorf("unknown native task family")
	}
	if len(r.Cases) != count {
		return g, fmt.Errorf("missing native oracle case")
	}
	if tr.Family == "resource-ownership" && r.PublicCheckPassed == nil {
		return g, fmt.Errorf("resource public check was skipped")
	}
	passed := true
	seen := map[string]bool{}
	for _, c := range r.Cases {
		expectedAssertions := map[int]int{1: 8, 2: 15, 3: 21}[number]
		validID := c.ID == fmt.Sprintf("s%d", number)
		if tr.Family == "resource-ownership" {
			expectedAssertions = map[int]int{1: 87, 2: 123, 3: 152}[number]
			validID = c.ID == fmt.Sprintf("s%d-normal", number) || c.ID == fmt.Sprintf("s%d-reordered", number)
		}
		if !validID || c.ExpectedAssertions != expectedAssertions {
			return g, fmt.Errorf("oracle fixture/assertion inventory drift")
		}
		if c.ID == "" || seen[c.ID] || !maintenanceSHA(c.InputDigest) || !maintenanceSHA(c.OutputDigest) || c.ExpectedAssertions <= 0 {
			return g, fmt.Errorf("invalid or duplicate native oracle case")
		}
		seen[c.ID] = true
		if c.Passed && (c.Assertions == nil || *c.Assertions != c.ExpectedAssertions || len(c.MismatchIDs) > 0) {
			return g, fmt.Errorf("oracle skipped or contradicted assertions")
		}
		if !c.Passed && len(c.MismatchIDs) == 0 {
			return g, fmt.Errorf("semantic failure lacks failed obligation")
		}
		passed = passed && c.Passed
	}
	if (r.Status == "passed") != (passed && (r.PublicCheckPassed == nil || *r.PublicCheckPassed)) {
		return g, fmt.Errorf("native status contradicts checks")
	}
	g.Visible = "pass"
	if r.PublicCheckPassed != nil {
		g.CheckKinds = append(g.CheckKinds, "public")
		if !*r.PublicCheckPassed {
			g.Visible = "fail"
		}
	}
	g.Oracle = "fail"
	if passed {
		g.Oracle = "pass"
	}
	g.OracleDigest = stage.OracleDigest
	g.CheckKinds = append(g.CheckKinds, "oracle")
	return g, nil
}

// Presence matters for booleans and nulls: zero-value Go fields are not receipt
// evidence that a controller reported an explicit negative/empty quantity.
func maintenanceGradeRequired(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	for _, name := range []string{"schema", "family", "language", "stage", "status", "compiled", "cases", "publicCheckPassed", "sourceDigest", "candidateDigestAfter", "containment", "modelRun"} {
		value, ok := fields[name]
		if !ok {
			return fmt.Errorf("candidate report omitted %s", name)
		}
		if string(value) == "null" && name != "publicCheckPassed" && name != "sourceDigest" && name != "candidateDigestAfter" {
			return fmt.Errorf("candidate report null %s", name)
		}
	}
	var containment map[string]json.RawMessage
	if err := json.Unmarshal(fields["containment"], &containment); err != nil {
		return err
	}
	for _, name := range []string{"mode", "qualified", "qualificationDigest"} {
		value, ok := containment[name]
		if !ok || string(value) == "null" && name != "qualificationDigest" {
			return fmt.Errorf("candidate report omitted/null containment %s", name)
		}
	}
	return nil
}
