package testing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// seedFixtureSource is a single contract-bearing module with all three property
// classes: a forall, a requires, and an ensures. It is module-bearing so its
// identity is the declared module path and is independent of temp locations.
const seedFixtureSource = `module seed_fixture

export pure func g(x: int) -> int
  requires { x > 0 }
  ensures { result == x }
{
  x
}

property "commutative" {
  forall(x: int) => x + 0 == x
}

export func main() -> int ! {} { 0 }
`

// runSeedFixture writes the fixture source to a temp file and runs it through
// RunTestsFromFileWithConfig with the given config, returning the suite result.
func runSeedFixture(t *testing.T, cfg TestConfig) *SuiteResult {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "seed_fixture.ail")
	if err := os.WriteFile(tmpFile, []byte(seedFixtureSource), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	l := lexer.New(seedFixtureSource, tmpFile)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	result, err := RunTestsFromFileWithConfig(tmpFile, file, cfg)
	if err != nil {
		t.Fatalf("RunTestsFromFileWithConfig: %v", err)
	}
	return result
}

// seedFixtureSeeds returns the seeds for the three property paths of
// seedFixtureSource, keyed by status class (requires/ensures forall comes first).
func seedFixtureSeeds(t *testing.T, result *SuiteResult) []int64 {
	t.Helper()
	var seeds []int64
	for _, p := range result.Properties {
		if p.Seed == 0 {
			t.Fatalf("property %q had zero seed", p.Name)
		}
		seeds = append(seeds, p.Seed)
	}
	return seeds
}

func testConfig(root string, master int64) TestConfig {
	return TestConfig{
		WorkspaceRoot: root,
		SeedMode:      SeedModeDerived,
		MasterSeed:    master,
	}
}

// S10 — TestNewRNG_HasNoWallClockBranch kills leaving the `seed == 0` sentinel
// in place. With a wall clock, two newRNG(0) calls would almost surely differ.
func TestNewRNG_HasNoWallClockBranch(t *testing.T) {
	a := newRNG(0).Int63()
	b := newRNG(0).Int63()
	if a != b {
		t.Fatalf("zero-seed generation is nondeterministic: first Int63() %d vs %d (wall clock still present)", a, b)
	}
	// Sanity: the seed actually matters.
	c := newRNG(1).Int63()
	if c == a {
		t.Fatalf("seed 1 produced the same first Int63() as seed 0: %d", c)
	}
}

// S11 — TestRunner_AllThreePropertyPathsUseDerivedSeed kills guarding two sites
// and missing contract_domain.go's ensures path. Run twice through
// RunTestsFromFileWithConfig; every property seed is non-zero, the three differ
// from each other, and both runs agree field-for-field.
func TestRunner_AllThreePropertyPathsUseDerivedSeed(t *testing.T) {
	cfg := testConfig(t.TempDir(), 0)

	first := runSeedFixture(t, cfg)
	second := runSeedFixture(t, cfg)

	firstSeeds := seedFixtureSeeds(t, first)
	if len(firstSeeds) != 3 {
		t.Fatalf("expected 3 property results, got %d", len(firstSeeds))
	}

	// The three property streams must differ from one another.
	if firstSeeds[0] == firstSeeds[1] || firstSeeds[0] == firstSeeds[2] || firstSeeds[1] == firstSeeds[2] {
		t.Fatalf("all three derived seeds should differ, got %v", firstSeeds)
	}

	// Both runs must agree seed-for-seed.
	secondSeeds := seedFixtureSeeds(t, second)
	for i := range firstSeeds {
		if firstSeeds[i] != secondSeeds[i] {
			t.Fatalf("run %d: seed %d != %d across runs (not deterministic)", i, firstSeeds[i], secondSeeds[i])
		}
	}
}

// streamObservables reduces a suite result to the per-property facts that are a
// function of the RNG *stream* rather than of the seed field: how many inputs
// were generated and discarded, and the counterexample text. PropertyResult.Seed
// is deliberately excluded — it is stamped in each path's initializer and is
// therefore identical whether or not the stream actually consumed it.
func streamObservables(t *testing.T, result *SuiteResult) map[string]string {
	t.Helper()
	obs := make(map[string]string, len(result.Properties))
	for _, p := range result.Properties {
		obs[p.Name] = fmt.Sprintf("status=%s testsRun=%d generated=%d discarded=%d err=%s",
			p.Status, p.TestsRun, p.GeneratedInputs, p.DiscardedInputs, p.Error)
	}
	return obs
}

