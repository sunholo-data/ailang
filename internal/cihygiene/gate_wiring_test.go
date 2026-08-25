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
	makeInvocation = regexp.MustCompile(`\bmake\s+([A-Za-z0-9][A-Za-z0-9_.-]*)`)
)

var notWiredIntoCI = map[string]string{
	"check-claude":                  "developer-local: needs the Claude CLI installed; CI has no Claude auth",
	"check-no-zero-arg-workarounds": "advisory sweep, not a gate: narrow grep for a historical surface-local workaround superseded by eval.CallFunction",
}

var notInMakeCI = map[string]string{
	"check-autoclose": "event-shaped: needs AUTOCLOSE_ARGS; measured rc=2 with no args, so it cannot be an unconditional local gate",
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
				for _, match := range makeInvocation.FindAllStringSubmatch(step.Run, -1) {
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

func gateTarget(target string) bool {
	return strings.HasPrefix(target, "check-") || strings.HasPrefix(target, "test-check-")
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
	// This deliberately covers only check-* and test-check-* targets. Whether
	// the repo's verify-* targets belong in CI is a separate policy decision.
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
