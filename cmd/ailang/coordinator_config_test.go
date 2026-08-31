package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// M-MESSAGE-PLANE-FAIL-LOUD M4.
//
// The shared cloud coordinator config is written with a plain `gsutil cp`. There
// is no generation precondition anywhere in the repo (verified 2026-08-26: zero
// hits for if-generation-match / IfGenerationMatch), so writes are
// last-writer-wins.
//
// Demonstrated live the same day, not theorised: a correct edit uploaded at
// 14:24:33Z was clobbered at 14:37:10Z by a copy fetched before it, restoring a
// byte-identical earlier version. Neither side saw an error. Two machines
// editing prod config silently lose each other's work.

// fakeConfigStore is an in-memory stand-in with generation semantics.
type fakeConfigStore struct {
	data       []byte
	generation int64
	writes     int
}

func (f *fakeConfigStore) Read() ([]byte, int64, error) {
	return append([]byte(nil), f.data...), f.generation, nil
}

func (f *fakeConfigStore) Write(data []byte, ifGeneration int64) error {
	if ifGeneration != f.generation {
		return &staleGenerationError{Expected: ifGeneration, Actual: f.generation}
	}
	f.data = append([]byte(nil), data...)
	f.generation++
	f.writes++
	return nil
}

const validConfig = `coordinator:
  agents:
    - id: a
      inbox: a-inbox
      workspace: org/repo
`

func TestConfigCAS_WriteSucceedsOnCurrentGeneration(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 7}

	_, gen, err := store.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := writeConfigCAS(store, []byte(validConfig), gen); err != nil {
		t.Fatalf("a write against the current generation must succeed, got: %v", err)
	}
	if store.writes != 1 {
		t.Errorf("expected 1 write, got %d", store.writes)
	}
}

// The regression that matters: a write built on a stale read must be REFUSED,
// not silently applied.
func TestConfigCAS_RefusesStaleGeneration(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 7}

	_, myGen, _ := store.Read() // machine A reads gen 7

	// Machine B writes in between.
	if err := writeConfigCAS(store, []byte(validConfig), 7); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}

	// Machine A now writes against the generation it read.
	err := writeConfigCAS(store, []byte(validConfig), myGen)
	if err == nil {
		t.Fatal("a stale-generation write must be refused — this is the 14:24->14:37 lost update")
	}

	var stale *staleGenerationError
	if !errors.As(err, &stale) {
		t.Fatalf("expected a staleGenerationError, got %T: %v", err, err)
	}
	// The message must name BOTH generations so the operator can see what happened.
	msg := err.Error()
	if !strings.Contains(msg, "7") || !strings.Contains(msg, "8") {
		t.Errorf("error should name the expected and live generations; got: %v", msg)
	}
	if store.writes != 1 {
		t.Errorf("the refused write must not have been applied; writes=%d", store.writes)
	}
}

// Validation happens BEFORE the write: a malformed config must never reach the
// bucket, because every coordinator reads it on next cold start.
func TestConfigCAS_RejectsMalformedYAMLBeforeWriting(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 1}

	err := writeConfigCAS(store, []byte("coordinator:\n  agents:\n  - id: [unclosed\n"), 1)
	if err == nil {
		t.Fatal("malformed YAML must be refused")
	}
	if store.writes != 0 {
		t.Error("a malformed config must not be written")
	}
}

// Agent invariants are part of validation: a config that would strand an agent
// is refused for the same reason.
func TestConfigCAS_RejectsAgentMissingWorkspace(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 1}

	bad := `coordinator:
  agents:
    - id: a
      inbox: a-inbox
`
	err := writeConfigCAS(store, []byte(bad), 1)
	if err == nil {
		t.Fatal("an agent with no workspace must be refused")
	}
	if !strings.Contains(err.Error(), "workspace") {
		t.Errorf("error should name the invariant; got: %v", err)
	}
	if store.writes != 0 {
		t.Error("an invalid config must not be written")
	}
}

// Control: validation must accept a good config, or every test above would pass
// on a validator that rejects everything.
func TestConfigCAS_AcceptsValidConfig(t *testing.T) {
	if err := validateCoordinatorConfigBytes([]byte(validConfig)); err != nil {
		t.Fatalf("a valid config must pass validation, got: %v", err)
	}
}

func TestConfigCAS_AppliesProviderDefaultsBeforeRouteValidation(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 1}
	candidate := `coordinator:
  default_provider: codex
  agents:
    - id: planner
      inbox: planner
      workspace: org/repo
      executor_variant: codex
      model: model-a
`

	if err := writeConfigCAS(store, []byte(candidate), 1); err != nil {
		t.Fatalf("inherited codex provider should validate with codex variant: %v", err)
	}
	if store.writes != 1 {
		t.Fatalf("writes = %d, want 1", store.writes)
	}
}

func TestConfigCAS_RejectsProviderVariantMismatchBeforeWriting(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 1}
	candidate := `coordinator:
  default_provider: opencode
  agents:
    - id: planner
      inbox: planner
      workspace: org/repo
      executor_variant: codex
      model: model-a
`

	err := writeConfigCAS(store, []byte(candidate), 1)
	if err == nil {
		t.Fatal("provider/variant mismatch must be rejected")
	}
	var permanent *coordinator.PermanentDispatchError
	if !errors.As(err, &permanent) {
		t.Fatalf("error type = %T, want *coordinator.PermanentDispatchError: %v", err, err)
	}
	for _, want := range []string{"planner", "opencode", "codex"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if store.writes != 0 {
		t.Fatalf("invalid route must be rejected before GCS write; writes = %d", store.writes)
	}
}

func TestConfigCAS_LocalLaneIsExemptFromCloudRouteMatrix(t *testing.T) {
	store := &fakeConfigStore{data: []byte(validConfig), generation: 1}
	candidate := `coordinator:
  agents:
    - id: local-rig
      inbox: local-rig
      workspace: /srv/ailang
      execution_lane: local
      provider: ollama
      executor_variant: local-ollama
`

	if err := writeConfigCAS(store, []byte(candidate), 1); err != nil {
		t.Fatalf("local lane must be exempt from Cloud Run compatibility: %v", err)
	}
	if store.writes != 1 {
		t.Fatalf("writes = %d, want 1", store.writes)
	}
}
