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
	// Flags such as `-C <dir>` / `-s` are skipped so `make -C sub target` is seen.
	// The prefix class is deliberately NARROW: iteration 275 round 2 widened it to admit
	// `!` and if/while/until, and it began capturing prose -- `echo "Great! make sure to
	// run X"` yielded the target `sure`, a false RED on any future PR containing "make
	// sure". Detection of a gate inside shell control flow is now done by NAME, in
	// gateTargetsTouched, which no syntax can hide from.
	makeInvocation = regexp.MustCompile(`(?m)(?:^|[;&|(]|&&|\|\|)[ \t]*(?:sudo[ \t]+)?make[ \t]+(?:-[A-Za-z-]+(?:[ \t]+[^ \t]+)?[ \t]+)*([A-Za-z0-9][A-Za-z0-9_.-]*)`)
	// shellComment strips `#`-to-end-of-line before matching.
	shellComment = regexp.MustCompile(`(?m)#.*$`)
)

var notWiredIntoCI = map[string]string{
	"check-claude":                  "developer-local: needs the Claude CLI installed; CI has no Claude auth",
	"check-no-zero-arg-workarounds": "advisory sweep, not a gate: narrow grep for a historical surface-local workaround superseded by eval.CallFunction",
	"check-pi-wire-budget":          "developer-local: makes a REAL paid pi/OpenRouter call, needs OPENROUTER_API_KEY (CI has none), and reads the result back from OpenRouter Broadcast ingest, so a quiet ingest reds it with nothing wrong in the repo. It is an on-demand instrument, not a verdict: it exists precisely because every config-vs-config gate is blind to the wire — pi-ai clamps the output budget to min(declared, 32000) downstream of every config file, and TestPiModelsConfigMatchesRegistry compared 65536 to 65536 and passed for weeks while every request went out at 32000",
}

var notWiredIntoCIVerify = map[string]string{
	"verify-lowering":          "body is `build verify-no-shim` plus one echo; verify-no-shim is already wired at ci.yml:201, so wiring this adds an alias, not coverage",
	"verify-model-pricing":     "queries the live OpenRouter API (network + third party) and its own Makefile header declares the result ADVISORY, not a verdict; a vendor promotion reds it with nothing wrong in the repo",
	"fmt-check-ail":            "RED at HEAD 2026-08-25 (rc=2). The enumerator half of this reason was FIXED in iteration 277 (roots are now `examples std`, a missing root and an empty enumeration both fail loudly), so the gate now sees all 450 .ail files rather than 404. It stays exempt because the CORPUS is red, and far more so than this entry used to say: a per-file scan over the corrected scope measured ok=63, drift=341, err=46 — i.e. only 14% of the corpus is canonical. The 46 errors are 38 comment-attachment refusals, 7 parse failures in deliberately-invalid corpora (examples/archive/broken, examples/bugs, examples/experimental), and 1 formatter SOUNDNESS defect (std/cognition.ail: valid input, `ailang check` rc=0, but formatted output fails to re-parse; it fails closed and does not corrupt). The single-file framing was an artifact of `ailang fmt --check` aborting the whole scan at the first error. Tracked as open queue rows and decision D-38 — wire it when it is green, do not widen the exemption",
	"verify-cli-examples":      "RED at HEAD 2026-08-25 (rc=2): fixture examples/cli_examples.txt has rotted: 9 of 26 commands fail, incl. list_sum.ail expecting `(15, 15)` and producing `(15, 5)`, and lambdas_full.ail using `++` on strings (list-only since v0.13.0); tracked as an open queue row — wire it when it is green, do not widen the exemption",
	"verify-examples-all":      "RED at HEAD 2026-08-25 (rc=2): 60% threshold gate over ALL examples currently exits 1; tracked as an open queue row — wire it when it is green, do not widen the exemption",
	"verify-examples-toplevel": "RED at HEAD 2026-08-25 (rc=2): examples/ai_modes.ail fails effect checking: \"AI requires mode=fixed; declaration provides mode=routeable\". NOTE: this target is a `make ci` prerequisite, so `make ci` is RED at HEAD on this one example; tracked as an open queue row — wire it when it is green, do not widen the exemption",
}

