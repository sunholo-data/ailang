// Package cihygiene holds repo-level guards over the GitHub Actions workflow
// definitions. It has no production code: the tests ARE the gate.
//
// Why this exists (measured, iteration 232, 2026-08-19/20):
//
// No job in any workflow declared timeout-minutes, so every job inherited
// GitHub's default 6-hour limit. Two jobs then wedged in one iteration, both on
// unbounded `apt-get install` shell-outs to a package mirror:
//
//	CI / test        step "Install z3"  >26m (attempt 1), 17m37s (attempt 2)
//	                 against a control of 49s / 100s / 9s on the three
//	                 immediately preceding completed dev runs.
//	Deploy Docs /    step "Install jq"  1h30m.
//	docs-build
//
// The provider's status API read "All Systems Operational" throughout, and
// attempt 3 cleared z3 in ~1 minute on a byte-identical tree -- outcome
// divergence with the code held constant, which pins the cause to the
// environment rather than to any diff. `test` and `docs-gate` are REQUIRED
// contexts, so a wedged mirror blocks every merge in the repo, for both
// missions on the rig, with no alarm and no upper bound.
//
// The fix is a bound, not a retry: a retry loop around an unbounded command
// inherits the unboundedness. A timeout converts an invisible multi-hour stall
// into a fast, legible red.
//
// The gate is written against a real YAML parser rather than grep on purpose.
// A gate's coverage is a property of its ENUMERATOR, one level below its
// branches -- a line-oriented matcher is defeated by comments, quoting, block
// scalars and indentation, and would report clean while seeing nothing.
package cihygiene

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// workflowDir is relative to this package (internal/cihygiene).
const workflowDir = "../../.github/workflows"

// maxJobTimeoutMinutes caps how lax a declared bound may be. GitHub's own
// default is 360; anything near it is a bound in name only.
const maxJobTimeoutMinutes = 180

// osPackageInstall matches a shell-out to an OS package mirror -- the class
// that was actually measured wedging. Deliberately NOT widened to every
// network fetch: `go mod download`, `npm ci` and the setup-* actions are
// already inside jobs that now carry a job-level bound, and a gate that fires
// on everything gets suppressed rather than fixed.
var osPackageInstall = regexp.MustCompile(`\b(apt-get|apt|yum|dnf|apk|brew|choco)\b[^\n]*\b(install|update|add|upgrade)\b`)

