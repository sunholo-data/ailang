package mission

import (
	"context"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"testing"
	"time"
)

func TestAdmissionUsesActualRouteAndObservations(t *testing.T) {
	now := time.Now()
	calls := map[string]int{}
	p := &AdmissionPolicy{Now: func() time.Time { return now }, Codex: func(time.Time) CodexQuotaObservation {
		calls["codex"]++
		return CodexQuotaObservation{State: "unknown"}
	}, Ollama: func(time.Time) OllamaQuotaObservation {
		calls["ollama"]++
		return OllamaQuotaObservation{State: "over"}
	}, Ledger: func(time.Time) (*Ledger, error) { calls["ledger"]++; return &Ledger{}, nil }}
	for _, c := range []dispatch.Candidate{{Executor: "codex"}, {Executor: "pi", WireModel: "ollama/deepseek:v4-cloud", Transport: "ollama"}} {
		got, err := p.Check(context.Background(), c)
		if err != nil || got.Allowed {
			t.Fatalf("blocked route admitted: %+v %v", got, err)
		}
	}
	if calls["codex"] != 1 || calls["ollama"] != 1 || calls["ledger"] != 0 {
		t.Fatal(calls)
	}
	got, err := p.Check(context.Background(), dispatch.Candidate{Executor: "pi", Transport: "openrouter"})
	if err != nil || !got.Allowed || got.Reason == "" {
		t.Fatalf("ledger policy changed: %+v %v", got, err)
	}
	p.Ledger = nil
	if _, err = p.Check(context.Background(), dispatch.Candidate{Transport: "openrouter"}); err == nil {
		t.Fatal("missing observer allowed")
	}
}
