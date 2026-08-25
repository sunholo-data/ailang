package cihygiene

import (
	"bytes"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var (
	makeTargetLine = regexp.MustCompile(`(?m)^([A-Za-z0-9][A-Za-z0-9_.-]*):`)
	// `make` must sit at a COMMAND position: start of line, or after a shell
	// separator. A bare \bmake\b also matches prose inside the run script --
	// `echo "run make check-foo by hand"` and `# TODO: wire up make check-foo`
	// both read as invocations, so a gate could be listed in ci: and mentioned
	// in a comment while never actually executing. Found by the iteration-274
	// evaluator, which demonstrated exactly that pair passing all three checks.
	makeInvocation = regexp.MustCompile(`(?m)(?:^|[;&|(]|&&|\|\|)[ \t]*(?:sudo[ \t]+)?make[ \t]+([A-Za-z0-9][A-Za-z0-9_.-]*)`)
	// shellComment strips `#`-to-end-of-line before matching.
	shellComment = regexp.MustCompile(`(?m)#.*$`)
)

var notWiredIntoCI = map[string]string{
	"check-claude":                  "developer-local: needs the Claude CLI installed; CI has no Claude auth",
	"check-no-zero-arg-workarounds": "advisory sweep, not a gate: narrow grep for a historical surface-local workaround superseded by eval.CallFunction",
}

var notWiredIntoCIVerify = map[string]string{
	"verify-lowering":          "body is `build verify-no-shim` plus one echo; verify-no-shim is already wired at ci.yml:201, so wiring this adds an alias, not coverage",
	"verify-model-pricing":     "queries the live OpenRouter API (network + third party) and its own Makefile header declares the result ADVISORY, not a verdict; a vendor promotion reds it with nothing wrong in the repo",
	"fmt-check-ail":            "RED at HEAD 2026-08-25 (rc=2): enumerator scans a `stdlib/` directory that has never existed (find examples stdlib -> 404 .ail files; find examples std -> 450, so 46 files are invisible), AND real fmt drift in 2 example files, AND `ailang fmt` errors on examples/snippets/v3_3/math/gcd.ail; tracked as an open queue row — wire it when it is green, do not widen the exemption",
	"verify-cli-examples":      "RED at HEAD 2026-08-25 (rc=2): fixture examples/cli_examples.txt has rotted: 9 of 26 commands fail, incl. list_sum.ail expecting `(15, 15)` and producing `(15, 5)`, and lambdas_full.ail using `++` on strings (list-only since v0.13.0); tracked as an open queue row — wire it when it is green, do not widen the exemption",
	"verify-examples-all":      "RED at HEAD 2026-08-25 (rc=2): 60% threshold gate over ALL examples currently exits 1; tracked as an open queue row — wire it when it is green, do not widen the exemption",
	"verify-examples-toplevel": "RED at HEAD 2026-08-25 (rc=2): examples/ai_modes.ail fails effect checking: \"AI requires mode=fixed; declaration provides mode=routeable\". NOTE: this target is a `make ci` prerequisite, so `make ci` is RED at HEAD on this one example; tracked as an open queue row — wire it when it is green, do not widen the exemption",
}

var suppressionAllowed = map[string]string{
	"verify-examples-trace": "advisory trace-determinism sweep, currently rc=1 on 2 of 217 examples; the `|| true` is deliberate and tracked as an open queue row — REMOVE this entry when the target is green rather than leaving it suppressed",
}

var notInMakeCI = map[string]string{
	// Measured 2026-08-25, both arms, after the iteration-274 evaluator refuted
	// the first draft of this reason (which said "needs AUTOCLOSE_ARGS" -- false:
	// code-health.mk defaults it with ?=). The true reason is RANGE-shaped, not
	// event-shaped: the target carries an anti-vacuity floor and exits rc=2
	// "no records enumerated" whenever the range is empty.
	//   at origin/dev  (0 commits in range) -> rc=2
	//   1 commit ahead (1 commit in range)  -> rc=0
	// A developer on a freshly-pulled clean checkout is the rc=2 case, so this
	// cannot be an unconditional member of the local `make ci` aggregate. CI
	// invokes it directly with event-scoped args (ci.yml:157-175), and its
	// self-test test-check-autoclose IS in ci:.
	"check-autoclose": "range-shaped: anti-vacuity floor exits rc=2 on an empty commit range, which is the state of a clean freshly-pulled checkout",
}

func makeTargetsAndDatabase(t *testing.T) (map[string]bool, string) {
	t.Helper()

	cmd := exec.Command("make", "-C", "../..", "-pn")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("instrument failure: make -C ../.. -pn: %v", err)
	}

	database := stdout.String()
	targets := make(map[string]bool)
	for _, line := range strings.Split(database, "\n") {
		if strings.Contains(line, ":=") {
			continue
		}
		if match := makeTargetLine.FindStringSubmatch(line); match != nil {
			targets[match[1]] = true
		}
	}
	if len(targets) == 0 || len(targets) < 120 || !targets["build"] {
		t.Fatalf("instrument failure: make target enumeration found %d targets (need at least 120, including build)", len(targets))
	}
	return targets, database
}

