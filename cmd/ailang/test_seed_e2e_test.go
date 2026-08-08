package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedE2EMixSource has one passing ensures property and one failing ensures
// property. The declared module gives both a stable, path-independent identity
// (ResolveModuleIdentity), so the derived seeds are stable across t.TempDir
// locations — which is what makes two runs byte-comparable.
const seedE2EMixSource = `module seed_e2e_mix

export func good(x: int) -> bool ! {}
ensures { result == true } { x == x }

export func bad(x: int) -> bool ! {}
ensures { result == true } { false }
`

// seedE2EFailFirstSource has the FAILING ensures property declared first, so
// .properties[0] carries the replay key (source order is preserved).
const seedE2EFailFirstSource = `module seed_e2e_fail_first

export func bad(x: int) -> bool ! {}
ensures { result == true } { false }

export func good(x: int) -> bool ! {}
ensures { result == true } { x == x }
`

// seedE2EPkgManifest is the package manifest for the package-mode replay
// fixture. Package mode is a SECOND ReplayTarget branch in cmd/ailang/main.go
// (it prefixes "--package "), and every other test in this file drives file
// mode only.
const seedE2EPkgManifest = `[package]
name = "seede2e/pkg"
version = "0.1.0"
edition = "1"

[exports]
modules = ["seede2e/pkg/fail"]

[stability]
level = "experimental"
`

const seedE2EPkgFailSource = `module seede2e/pkg/fail_test

export func bad(x: int) -> bool ! {}
ensures { result == true } { false }
`

// writeSeedE2EFixture writes src to a fresh t.TempDir file and returns it.
func writeSeedE2EFixture(t *testing.T, name, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// normalizeRunJSON strips the two timing fields that legitimately vary between
// identical runs (the suite total and every per-test / per-property duration)
// and re-marshals deterministically. It is the same field-deletion set AC6-M2
// uses, so byte-identity after normalization is a real determinism assertion
// over every remaining field (seed, seed_derivation, per-property seed, verdicts).
func normalizeRunJSON(t *testing.T, raw []byte) []byte {
	t.Helper()
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("run output is not JSON: %v\n%s", err, raw)
	}
	delete(doc, "total_duration")
	for _, key := range []string{"tests", "properties"} {
		list, _ := doc[key].([]interface{})
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				delete(m, "duration")
			}
		}
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("re-marshal normalized JSON: %v", err)
	}
	return out
}

