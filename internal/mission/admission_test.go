package mission

import (
	"context"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
	"strings"
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

func TestQuotaAdmissionTransport(t *testing.T) {
	for _, bucket := range []string{"opencode", "unknown", "anthropic", "openrouter"} {
		t.Run(bucket, func(t *testing.T) {
			calls := 0
			p := &AdmissionPolicy{
				Now: func() time.Time { return opencodeQuotaTime },
				Codex: func(time.Time) CodexQuotaObservation {
					t.Fatal("unexpected Codex probe")
					return CodexQuotaObservation{}
				},
				Ollama: func(time.Time) OllamaQuotaObservation {
					t.Fatal("unexpected Ollama probe")
					return OllamaQuotaObservation{}
				},
				Ledger: func(time.Time) (*Ledger, error) { calls++; return &Ledger{}, nil },
			}
			got, err := p.Check(context.Background(), dispatch.Candidate{Executor: "pi", Transport: bucket, WireModel: "plain-model"})
			if bucket == "anthropic" || bucket == "openrouter" {
				if err != nil || !got.Allowed || got.Bucket != bucket || calls != 1 {
					t.Fatalf("backend admission: %+v err=%v ledger calls=%d", got, err, calls)
				}
			} else if err == nil || !strings.Contains(err.Error(), "unsupported quota bucket") || calls != 0 {
				t.Fatalf("transport must be unsupported before ledger probe: admission=%+v err=%v calls=%d", got, err, calls)
			}
		})
	}
}