func makeTargets(t *testing.T) map[string]bool {
	t.Helper()
	targets, _ := makeTargetsAndDatabase(t)
	return targets
}

func invokedTargets(t *testing.T) map[string][]string {
	t.Helper()

	invoked := make(map[string][]string)
	for workflowName, wf := range loadWorkflows(t) {
		for jobID, job := range wf.Jobs {
			for stepIndex, step := range job.Steps {
				script := shellComment.ReplaceAllString(step.Run, "")
				for _, match := range makeInvocation.FindAllStringSubmatch(script, -1) {
					where := workflowName + ":" + jobID + ":step " + strconv.Itoa(stepIndex)
					invoked[match[1]] = append(invoked[match[1]], where)
				}
			}
		}
	}
	if len(invoked) == 0 || len(invoked["check-file-sizes"]) == 0 {
		t.Fatal("instrument failure: workflow make-target enumeration is empty or missed check-file-sizes")
	}
	return invoked
}

// KNOWN LIMITATION (declared, not fixed -- dormant at HEAD): loadWorkflows reads
// only .github/workflows/*.yml|*.yaml and this scan reads only step.Run, so a
// `make` invocation inside a composite action (uses: ./.github/actions/x) or a
// first-party reusable workflow is invisible. Measured by the iteration-274
// evaluator: the repo has ZERO composite actions and ZERO first-party reusable
// workflows today, and the failure direction is a FALSE RED (the target reads as
// unwired), which is loud rather than silent. If either is ever added, widen the
// enumerator -- do not answer the red with an exemption-map entry.
func gateTarget(target string) bool {
	return strings.HasPrefix(target, "check-") || strings.HasPrefix(target, "test-check-")
}

