//go:build !js

package effects

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func spawnArgs(cmd string) []eval.Value {
	return []eval.Value{&eval.StringValue{Value: cmd}, &eval.ListValue{}}
}

func TestProcessSpawnLegacyDenials(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		pc         *ProcessContext
	}{
		{"missing", "_process_spawn_process: command not found: absent", &ProcessContext{HasAllowlist: true, Allowlist: map[string]string{"absent": ""}}},
		{"denied", "_process_spawn_process: command not allowed: absent", &ProcessContext{HasAllowlist: true}},
		{"confined", "_process_spawn_process: command not allowed: absent (spawnProcess is not available under confined Process; use exec)", &ProcessContext{Confined: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewEffContext(nil)
			ctx.Process = tc.pc
			v, err := ProcessSpawn(ctx, spawnArgs("absent"))
			if v != nil || err == nil || err.Error() != tc.want {
				t.Fatalf("got %v, %v; want %s", v, err, tc.want)
			}
		})
	}
}

func TestProcessSpawnLegacyAdmission(t *testing.T) {
	ctx, _, _ := managedSleepFixture(t)
	owner := ctx.executionWorkerOwner()
	owner.mu.Lock()
	owner.closing = true
	owner.mu.Unlock()
	// Reuse the already resolved executable rather than rely on PATH lookup.
	ctx.Process.HasAllowlist = true
	ctx.Process.Allowlist = map[string]string{"worker": "unused"}
	_, err := ProcessSpawn(ctx, spawnArgs("worker"))
	if err == nil || err.Error() != "execution owner is shutting down" || strings.Contains(err.Error(), "_process_spawn") {
		t.Fatalf("legacy admission: %v", err)
	}
}