// S11b — TestRunner_DerivedSeedDrivesSampleStreams is the arm S11 cannot be.
//
// S11 observes PropertyResult.Seed, which each path stamps into its result
// initializer *independently of what it hands to newRNG*. So S11 stays green
// under the exact defect it is documented to kill: replacing
// `newRNG(r.propertySeed(...))` with `newRNG(<constant>)` at contract_domain.go's
// ensures site leaves every seed field untouched. Verified by mutation at
// mission iteration 162 — the mutant built and the whole seed suite passed.
//
// This test instead observes facts downstream of the generator stream:
//   - the ensures path's accept/discard counts under M1's domain filter, and
//   - the requires path's counterexample text,
//
// each of which changes if and only if that site's RNG actually consumed the
// derived seed. Both are exact and reproducible, not statistical: the seeds are
// fixed constants, so the sampled values are fixed too.
//
// The forall site (runner.go's runProperty) has no stream observable in this
// package today — every forall property in a unit-test fixture fails on its
// first generated input with "evaluation failed: empty program", a pre-existing
// harness limitation unrelated to this milestone. That site is pinned by the
// seed stamp plus AC-SEED-SWEEP-M2 arm (c), and is called out here so a future
// reader does not mistake its absence for coverage.
func TestRunner_DerivedSeedDrivesSampleStreams(t *testing.T) {
	const (
		requiresProp = "g_property_1" // the requires path (runner.go)
		ensuresProp  = "g_property_2" // the ensures path (contract_domain.go)
	)

	master0a := streamObservables(t, runSeedFixture(t, testConfig(t.TempDir(), 0)))
	master0b := streamObservables(t, runSeedFixture(t, testConfig(t.TempDir(), 0)))
	master42 := streamObservables(t, runSeedFixture(t, testConfig(t.TempDir(), 42)))

	// Non-vacuity: an empty or renamed property set must fail loudly rather than
	// leave every assertion below with nothing to compare.
	for _, name := range []string{requiresProp, ensuresProp} {
		if _, ok := master0a[name]; !ok {
			t.Fatalf("instrument failure: fixture produced no property %q (have %v)", name, master0a)
		}
	}

	// Determinism: the same master must reproduce the stream exactly, from a
	// different workspace root, in a different temp directory.
	for name, want := range master0a {
		if got := master0b[name]; got != want {
			t.Fatalf("property %q is not reproducible at the same master:\n  run A: %s\n  run B: %s", name, want, got)
		}
	}

	// Sensitivity, per site. Each of these fails if that site's newRNG stops
	// consuming the derived seed.
	if master42[ensuresProp] == master0a[ensuresProp] {
		t.Fatalf("ensures path (contract_domain.go) ignored the master seed: master 0 and 42 both gave %s",
			master0a[ensuresProp])
	}
	if master42[requiresProp] == master0a[requiresProp] {
		t.Fatalf("requires path (runner.go) ignored the master seed: master 0 and 42 both gave %s",
			master0a[requiresProp])
	}
}

// S12 — TestRunner_MasterSeedChangesEveryStream kills a hard-coded or ignored
// master: at master 0 vs 42 all three seed values must differ.
func TestRunner_MasterSeedChangesEveryStream(t *testing.T) {
	base := testConfig(t.TempDir(), 0)
	altered := testConfig(t.TempDir(), 42)

	baseSeeds := seedFixtureSeeds(t, runSeedFixture(t, base))
	alteredSeeds := seedFixtureSeeds(t, runSeedFixture(t, altered))

	for i := 0; i < 3; i++ {
		if baseSeeds[i] == alteredSeeds[i] {
			t.Fatalf("property %d: master 0 and master 42 produced the same seed %d (master ignored)", i, baseSeeds[i])
		}
	}
}

// S13 — TestRunProperty_SeedSetEvenOnSkip kills setting the seed after an early
// return: a no_generator property must still carry a non-zero Seed.
func TestRunProperty_SeedSetEvenOnSkip(t *testing.T) {
	src := `module skip_seed

export pure func walk(t: ImportedTree) -> int
  ensures { result >= 0 }
{
  0
}

export func main() -> int ! {} { 0 }
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "skip_seed.ail")
	if err := os.WriteFile(tmpFile, []byte(src), 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	l := lexer.New(src, tmpFile)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	cfg := TestConfig{
		WorkspaceRoot: tmpDir,
		SeedMode:      SeedModeDerived,
		MasterSeed:    0,
	}
	result, err := RunTestsFromFileWithConfig(tmpFile, file, cfg)
	if err != nil {
		t.Fatalf("RunTestsFromFileWithConfig: %v", err)
	}

	// Non-vacuity: with no property results the loop below asserts nothing and
	// this test passes for the wrong reason. Verified non-vacuous at iteration
	// 162 by adding this guard and observing the test still pass.
	if len(result.Properties) == 0 {
		t.Fatal("instrument failure: fixture produced zero property results")
	}

	for _, prop := range result.Properties {
		if prop.Status != StatusSkip {
			t.Fatalf("expected a skip, got %s (%s)", prop.Status, prop.Error)
		}
		if prop.Seed == 0 {
			t.Fatalf("property %q skipped but had zero seed (seed set too late)", prop.Name)
		}
	}
}

// S14 — TestSuiteResult_SetSeedMetadata kills a partial copy: all three fields
// must be copied from a TestConfig.
func TestSuiteResult_SetSeedMetadata(t *testing.T) {
	cfg := TestConfig{
		WorkspaceRoot: "/work",
		SeedMode:      SeedModeMaster,
		MasterSeed:    -987654321,
	}
	sr := NewSuiteResult("some/module")
	sr.SetSeedMetadata(cfg)

	if sr.SeedMode != SeedModeMaster {
		t.Fatalf("SeedMode = %q, want %q", sr.SeedMode, SeedModeMaster)
	}
	if sr.MasterSeed != -987654321 {
		t.Fatalf("MasterSeed = %d, want -987654321", sr.MasterSeed)
	}
	if sr.SeedDerivation != SeedDerivationV1 {
		t.Fatalf("SeedDerivation = %q, want %q", sr.SeedDerivation, SeedDerivationV1)
	}
}
