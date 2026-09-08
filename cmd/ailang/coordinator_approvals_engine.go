package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ailembed "github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The authority DECISION runs in AILANG; this file is only the shell that
// reaches it.
//
// Why the decision is not Go: it is a pure function from a declared
// configuration and a set of observed facts to a verdict with a reason —
// exactly the shape `internal/dashboard_transforms/budget_checker.ail` already
// occupies, and Axiom A4 is "Explicit Authority". A rule about who may merge
// unattended, buried in a Go if-chain, is neither explicit nor separately
// checkable. In AILANG it carries its own inline `tests [...]`, which run under
// `ailang test` without the Go suite, and it can be read by anyone deciding
// whether to trust the fleet.
//
// Measured 2026-09-08: module load 65ms once, then 13µs per call. Against the
// ~1.1s Firestore round trip this path already pays, the cost is not a
// consideration.
//
// ONE DELIBERATE DIVERGENCE FROM THE budget_checker PRECEDENT. That bridge
// keeps a second, Go implementation and silently falls back to it on any error
// (`log.Printf(...)` then `goCheckTaskBudget`). For a budget that is merely
// risky — two implementations of one rule, free to drift, with the substitution
// visible only in a log. For an AUTHORITY rule it would be a defect: a failure
// to load the module would hand the decision about who may merge to an
// unreviewed second copy. So there is no second copy. A decision that cannot be
// evaluated REFUSES, carrying the reason, which is also the safe direction —
// the refusal is exactly the default posture.

const authorityModule = "internal/dashboard_transforms/approval_authority"

var (
	authorityEngineOnce sync.Once
	authorityEngine     *ailembed.Engine
	authorityEngineErr  error
)

// authorityRoot finds the tree holding the authority module.
//
// The CLI runs from anywhere — the hook from the project root, a mission
// controller from $REPO, a package agent from a different repository entirely —
// so a bare ailembed.New(".") would resolve differently for each caller and
// silently change who is trusted. Walking up for the module itself makes the
// answer depend on the tree, not on the caller's cwd.
func authorityRoot() (string, error) {
	if root := os.Getenv("AILANG_ROOT"); root != "" {
		if _, err := os.Stat(filepath.Join(root, authorityModule+".ail")); err == nil {
			return root, nil
		}
		return "", fmt.Errorf("AILANG_ROOT=%s does not contain %s.ail", root, authorityModule)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, authorityModule+".ail")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s.ail found above %s (set AILANG_ROOT)", authorityModule, dir)
		}
		dir = parent
	}
}

// loadAuthorityEngine compiles the authority module once per process.
func loadAuthorityEngine() (*ailembed.Engine, error) {
	authorityEngineOnce.Do(func() {
		root, err := authorityRoot()
		if err != nil {
			authorityEngineErr = err
			return
		}
		eng := ailembed.New(root)
		if err := eng.Load(authorityModule); err != nil {
			authorityEngineErr = fmt.Errorf("loading %s: %w", authorityModule, err)
			return
		}
		authorityEngine = eng
	})
	return authorityEngine, authorityEngineErr
}

// deciderFacts is the Go-side gathering of what the DRIVER published about this
// session. Reading the environment is not a decision, so it stays here; what
// the facts MEAN is the AILANG module's business.
type deciderFacts struct {
	MissionControlActive bool   `json:"missionControlActive"`
	ControllerID         string `json:"controllerId"`
	MissionRole          string `json:"missionRole"`
	AttendedGrant        bool   `json:"attendedGrant"`
}

// callDecide runs the AILANG decision.
func callDecide(facts deciderFacts, trusted []string) (granted bool, identity, reason string, err error) {
	eng, err := loadAuthorityEngine()
	if err != nil {
		return false, "", "", err
	}

	trustedAny := make([]interface{}, 0, len(trusted))
	for _, t := range trusted {
		trustedAny = append(trustedAny, t)
	}

	v, err := eng.Call(authorityModule, "decide", map[string]interface{}{
		"missionControlActive": facts.MissionControlActive,
		"controllerId":         facts.ControllerID,
		"missionRole":          facts.MissionRole,
		"attendedGrant":        facts.AttendedGrant,
	}, trustedAny)
	if err != nil {
		return false, "", "", fmt.Errorf("calling %s.decide: %w", authorityModule, err)
	}

	fields, err := recordFields(v)
	if err != nil {
		return false, "", "", err
	}
	g, _ := fields["granted"].(bool)
	id, _ := fields["identity"].(string)
	rs, _ := fields["reason"].(string)
	return g, id, rs, nil
}

// callCoversRow asks whether a granted authority covers one row.
func callCoversRow(diffAvailable bool, evaluation string, requireReviewable, requireEvaluatorPass bool) (bool, error) {
	eng, err := loadAuthorityEngine()
	if err != nil {
		return false, err
	}
	v, err := eng.Call(authorityModule, "coversRow", map[string]interface{}{
		"diffAvailable": diffAvailable,
		"evaluation":    evaluation,
	}, requireReviewable, requireEvaluatorPass)
	if err != nil {
		return false, fmt.Errorf("calling %s.coversRow: %w", authorityModule, err)
	}
	goVal, err := ailembed.ToGo(v)
	if err != nil {
		return false, fmt.Errorf("converting %s.coversRow result: %w", authorityModule, err)
	}
	covered, ok := goVal.(bool)
	if !ok {
		return false, fmt.Errorf("%s.coversRow returned %T, want bool", authorityModule, goVal)
	}
	return covered, nil
}

// recordFields converts an AILANG record result to a Go map, reporting the
// shape it actually got rather than yielding an empty verdict — a silently
// empty Verdict would read as "not granted, no reason", which is a refusal
// nobody can debug.
func recordFields(v eval.Value) (map[string]interface{}, error) {
	goVal, err := ailembed.ToGo(v)
	if err != nil {
		return nil, fmt.Errorf("converting %s result: %w", authorityModule, err)
	}
	fields, ok := goVal.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s returned %T, want a record", authorityModule, goVal)
	}
	return fields, nil
}
