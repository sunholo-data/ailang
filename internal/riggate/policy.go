// Package riggate is the rig GPU admission gateway (M-RIG-GPU-ADMISSION-GATEWAY):
// a loopback reverse proxy in front of ollama that admits long GPU work only from
// the holder of the rig lock's lease.
//
// Why it exists: the rig lock gated STARTING a job, and nothing gated SENDING a
// request to ollama. On 2026-09-27 two orphaned agents with no lease streamed at
// the 27B model for 15-18 hours; with OLLAMA_NUM_PARALLEL=1 every real eval
// queued behind them, and the day produced no usable eval rows.
//
// The admission DECISION is AILANG (internal/dashboard_transforms/rig_admission.ail),
// per the project rule that policy lives in AILANG and Go is the shell. This
// package only gathers facts, calls it, proxies, and writes the ledger. There is
// no Go copy of the rule: if the module cannot be evaluated the request fails
// loudly (500), never silently admitted.
package riggate

import (
	"fmt"
	"sync"

	ailembed "github.com/sunholo-data/ailang/internal/embed"
)

const policyModule = "internal/dashboard_transforms/rig_admission"

// Facts is what the Go shell observed about one request.
type Facts struct {
	Path          string
	LeaseHeld     bool
	LeaseHasToken bool
	TokenPresent  bool
	TokenMatches  bool
	Holder        string
}

// Verdict is the policy's answer.
type Verdict struct {
	Admit    bool
	Status   int
	Decision string
	Reason   string
}

// Decider evaluates Facts. The production one is the AILANG module; tests may
// substitute their own only through NewGate's argument, never by fallback.
type Decider interface {
	Decide(Facts) (Verdict, error)
}

// Policy is the AILANG-backed Decider.
type Policy struct {
	mu  sync.Mutex
	eng *ailembed.Engine
}

// LoadPolicy compiles the admission module once. Call at startup so a broken
// module stops the gateway from starting rather than failing every request.
func LoadPolicy() (*Policy, error) {
	eng, err := ailembed.NewForModule(policyModule)
	if err != nil {
		return nil, fmt.Errorf("riggate: loading %s: %w", policyModule, err)
	}
	return &Policy{eng: eng}, nil
}

// Decide calls rig_admission.decide.
func (p *Policy) Decide(f Facts) (Verdict, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, err := p.eng.Call(policyModule, "decide", map[string]interface{}{
		"path":          f.Path,
		"leaseHeld":     f.LeaseHeld,
		"leaseHasToken": f.LeaseHasToken,
		"tokenPresent":  f.TokenPresent,
		"tokenMatches":  f.TokenMatches,
		"holder":        f.Holder,
	})
	if err != nil {
		return Verdict{}, fmt.Errorf("riggate: calling %s.decide: %w", policyModule, err)
	}
	goVal, err := ailembed.ToGo(v)
	if err != nil {
		return Verdict{}, fmt.Errorf("riggate: converting verdict: %w", err)
	}
	m, ok := goVal.(map[string]interface{})
	if !ok {
		return Verdict{}, fmt.Errorf("riggate: %s.decide returned %T, want a record", policyModule, goVal)
	}
	admit, ok1 := m["admit"].(bool)
	decision, ok2 := m["decision"].(string)
	reason, ok3 := m["reason"].(string)
	status, ok4 := asInt(m["status"])
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return Verdict{}, fmt.Errorf("riggate: malformed verdict %v", m)
	}
	return Verdict{Admit: admit, Status: status, Decision: decision, Reason: reason}, nil
}

func asInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}