var nonCanonicalAllowed = map[string]string{
	"verify-examples-trace": "advisory trace-determinism sweep, rc=1 on 2 of 217 examples at 2026-08-25; the `|| true` at ci.yml is deliberate and tracked as an open queue row -- REMOVE this entry when the target is green rather than leaving it suppressed",
	"verify-examples":       "piped to `tee` so the job can attach the output as an artifact; safe because the step runs under the default pipefail shell, but the safety depends on that shell rather than on the line, which is why it needs a stated reason",
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
	"verify-examples-trace": "advisory 42s trace-determinism sweep, rc=1 on 2 of 217 examples at 2026-08-25; invoked by ci.yml under a declared `|| true` (see nonCanonicalAllowed) and deliberately NOT a `make ci` member -- remove BOTH entries together when the target goes green",
	"check-autoclose":       "range-shaped: anti-vacuity floor exits rc=2 on an empty commit range, which is the state of a clean freshly-pulled checkout",
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

func TestWiredGatesAreCanonical(t *testing.T) {
	known := makeTargets(t)
	scanned, sawTrace := 0, false
	for workflowName, wf := range loadWorkflows(t) {
		for jobID, job := range wf.Jobs {
			for stepIndex, step := range job.Steps {
				script := shellComment.ReplaceAllString(step.Run, "")
				for target, lines := range gateTargetsTouched(script, known) {
					scanned++
					if target == "verify-examples-trace" {
						sawTrace = true
					}
					where := workflowName + ":" + jobID + ":step " + strconv.Itoa(stepIndex)
					var reasons []string
					for _, line := range lines {
						if !canonicalGateLine.MatchString(line) {
							reasons = append(reasons, "non-canonical run line `"+strings.TrimSpace(line)+"`")
						}
					}
					if errexitDisabled.MatchString(script) {
						reasons = append(reasons, "the step disables errexit or installs a trap")
					}
					if step.If != "" || job.If != "" {
						reasons = append(reasons, "gated by an `if:` condition, so the gate may never run at all")
					}
					if continueOnError(job.ContinueOnError) || continueOnError(step.ContinueOnError) {
						reasons = append(reasons, "continue-on-error")
					}
					if !pipefailSafeShell(step.Shell) {
						reasons = append(reasons, "shell "+strconv.Quote(step.Shell)+" does not set pipefail")
					}
					if len(reasons) == 0 {
						continue
					}
					if _, allowed := nonCanonicalAllowed[target]; !allowed {
						t.Errorf("%s invokes %s in a form that may not fail the job: %s", where, target, strings.Join(reasons, "; "))
					}
				}
			}
		}
	}
	if scanned < 15 || !sawTrace {
		t.Fatalf("instrument failure: canonical-form scan covered %d gate invocations and saw verify-examples-trace=%v (need at least 15, including the known-suppressed trace step)", scanned, sawTrace)
	}
}

// continueOnError reads GitHub's `continue-on-error`, which may be a literal bool, the
// strings "true"/"false", or a `${{ }}` expression. An expression is treated as
// SUPPRESSED: it MAY evaluate true, and a gate that might not fail the job is not a gate.
func continueOnError(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != "false"
	default:
		return true
	}
}

// pipefailSafeShell reports whether a step's shell applies `set -o pipefail`. GitHub's
// default for `run:` on Linux/macOS is `bash --noprofile --norc -eo pipefail`, and an
// explicit `shell: bash` resolves to the same command line. Anything else -- sh, pwsh,
// python, a custom template -- does not, so a pipe there discards the gate's exit code.
func pipefailSafeShell(shell string) bool {
	return shell == "" || shell == "bash"
}

var (
	// canonicalGateLine is the ALLOWED shape: optional leading VAR=value assignments,
	// `make`, optional flags, the target, optional trailing VAR=value arguments. No shell
	// metacharacters, no control flow, no redirection.
	//
	// The unquoted value class EXCLUDES shell metacharacters, and that exclusion is
	// load-bearing rather than tidy. Its first draft was `[^ \t]*`, and bash does not
	// require whitespace around `||`, `&` or `;` -- so `make check-boundaries FOO=a||true`
	// parsed as a canonical line with a harmless-looking argument and is a REAL bypass
	// (measured: `bash -eo pipefail -c 'make -C /tmp nonexistent FOO=a||true'` exits 0
	// against rc=2 without the `||`). The whitelist smuggled the exact suppression it
	// exists to reject through the one syntax it declares safe. Found by the iteration-275
	// round-3 evaluator.
	canonicalGateLine = regexp.MustCompile(`^[ \t]*(?:[A-Za-z_][A-Za-z0-9_]*=[^ \t|&;()<>$\x60'"]*[ \t]+)*make[ \t]+(?:-[A-Za-z-]+[ \t]+)*[A-Za-z0-9][A-Za-z0-9_.-]*(?:[ \t]+[A-Za-z_][A-Za-z0-9_]*=(?:"[^"]*"|'[^']*'|[^ \t|&;()<>$\x60'"]*))*[ \t]*$`)
	// errexitDisabled covers `set +e`, `set +eu`, `set +o errexit` and `trap ... EXIT`.
	errexitDisabled = regexp.MustCompile(`(?m)^[ \t]*(?:set[ \t]+[+](?:[a-zA-Z]*e[a-zA-Z]*\b|o[ \t]+errexit\b)|trap[ \t])`)
)

