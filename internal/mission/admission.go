package mission

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// AdmissionPolicy shares the observers and ration evaluation used by mission quota.
// Inject observers in tests; production callers use NewAdmissionPolicy.
type AdmissionPolicy struct {
	Now    func() time.Time
	Codex  func(time.Time) CodexQuotaObservation
	Ollama func(time.Time) OllamaQuotaObservation
	Ledger func(time.Time) (*Ledger, error)
}

func NewAdmissionPolicy(paths Paths) *AdmissionPolicy {
	return &AdmissionPolicy{Now: time.Now,
		Codex: func(now time.Time) CodexQuotaObservation {
			home := os.Getenv("CODEX_HOME")
			if home == "" {
				home = filepath.Join(paths.Home, ".codex")
			}
			return ObserveCodexQuota(home, now)
		},
		Ollama: func(now time.Time) OllamaQuotaObservation {
			return ObserveOllamaQuota(paths, os.Getenv("OLLAMA_API_KEY"), now)
		},
		Ledger: func(now time.Time) (*Ledger, error) { return LoadLedger(paths, now) },
	}
}
func (p *AdmissionPolicy) Check(ctx context.Context, c dispatch.Candidate) (dispatch.Admission, error) {
	if err := ctx.Err(); err != nil {
		return dispatch.Admission{}, err
	}
	if p == nil || p.Now == nil {
		return dispatch.Admission{}, fmt.Errorf("missing admission policy")
	}
	now := p.Now()
	o := dispatch.Admission{ObservedAt: now, Policy: "mission-subscription-v1"}
	switch {
	case c.Executor == "codex":
		if p.Codex == nil {
			return o, fmt.Errorf("missing Codex observer")
		}
		q := p.Codex(now)
		o.Bucket = "codex"
		o.Allowed = !q.Blocked()
		o.Reason = q.Reason
		o.ObservedAt = q.ObservedAt
		// Unknown snapshots can have no timestamp; timestamp the refusal, not fabricated usage.
		if o.ObservedAt.IsZero() {
			o.ObservedAt = now
		}
	case modelreg.IsOllamaCloudRoute(c.WireModel):
		if p.Ollama == nil {
			return o, fmt.Errorf("missing Ollama observer")
		}
		q := p.Ollama(now)
		o.Bucket = "ollama"
		o.Allowed = !q.Blocked()
		o.Reason = q.Reason
	default:
		if p.Ledger == nil {
			return o, fmt.Errorf("missing token ledger observer")
		}
		bucket := c.Transport
		if c.Executor == "claude" {
			bucket = "anthropic"
		}
		if bucket != "anthropic" && bucket != "openrouter" && bucket != "opencode" {
			return o, fmt.Errorf("unsupported quota bucket %q", bucket)
		}
		ledger, err := p.Ledger(now)
		if err != nil {
			return o, err
		}
		if ledger == nil {
			return o, fmt.Errorf("missing ledger")
		}
		o.Bucket = bucket
		o.Allowed = true
		o.Reason = "no proven ledger exceedance; capacity may be unknown"
		for _, v := range ledger.Verdicts(now) {
			if v.Bucket == bucket && v.Over() {
				o.Allowed = false
				o.Reason = v.Reason
			}
		}
	}
	return o, ctx.Err()
}
