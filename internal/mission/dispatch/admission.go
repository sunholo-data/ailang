package dispatch

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Admission is recorded independently of token spend: account quota is an observation,
// not a reservation. Policy is mandatory even for routes the policy leaves unrationed.
type Admission struct {
	Allowed    bool      `json:"allowed"`
	Bucket     string    `json:"bucket"`
	Reason     string    `json:"reason"`
	Policy     string    `json:"policy"`
	ObservedAt time.Time `json:"observed_at"`
}

func (r *Runner) admission(ctx context.Context, c Candidate) string {
	if err := ctx.Err(); err != nil {
		return err.Error()
	}
	observation, err := r.Admit(ctx, c)
	if err != nil {
		return fmt.Sprintf("admission unavailable: %v", err)
	}
	if observation.Policy == "" || observation.ObservedAt.IsZero() {
		return "admission unavailable: incomplete observation"
	}
	if err := r.Record(Event{Kind: "admission", Admission: &observation, Attempt: &Attempt{Candidate: c}}); err != nil {
		return fmt.Sprintf("admission evidence: %v", err)
	}
	if !observation.Allowed {
		return "quota admission: " + observation.Reason
	}
	return ""
}

// Preflight checks candidate compatibility and admission without dispatching a role.
// Iterations call it before occupying a child execution slot; Run repeats the checks.
func (r *Runner) Preflight(ctx context.Context, req Request) error {
	if r.Admit == nil || r.Record == nil || r.Executors == nil {
		return fmt.Errorf("executor factory, admission policy and recorder required")
	}
	p, err := Resolve(req, r.Models)
	if err != nil {
		return err
	}
	var reasons []string
	for _, c := range p.Candidates {
		_, reason := r.preflight(ctx, c)
		if reason == "" {
			return nil
		}
		reasons = append(reasons, c.Model+": "+reason)
	}
	return fmt.Errorf("no available candidate: %s", strings.Join(reasons, "; "))
}