// targetToken matches a make target as a whole token. A plain \b boundary is wrong here:
// `-` is a non-word character, so `\bverify-examples\b` matches INSIDE
// `verify-examples-trace`.
func targetToken(target string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_.-])` + regexp.QuoteMeta(target) + `($|[^A-Za-z0-9_.-])`)
}

// gateTargetsTouched returns every gate- or verify-shaped target NAMED on a `make` line of
// this script, mapped to the lines that name it.
//
// Detection is by NAME, not by the invocation regex, and that is the whole point. Rounds 1
// and 2 of iteration 275 were a sequence of shell shapes the regex did not anticipate --
// `|| true`, `|| exit 0`, `|| { echo fail; }`, `|| (exit 0)`, `set +e`, `set +o errexit`,
// a background `&`, an `if !` condition, a `trap` -- and each fix was another pattern. An
// enumeration of FORBIDDEN shapes is never complete. An enumeration of ALLOWED shapes is
// complete by construction: anything that is not canonical becomes a loud red demanding a
// stated reason, which is the same discipline the exemption maps already impose. A target
// name on a `make` line cannot be hidden by syntax.
func gateTargetsTouched(script string, known map[string]bool) map[string][]string {
	touched := make(map[string][]string)
	for _, line := range strings.Split(script, "\n") {
		if !strings.Contains(line, "make") {
			continue
		}
		for target := range known {
			if !gateTarget(target) && !verifyTarget(target) {
				continue
			}
			if targetToken(target).MatchString(line) {
				touched[target] = append(touched[target], line)
			}
		}
	}
	return touched
}

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
		if !gateTarget(target) && !verifyTarget(target) {
			t.Errorf("notInMakeCI exempts %q, which neither predicate selects -- the exemption is dead", target)
		}
		if len(invoked[target]) == 0 {
			t.Errorf("notInMakeCI exempts %q, but no workflow invokes it -- the exemption cannot apply", target)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("notInMakeCI exempts %q with an empty reason", target)
		}
	}

	// nonCanonicalAllowed's precondition is that the target is actually invoked.
	for target, reason := range nonCanonicalAllowed {
		checked++
		if !targets[target] {
			t.Errorf("nonCanonicalAllowed permits %q, which is not a make target -- stale exemption", target)
		}
		if !gateTarget(target) && !verifyTarget(target) {
			t.Errorf("nonCanonicalAllowed permits %q, which neither predicate selects -- the entry is dead", target)
		}
		if len(invoked[target]) == 0 {
			t.Errorf("nonCanonicalAllowed permits %q, but no workflow invokes it -- remove the entry", target)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("nonCanonicalAllowed permits %q with an empty reason", target)
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

	// Widened to verifyTarget at iteration 275 round 2. Before that, this test covered
	// only check-*, so the two verify-* targets this very sprint added to `ci:` were
	// themselves unprotected -- a fresh instance of the "wired but not verified-connected"
	// defect this lineage exists to close. Found by the iteration-275 evaluator.
	missing := make(map[string]bool)
	scanned := 0
	for target := range invoked {
		if !gateTarget(target) && !verifyTarget(target) {
			continue
		}
		scanned++
		if !ciPrerequisites[target] {
			if _, exempt := notInMakeCI[target]; !exempt {
				missing[target] = true
			}
		}
	}
	// The known-positive is `lint` -- a ci: member that is neither gate- nor
	// verify-shaped, so it is not the subject of this test. Asserting a target that IS
	// under test would make the floor, rather than the `missing` check, kill every
	// mutant, and a red attributed to the wrong assertion is not evidence.
	if scanned < 15 || !ciPrerequisites["lint"] {
		t.Fatalf("instrument failure: ci-superset scan covered %d invoked gate/verify targets and ci: lists lint=%v (need at least 15, and a parseable ci: line)", scanned, ciPrerequisites["lint"])
	}
	if len(missing) != 0 {
		t.Errorf("workflow gates missing from ci prerequisites: %s; add them to make/ci.mk", strings.Join(sortedKeys(missing), ", "))
	}
}
