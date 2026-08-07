package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// seedAggFixtureSource is a passing module-bearing fixture. In FILE mode
// runTestsV2 is the aggregate; in PACKAGE mode runPackageTests is the aggregate.
// The declared module is the stable identity on both paths, so the fixture
// needs no module-path match (the package loader relaxes MOD010 for the
// on-disk mismatch in the package arm, and the file arm never runs `check`).
const seedAggFixtureSource = `module seedtest/pkg/prop_test

export func ok(x: int) -> bool ! {}
ensures { result == true } { true }
`

const seedAggManifest = `[package]
name = "seedtest/pkg"
version = "0.1.0"
edition = "1"

[exports]
modules = ["seedtest/pkg/prop"]

[stability]
level = "experimental"
`

// S17 — the AGGREGATE that the command path actually built must carry the seed
// metadata in BOTH modes. runTestsV2 and runPackageTests both os.Exit, so they
// cannot be called directly from a unit test; this test drives the built
// binary and asserts on the JSON the command path emitted, exactly as
// AC-SEED-AGG-M2C does. This is deliberately NOT a SetSeedMetadata-unit test
// (that is S14, which passes with both call sites deleted): deleting the
// SetSeedMetadata call in runPackageTests must make the package arm here fail.
func TestPackageAndFileAggregatesBothCarrySeedMetadata(t *testing.T) {
	bin := buildAilang(t)

	// FILE mode drives runTestsV2.
	file := filepath.Join(t.TempDir(), "prop_test.ail")
	if err := os.WriteFile(file, []byte(seedAggFixtureSource), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, exit := runAilangBin(t, bin, "test", "--format", "json", "--no-color", file)
	if exit != 0 {
		t.Fatalf("file mode exit = %d, want 0\nstderr:\n%s", exit, stderr)
	}
	assertSeedMetadataJSON(t, stdout, "file-mode aggregate (runTestsV2)")

	// PACKAGE mode drives runPackageTests.
	pkgDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pkgDir, "ailang.toml"), []byte(seedAggManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "prop_test.ail"), []byte(seedAggFixtureSource), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgOut, pkgErr, pkgExit := runAilangBin(t, bin, "test", "--package", "--format", "json", "--no-color", pkgDir)
	if pkgExit != 0 {
		t.Fatalf("package mode exit = %d, want 0\nstderr:\n%s", pkgExit, pkgErr)
	}
	assertSeedMetadataJSON(t, pkgOut, "package-mode aggregate (runPackageTests)")
}

// assertSeedMetadataJSON verifies the three top-level seed fields that the
// aggregate must stamp via SetSeedMetadata. An empty seed_mode / seed_derivation
// here is exactly the silent green-passing defect §5.5(b) and S17 guard against.
func assertSeedMetadataJSON(t *testing.T, raw, label string) {
	t.Helper()
	var output struct {
		SeedMode       string            `json:"seed_mode"`
		Seed           string            `json:"seed"`
		SeedDerivation string            `json:"seed_derivation"`
		Properties     []json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		t.Fatalf("%s: invalid JSON from aggregate: %v\n%s", label, err, raw)
	}
	if output.SeedMode != "derived" {
		t.Errorf("%s: seed_mode = %q, want %q", label, output.SeedMode, "derived")
	}
	if output.Seed != "0" {
		t.Errorf("%s: seed = %q, want %q", label, output.Seed, "0")
	}
	if output.SeedDerivation != "ailang-property-seed-v1" {
		t.Errorf("%s: seed_derivation = %q, want %q", label, output.SeedDerivation, "ailang-property-seed-v1")
	}
	if len(output.Properties) == 0 {
		t.Errorf("%s: aggregate emitted no properties", label)
	}
	var anySeed bool
	for _, p := range output.Properties {
		var prop struct {
			Seed *string `json:"seed"`
		}
		if err := json.Unmarshal(p, &prop); err != nil {
			t.Fatalf("%s: invalid property object: %v", label, err)
		}
		if prop.Seed != nil {
			anySeed = true
		}
	}
	if !anySeed {
		t.Errorf("%s: no property carried a per-property seed", label)
	}
}
