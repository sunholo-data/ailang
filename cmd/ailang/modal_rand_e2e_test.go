package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/replay"
)

func runModalRandEntry(t *testing.T, bin, entry string, seed *string, emitTrace bool) []byte {
	t.Helper()

	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}

	args := []string{"run", "--caps", "Rand,IO", "--entry", entry, "--quiet"}
	if emitTrace {
		args = append(args, "--emit-trace", "jsonl")
	}
	args = append(args, filepath.Join("examples", "modal_rand.ail"))
	cmd := exec.Command(bin, args...)
	cmd.Dir = projectRoot
	cmd.Env = append([]string{}, os.Environ()...)
	cmd.Env = removeEnv(cmd.Env, "AILANG_SEED")
	if seed != nil {
		cmd.Env = append(cmd.Env, "AILANG_SEED="+*seed)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("%s failed: %v\nstdout:\n%s\nstderr:\n%s", entry, err, stdout.String(), stderr.String())
	}
	return stdout.Bytes()
}

func removeEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := env[:0]
	for _, item := range env {
		if !bytes.HasPrefix([]byte(item), []byte(prefix)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func TestModalRandEntrypoints(t *testing.T) {
	bin := buildAilang(t)
	seedA := "424242"
	seedB := "424243"

	first := runModalRandEntry(t, bin, "main_seeded", &seedA, false)
	second := runModalRandEntry(t, bin, "main_seeded", &seedA, false)
	if !bytes.Equal(first, second) {
		t.Fatalf("same AILANG_SEED produced different output:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	different := runModalRandEntry(t, bin, "main_seeded", &seedB, false)
	if bytes.Equal(first, different) {
		t.Fatalf("different AILANG_SEED values produced identical output:\n%s", first)
	}

	runModalRandEntry(t, bin, "main_crypto", nil, false)

	// os-mode is the entrypoint tools/verify_examples.sh exercises (it hardcodes
	// --entry main), so keep it covered at runtime here too: a break in the os
	// dispatch path must fail this test, not only the example gate.
	osTrace := runModalRandEntry(t, bin, "main", nil, true)
	if !bytes.Contains(osTrace, []byte(`"mode":"os","contract":"re-sampleable"`)) {
		t.Fatalf("os trace missing re-sampleable replay coverage:\n%s", osTrace)
	}

	seededTrace := runModalRandEntry(t, bin, "main_seeded", &seedA, true)
	if !bytes.Contains(seededTrace, []byte(`"mode":"seeded","contract":"deterministic"`)) {
		t.Fatalf("seeded trace missing deterministic replay coverage:\n%s", seededTrace)
	}
	cryptoTrace := runModalRandEntry(t, bin, "main_crypto", nil, true)
	if !bytes.Contains(cryptoTrace, []byte(`"mode":"crypto","contract":"opaque"`)) {
		t.Fatalf("crypto trace missing opaque replay coverage:\n%s", cryptoTrace)
	}

	if got, ok := replay.ContractFor("Rand", "os"); !ok || got != replay.ReSampleable {
		t.Fatalf("Rand[mode=os] contract = (%q, %v), want (%q, true)", got, ok, replay.ReSampleable)
	}
	if got, ok := replay.ContractFor("Rand", "seeded"); !ok || got != replay.Deterministic {
		t.Fatalf("Rand[mode=seeded] contract = (%q, %v), want (%q, true)", got, ok, replay.Deterministic)
	}
	if got, ok := replay.ContractFor("Rand", "crypto"); !ok || got != replay.Opaque {
		t.Fatalf("Rand[mode=crypto] contract = (%q, %v), want (%q, true)", got, ok, replay.Opaque)
	}
}