// TestReplayTargetArg pins the shell-quoting rules the CLI applies when
// recording the replay target (T2 §3). A plain path is bare; any space, single
// quote, or double quote forces single-quote wrapping with embedded single
// quotes escaped as '\”. The emitted command must re-tokenize to the original
// path in a shell, which is what makes the replay command runnable.
func TestReplayTargetArg(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "folder/a.ail", want: "folder/a.ail"},
		{in: "path with space/a.ail", want: "'path with space/a.ail'"},
		{in: "it's/a.ail", want: "'it'\\''s/a.ail'"},
		{in: `quo"te/a.ail`, want: "'quo\"te/a.ail'"},
		{in: "sp ace and 'quote/f.ail", want: "'sp ace and '\\''quote/f.ail'"},
	}
	for _, c := range cases {
		if got := replayTargetArg(c.in); got != c.want {
			t.Errorf("replayTargetArg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSeedE2E_DefaultRunIsDeterministic — AC5-M2: two default runs over the
// same fixture produce byte-identical normalized JSON, and the default mode is
// reported as derived with the v1 derivation tag.
func TestSeedE2E_DefaultRunIsDeterministic(t *testing.T) {
	bin := buildAilang(t)
	file := writeSeedE2EFixture(t, "mix.ail", seedE2EMixSource)

	raw1, _, exit1 := runAilangBin(t, bin, "test", "--format", "json", "--no-color", file)
	raw2, _, exit2 := runAilangBin(t, bin, "test", "--format", "json", "--no-color", file)
	if exit1 != 1 || exit2 != 1 {
		t.Fatalf("expected both runs to exit 1 (one failing property), got %d then %d", exit1, exit2)
	}

	a := normalizeRunJSON(t, []byte(raw1))
	b := normalizeRunJSON(t, []byte(raw2))
	if string(a) != string(b) {
		t.Fatalf("default runs are NOT byte-identical after normalization:\n---run 1---\n%s\n---run 2---\n%s", a, b)
	}

	var doc struct {
		SeedMode       string `json:"seed_mode"`
		SeedDerivation string `json:"seed_derivation"`
	}
	if err := json.Unmarshal([]byte(raw1), &doc); err != nil {
		t.Fatalf("parse run 1: %v", err)
	}
	if doc.SeedMode != "derived" {
		t.Errorf("seed_mode = %q, want %q", doc.SeedMode, "derived")
	}
	if doc.SeedDerivation != "ailang-property-seed-v1" {
		t.Errorf("seed_derivation = %q, want %q", doc.SeedDerivation, "ailang-property-seed-v1")
	}
}

// TestSeedE2E_RandomThenExplicitReplaysByteIdentical — the AC6-M2 shape in Go:
// a --random-seed run reports a master seed, and re-running with --seed=<that
// value> reproduces the run byte-for-byte (exit codes included), with the
// replay run reporting seed_mode master.
func TestSeedE2E_RandomThenExplicitReplaysByteIdentical(t *testing.T) {
	bin := buildAilang(t)
	file := writeSeedE2EFixture(t, "mix.ail", seedE2EMixSource)

	randomOut, _, randomExit := runAilangBin(t, bin, "test", "--random-seed", "--format", "json", "--no-color", file)
	var randomDoc struct {
		Seed     string `json:"seed"`
		SeedMode string `json:"seed_mode"`
	}
	if err := json.Unmarshal([]byte(randomOut), &randomDoc); err != nil {
		t.Fatalf("random run output is not JSON: %v\n%s", err, randomOut)
	}
	if randomDoc.Seed == "" {
		t.Fatalf("--random-seed run reported no seed")
	}
	if randomDoc.SeedMode != "master" {
		t.Errorf("random run seed_mode = %q, want %q", randomDoc.SeedMode, "master")
	}

	// Replay with the reported seed as --seed=<value>.
	replayOut, _, replayExit := runAilangBin(t, bin, "test", "--seed="+randomDoc.Seed, "--format", "json", "--no-color", file)
	if replayExit != randomExit {
		t.Errorf("replay exit = %d, random exit = %d — exit codes differ", replayExit, randomExit)
	}

	a := normalizeRunJSON(t, []byte(randomOut))
	b := normalizeRunJSON(t, []byte(replayOut))
	if string(a) != string(b) {
		t.Errorf("random run and --seed replay are NOT byte-identical after normalization:\n---random---\n%s\n---replay---\n%s\n", a, b)
	}

	var replayDoc struct {
		SeedMode string `json:"seed_mode"`
	}
	if err := json.Unmarshal([]byte(replayOut), &replayDoc); err != nil {
		t.Fatalf("parse replay output: %v", err)
	}
	if replayDoc.SeedMode != "master" {
		t.Errorf("replay seed_mode = %q, want %q (explicit seed means master mode)", replayDoc.SeedMode, "master")
	}

	// The reported master must actually DRIVE the sample stream — the seed is
	// derived, not a constant. Two distinct explicit masters must produce
	// different SAMPLES. The reported per-property `seed` field is derived from
	// the master independently of the RNG consumption, so it differs between
	// masters even under a constant-seed mutant and cannot discriminate; the
	// sample-driven fields (the failing input embedded in `error`) can. M-b of
	// the M3 drill forces the ensures RNG to a constant, which collapses the
	// sample output for two distinct masters to identical — caught here.
	normalizeNoSeed := func(raw []byte) []byte {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("run output is not JSON: %v", err)
		}
		delete(doc, "total_duration")
		delete(doc, "seed")
		if props, ok := doc["properties"].([]interface{}); ok {
			for _, item := range props {
				if m, ok := item.(map[string]interface{}); ok {
					delete(m, "duration")
					delete(m, "seed")
					delete(m, "replay") // embeds the master seed; would mask a constant RNG
				}
			}
		}
		out, _ := json.MarshalIndent(doc, "", "  ")
		return out
	}
	seedA, _, exitA := runAilangBin(t, bin, "test", "--seed=100", "--format", "json", "--no-color", file)
	seedB, _, exitB := runAilangBin(t, bin, "test", "--seed=200", "--format", "json", "--no-color", file)
	if exitA != exitB {
		t.Errorf("distinct seeds exited differently (%d vs %d)", exitA, exitB)
	}
	nA := normalizeNoSeed([]byte(seedA))
	nB := normalizeNoSeed([]byte(seedB))
	if string(nA) == string(nB) {
		t.Errorf("two distinct explicit seeds produced byte-identical SAMPLE output — the master seed does not drive the sample stream")
	}
}

// TestSeedE2E_HelpNamesBothFlags — `test --help` advertises both new flags.
// Help currently goes to stdout, but we check both streams so the assertion
// survives a future move.
func TestSeedE2E_HelpNamesBothFlags(t *testing.T) {
	bin := buildAilang(t)
	stdout, stderr, _ := runAilangBin(t, bin, "test", "--help")
	blob := stdout + stderr
	for _, literal := range []string{"--seed", "--random-seed"} {
		if !strings.Contains(blob, literal) {
			t.Errorf("test --help output missing %q\nstdout:\n%s\nstderr:\n%s", literal, stdout, stderr)
		}
	}
}

// TestSeedE2E_EmittedReplayCommandActuallyReplays — THE pin for the §3(A)
// defect. AC6-M2 misses it because it builds --seed=${seed} "$tmp/multi.ail"
// from its own inputs rather than executing the exact string the tool emitted.
// Here we read .properties[0].replay straight out of the JSON and RUN THAT
// TEXT back through the binary; a test that rebuilt the command from .seed
// instead would not close the defect and is rejected on review.
func TestSeedE2E_EmittedReplayCommandActuallyReplays(t *testing.T) {
	bin := buildAilang(t)
	file := writeSeedE2EFixture(t, "fail_first.ail", seedE2EFailFirstSource)

	// Run with an explicit seed so the replay run is also master-mode (a
	// derived initial run would emit "--seed 0" and differ only in seed_mode,
	// which is outside the AC6-M2 duration-deletion normalisation set).
	firstOut, firstErr, firstExit := runAilangBin(t, bin, "test", "--seed", "42", "--format", "json", "--no-color", file)
	if firstExit != 1 {
		t.Fatalf("expected failing fixture to exit 1, got %d\nstderr:\n%s", firstExit, firstErr)
	}
	var firstDoc struct {
		Properties []struct {
			Replay string `json:"replay"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(firstOut), &firstDoc); err != nil {
		t.Fatalf("first run output is not JSON: %v\n%s", err, firstOut)
	}
	if len(firstDoc.Properties) == 0 || firstDoc.Properties[0].Replay == "" {
		t.Fatalf("properties[0] did not emit a replay command\n%s", firstOut)
	}
	replay := firstDoc.Properties[0].Replay
	if strings.Contains(replay, "All Tests") {
		t.Fatalf("emitted replay contains the un-runnable %q label: %q", "All Tests", replay)
	}

	// Execute the EMITTED command text: split it, drop the leading `ailang`
	// token, and hand the remaining args back to the binary. The JSON/colour
	// flags are INSERTED after the `test` subcommand (not appended after the
	// positional path) because Go's flag parser stops at the first non-flag
	// argument — a flag placed after the path is silently ignored and the run
	// would fall back to human output. The emitted seed AND path tokens are
	// passed through verbatim; only the two output-format flags are added.
	fields := strings.Fields(replay)
	if len(fields) < 3 || fields[0] != "ailang" || fields[1] != "test" {
		t.Fatalf("unexpected replay command shape: %q", replay)
	}
	replayArgs := []string{"test", "--format", "json", "--no-color"}
	replayArgs = append(replayArgs, fields[2:]...)
	replayOut, replayErr, replayExit := runAilangBin(t, bin, replayArgs...)
	if replayExit != firstExit {
		t.Errorf("replayed exit = %d, first exit = %d\nreplay stderr:\n%s", replayExit, firstExit, replayErr)
	}
	_ = replayErr

	nFirst := normalizeRunJSON(t, []byte(firstOut))
	nReplay := normalizeRunJSON(t, []byte(replayOut))
	if string(nFirst) != string(nReplay) {
		t.Errorf("emitted replay command did NOT reproduce the run byte-identically:\n---first---\n%s\n---replay (%q)---\n%s", nFirst, replay, nReplay)
	}
}

// TestSeedE2E_PackageModeEmittedReplayCarriesPackageFlag closes a gap the
// iteration-166 evaluator found by mutation, and it is the same defect class as
// the "All Tests" bug this milestone exists to fix — one branch of an emitted
// artifact guarded, the sibling branch not.
//
// cmd/ailang/main.go builds ReplayTarget on TWO branches: package mode prefixes
// "--package ", ordinary mode does not. Every other test here drives file mode,
// and the pre-existing package-mode test
// (TestPackageAndFileAggregatesBothCarrySeedMetadata) uses a PASSING fixture —
// and `replay` is emitted only for FAILING properties. So the package branch had
// no observer at all: dropping the `"--package " +` prefix left
// `go test ./cmd/ailang/... ./internal/testing/...` fully rc=0 (reproduced
// first-party before this test was written). This test uses a failing fixture so
// the key is emitted, asserts the flag survives, and then RUNS the emitted text.
func TestSeedE2E_PackageModeEmittedReplayCarriesPackageFlag(t *testing.T) {
	bin := buildAilang(t)

	pkgDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkgDir, "ailang.toml"), []byte(seedE2EPkgManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "fail_test.ail"), []byte(seedE2EPkgFailSource), 0o644); err != nil {
		t.Fatal(err)
	}

	firstOut, firstErr, firstExit := runAilangBin(t, bin,
		"test", "--package", "--seed", "42", "--format", "json", "--no-color", pkgDir)
	if firstExit != 1 {
		t.Fatalf("expected failing package fixture to exit 1, got %d\nstdout:\n%s\nstderr:\n%s", firstExit, firstOut, firstErr)
	}
	var firstDoc struct {
		Properties []struct {
			Replay string `json:"replay"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(firstOut), &firstDoc); err != nil {
		t.Fatalf("package run output is not JSON: %v\n%s", err, firstOut)
	}
	if len(firstDoc.Properties) == 0 || firstDoc.Properties[0].Replay == "" {
		t.Fatalf("package mode emitted no replay command for a failing property\n%s", firstOut)
	}
	replay := firstDoc.Properties[0].Replay

	// The discriminating assertion: without the prefix the emitted command
	// re-runs the directory in FILE mode, which is a different discovery path.
	if !strings.Contains(replay, "--package") {
		t.Fatalf("package-mode replay lost the --package flag: %q", replay)
	}
	if strings.Contains(replay, "All Tests") {
		t.Fatalf("emitted replay contains the un-runnable %q label: %q", "All Tests", replay)
	}

	// Execute the emitted text verbatim (flags inserted after the subcommand —
	// see the note in TestSeedE2E_EmittedReplayCommandActuallyReplays). The
	// t.TempDir path has no spaces, so Fields-splitting is faithful here;
	// replayTargetArg's quoting is covered directly by TestReplayTargetArg.
	fields := strings.Fields(replay)
	if len(fields) < 3 || fields[0] != "ailang" || fields[1] != "test" {
		t.Fatalf("unexpected replay command shape: %q", replay)
	}
	replayArgs := append([]string{"test", "--format", "json", "--no-color"}, fields[2:]...)
	replayOut, replayErr, replayExit := runAilangBin(t, bin, replayArgs...)
	if replayExit != firstExit {
		t.Errorf("replayed exit = %d, first exit = %d\nreplay stderr:\n%s", replayExit, firstExit, replayErr)
	}
	if got, want := string(normalizeRunJSON(t, []byte(replayOut))), string(normalizeRunJSON(t, []byte(firstOut))); got != want {
		t.Errorf("package-mode replay %q did NOT reproduce the run byte-identically:\n---first---\n%s\n---replay---\n%s", replay, want, got)
	}
}
