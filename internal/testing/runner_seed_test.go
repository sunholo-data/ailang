package testing

import (
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

type Tree = { }

export pure func walk(t: Tree) -> int
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