type workflow struct {
	Jobs map[string]struct {
		Uses            string `yaml:"uses"`
		TimeoutMinutes  *int   `yaml:"timeout-minutes"`
		ContinueOnError any    `yaml:"continue-on-error"`
		Steps           []struct {
			Name            string `yaml:"name"`
			Run             string `yaml:"run"`
			TimeoutMinutes  *int   `yaml:"timeout-minutes"`
			ContinueOnError any    `yaml:"continue-on-error"`
			Shell           string `yaml:"shell"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// loadWorkflows enumerates every workflow file and parses it. Both extensions
// are read: GitHub accepts .yml and .yaml, and an enumerator that knows only
// one of them reports clean on a file it never opened.
func loadWorkflows(t *testing.T) map[string]workflow {
	t.Helper()

	info, err := os.Stat(workflowDir)
	if err != nil || !info.IsDir() {
		// A missing directory makes every glob below return zero matches, and a
		// zero that means "the path is wrong" is indistinguishable from a zero
		// that means "nothing to check" unless the scope is asserted first.
		t.Fatalf("instrument failure: %s is not a directory (%v)", workflowDir, err)
	}

	var paths []string
	for _, pat := range []string{"*.yml", "*.yaml"} {
		m, err := filepath.Glob(filepath.Join(workflowDir, pat))
		if err != nil {
			t.Fatalf("instrument failure: glob %q: %v", pat, err)
		}
		paths = append(paths, m...)
	}
	sort.Strings(paths)

	if len(paths) == 0 {
		t.Fatal("instrument failure: no workflow files found -- the gate is vacuous")
	}

	out := make(map[string]workflow, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p) //nolint:gosec // fixed repo-relative path
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		var wf workflow
		if err := yaml.Unmarshal(b, &wf); err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		out[filepath.Base(p)] = wf
	}
	return out
}

// TestEveryJobDeclaresATimeout is the primary gate: a job with no
// timeout-minutes inherits a 6-hour ceiling nobody is watching.
func TestEveryJobDeclaresATimeout(t *testing.T) {
	wfs := loadWorkflows(t)

	var checked, exempt int
	for name, wf := range wfs {
		for jobID, job := range wf.Jobs {
			// A job that CALLS a reusable workflow cannot carry
			// timeout-minutes -- GitHub rejects the key there. The bound is the
			// callee's to declare, so this is a real exemption rather than a
			// waiver, and it is counted so it stays visible.
			if job.Uses != "" {
				exempt++
				if job.TimeoutMinutes != nil {
					t.Errorf("%s: job %q calls a reusable workflow and also sets timeout-minutes; "+
						"GitHub rejects that key on a `uses:` job", name, jobID)
				}
				continue
			}
			checked++
			if job.TimeoutMinutes == nil {
				t.Errorf("%s: job %q declares no timeout-minutes, so it inherits GitHub's "+
					"6-hour default; a wedged step burns the whole slot with no signal", name, jobID)
				continue
			}
			switch v := *job.TimeoutMinutes; {
			case v <= 0:
				t.Errorf("%s: job %q has timeout-minutes: %d (must be positive)", name, jobID, v)
			case v > maxJobTimeoutMinutes:
				t.Errorf("%s: job %q has timeout-minutes: %d, above the %d-minute cap -- "+
					"that is a bound in name only", name, jobID, v, maxJobTimeoutMinutes)
			}
		}
	}

	if checked == 0 {
		t.Fatal("instrument failure: zero jobs examined -- the gate is vacuous")
	}
	t.Logf("%d workflow files, %d jobs bounded, %d reusable-workflow calls exempt",
		len(wfs), checked, exempt)
}

// TestPackageInstallStepsAreBounded pins the step level. A job-level bound
// alone reds the whole job with no indication of WHICH step stalled; a step
// bound names it, which is the difference between "the job timed out" and "the
// package mirror is the problem".
func TestPackageInstallStepsAreBounded(t *testing.T) {
	wfs := loadWorkflows(t)

	var matched int
	for name, wf := range wfs {
		for jobID, job := range wf.Jobs {
			for i, step := range job.Steps {
				if step.Run == "" || !osPackageInstall.MatchString(step.Run) {
					continue
				}
				matched++
				if step.TimeoutMinutes == nil {
					label := step.Name
					if label == "" {
						label = fmt.Sprintf("step #%d", i)
					}
					t.Errorf("%s: job %q step %q shells out to a package mirror with no "+
						"step-level timeout-minutes:\n    %s",
						name, jobID, label, strings.TrimSpace(step.Run))
				}
			}
		}
	}
	// No anti-vacuity FLOOR here on purpose: removing the last package install
	// is a legitimate end state. The classifier itself is pinned by
	// TestOSPackageInstallClassifier below, so a silently-broken pattern cannot
	// masquerade as a clean repo.
	t.Logf("%d package-install steps found, all bounded", matched)
}

// TestOSPackageInstallClassifier is the enumerator's control. Without it, a
// regex that stopped matching anything would make the step gate above pass
// vacuously and look identical to a repo with nothing to bound.
func TestOSPackageInstallClassifier(t *testing.T) {
	positives := []string{
		"sudo apt-get update -qq && sudo apt-get install -y -qq --no-install-recommends z3",
		"sudo apt-get update && sudo apt-get install -y jq",
		"sudo apt-get install -y -qq --no-install-recommends jq && jq --version",
		"brew install z3",
		"sudo apk add --no-cache jq",
		"sudo dnf install -y jq",
	}
	negatives := []string{
		"make test",
		"go mod download",
		"npm ci",
		"go install ./cmd/ailang",
		"echo 'installing nothing'",
	}
	for _, s := range positives {
		if !osPackageInstall.MatchString(s) {
			t.Errorf("classifier missed a package install: %q", s)
		}
	}
	for _, s := range negatives {
		if osPackageInstall.MatchString(s) {
			t.Errorf("classifier fired on a non-install command: %q", s)
		}
	}
}