func verifyTarget(target string) bool {
	return strings.HasPrefix(target, "verify-") || target == "fmt-check-ail"
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestWorkflowMakeTargetsExist(t *testing.T) {
	// A bogus target already fails loudly with rc=2 when its step runs. This
	// check earns its place for targets hidden behind if: conditions that rarely
	// fire and would otherwise remain stale unnoticed.
	targets := makeTargets(t)
	for target, locations := range invokedTargets(t) {
		if !targets[target] {
			t.Errorf("workflow invokes unknown make target %q at %s", target, strings.Join(locations, ", "))
		}
	}
}

func TestGateTargetsAreWiredIntoAWorkflow(t *testing.T) {
	// Keep check-* policy separate from verify-* policy below.
	targets := makeTargets(t)
	invoked := invokedTargets(t)
	missing := make(map[string]bool)
	for target := range targets {
		if gateTarget(target) && len(invoked[target]) == 0 {
			if _, exempt := notWiredIntoCI[target]; !exempt {
				missing[target] = true
			}
		}
	}
	if len(missing) != 0 {
		t.Errorf("gate-shaped make targets are not wired into any workflow: %s", strings.Join(sortedKeys(missing), ", "))
	}
}

func TestVerifyTargetsAreWiredIntoAWorkflow(t *testing.T) {
	targets := makeTargets(t)
	invoked := invokedTargets(t)
	missing := make(map[string]bool)
	verifyCount := 0
	for target := range targets {
		if !verifyTarget(target) {
			continue
		}
		verifyCount++
		if len(invoked[target]) == 0 {
			if _, exempt := notWiredIntoCIVerify[target]; !exempt {
				missing[target] = true
			}
		}
	}
	if verifyCount < 10 || !targets["verify-stdlib"] {
		t.Fatalf("instrument failure: verify-target enumeration found %d targets (need at least 10, including verify-stdlib)", verifyCount)
	}
	if len(missing) != 0 {
		t.Errorf("verify-shaped make targets are not wired into any workflow: %s", strings.Join(sortedKeys(missing), ", "))
	}
}

func TestWiredGatesCanFailTheJob(t *testing.T) {
	matched := 0
	foundTrace := false
	for workflowName, wf := range loadWorkflows(t) {
		for jobID, job := range wf.Jobs {
			for stepIndex, step := range job.Steps {
				script := shellComment.ReplaceAllString(step.Run, "")
				for _, line := range strings.Split(script, "\n") {
					for _, match := range makeInvocation.FindAllStringSubmatchIndex(line, -1) {
						target := line[match[2]:match[3]]
						if !gateTarget(target) && !verifyTarget(target) {
							continue
						}
						matched++
						if target == "verify-examples-trace" {
							foundTrace = true
						}
						where := workflowName + ":" + jobID + ":step " + strconv.Itoa(stepIndex)
						if (job.ContinueOnError != nil && *job.ContinueOnError) ||
							(step.ContinueOnError != nil && *step.ContinueOnError) {
							if _, allowed := suppressionAllowed[target]; !allowed {
								t.Errorf("%s suppresses failure from %s with continue-on-error", where, target)
							}
						}
						tail := line[match[1]:]
						if strings.Contains(tail, "|| true") || strings.Contains(tail, "|| :") || strings.Contains(tail, "|| echo") {
							if _, allowed := suppressionAllowed[target]; !allowed {
								t.Errorf("%s suppresses failure from %s in-shell: %s", where, target, strings.TrimSpace(line))
							}
						}
					}
				}
			}
		}
	}
	if matched == 0 || !foundTrace {
		t.Fatalf("instrument failure: wired-gate scan found %d invocations; must include verify-examples-trace", matched)
	}
}

// TestExemptionMapsAreLive pins the maps themselves. Every test above consults an
// exemption map only for targets its predicate already selected, so a predicate
// that SHRINKS silently drops coverage and the orphaned exemption entry keeps
// looking deliberate -- measured 2026-08-25 (iteration 275): removing
// `fmt-check-ail` from verifyTarget left the whole package rc=0. A removal proves
// a check FIRES; only asserting the map against the predicate proves it still
// LOOKS. Same for a target that is renamed, deleted, or later wired for real:
// the exemption outlives its reason with no signal.
func TestExemptionMapsAreLive(t *testing.T) {
	targets := makeTargets(t)
	invoked := invokedTargets(t)
	checked := 0

	// Maps whose precondition is "no workflow invokes this target".
	for _, entry := range []struct {
		name    string
		values  map[string]string
		inScope func(string) bool
	}{
		{"notWiredIntoCI", notWiredIntoCI, gateTarget},
		{"notWiredIntoCIVerify", notWiredIntoCIVerify, verifyTarget},
	} {
		for target, reason := range entry.values {
			checked++
			if !targets[target] {
				t.Errorf("%s exempts %q, which is not a make target -- stale exemption", entry.name, target)
			}
			if !entry.inScope(target) {
				t.Errorf("%s exempts %q, which its own predicate no longer selects -- the exemption is dead and the target is unchecked", entry.name, target)
			}
			if len(invoked[target]) != 0 {
				t.Errorf("%s exempts %q, but %s invokes it -- remove the exemption rather than leaving it", entry.name, target, strings.Join(invoked[target], ", "))
			}
			if strings.TrimSpace(reason) == "" {
				t.Errorf("%s exempts %q with an empty reason", entry.name, target)
			}
		}
	}

	// notInMakeCI's precondition is the opposite: the target IS invoked by a
	// workflow and is deliberately absent from `make ci`.
	for target, reason := range notInMakeCI {
		checked++
		if !targets[target] {
			t.Errorf("notInMakeCI exempts %q, which is not a make target -- stale exemption", target)
		}
		if !gateTarget(target) {
			t.Errorf("notInMakeCI exempts %q, which gateTarget no longer selects -- the exemption is dead", target)
		}
		if len(invoked[target]) == 0 {
			t.Errorf("notInMakeCI exempts %q, but no workflow invokes it -- the exemption cannot apply", target)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("notInMakeCI exempts %q with an empty reason", target)
		}
	}

	// suppressionAllowed's precondition is that the target is actually invoked.
	for target, reason := range suppressionAllowed {
		checked++
		if !targets[target] {
			t.Errorf("suppressionAllowed permits %q, which is not a make target -- stale exemption", target)
		}
		if !gateTarget(target) && !verifyTarget(target) {
			t.Errorf("suppressionAllowed permits %q, which neither predicate selects -- the entry is dead", target)
		}
		if len(invoked[target]) == 0 {
			t.Errorf("suppressionAllowed permits %q, but no workflow invokes it -- remove the entry", target)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("suppressionAllowed permits %q with an empty reason", target)
		}
	}

	if checked < 8 {
		t.Fatalf("instrument failure: exemption-map scan checked only %d entries (need at least 8 across four maps)", checked)
	}
}

func TestMakeCIIncludesWorkflowGates(t *testing.T) {
	_, database := makeTargetsAndDatabase(t)
	invoked := invokedTargets(t)
	ciLine := regexp.MustCompile(`(?m)^ci:\s*([^\n]*)$`).FindStringSubmatch(database)
	if ciLine == nil {
		t.Fatal("instrument failure: make -pn output contains no ci target")
	}
	ciPrerequisites := make(map[string]bool)
	for _, prerequisite := range strings.Fields(ciLine[1]) {
		ciPrerequisites[prerequisite] = true
	}

	missing := make(map[string]bool)
	for target := range invoked {
		if gateTarget(target) && !ciPrerequisites[target] {
			if _, exempt := notInMakeCI[target]; !exempt {
				missing[target] = true
			}
		}
	}
	if len(missing) != 0 {
		t.Errorf("workflow gates missing from ci prerequisites: %s; add them to make/ci.mk", strings.Join(sortedKeys(missing), ", "))
	}
}
