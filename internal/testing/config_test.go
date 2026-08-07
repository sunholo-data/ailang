package testing

import (
	"bytes"
	"io"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

// S1 — golden vectors pin the v1 derivation. These literals were produced by
// running DeriveSeedV1; any change to framing, hash, or byte order reds this test.
func TestDeriveSeedV1_GoldenVectors(t *testing.T) {
	cases := []struct {
		master   int64
		identity string
		name     string
		want     int64
	}{
		{master: 0, identity: "m", name: "p", want: 4489511486305051779},
		{master: 42, identity: "a/b/c", name: "q", want: 38716288277485395},
		{master: -7, identity: "x", name: "z", want: -1903907135015076133},
	}
	for _, c := range cases {
		got := DeriveSeedV1(c.master, c.identity, c.name)
		if got != c.want {
			t.Errorf("DeriveSeedV1(%d,%q,%q) = %d, want %d", c.master, c.identity, c.name, got, c.want)
		}
	}
}

func checkSeedVectorsDiffer(t *testing.T, got ...int64) {
	t.Helper()
	for i := 0; i < len(got); i++ {
		for j := i + 1; j < len(got); j++ {
			if got[i] == got[j] {
				t.Errorf("expected seeds %d and %d to differ, both were %d", i, j, got[i])
			}
		}
	}
}

// S2 — master 0, -1, and MinInt64 all derive without panic and differ.
func TestDeriveSeedV1_ZeroAndNegativeMaster(t *testing.T) {
	a := DeriveSeedV1(0, "m", "p")
	b := DeriveSeedV1(-1, "m", "p")
	c := DeriveSeedV1(math.MinInt64, "m", "p")
	checkSeedVectorsDiffer(t, a, b, c)
}

// S3 — the identity and property fields are \x00-framed: different splits must
// not collide.
func TestDeriveSeedV1_IdentityFieldsAreFramed(t *testing.T) {
	a := DeriveSeedV1(0, "ab", "c")
	b := DeriveSeedV1(0, "a", "bc")
	if a == b {
		t.Errorf("expected derive(0,\"ab\",\"c\") != derive(0,\"a\",\"bc\"); both were %d", a)
	}
}

// S4 — a declared module wins verbatim and workspaceRoot is ignored.
func TestResolveModuleIdentity_DeclaredModuleWins(t *testing.T) {
	got, err := ResolveModuleIdentity("/ignored/root", "some/input.ail", "exact/module/name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "exact/module/name" {
		t.Errorf("got %q, want %q", got, "exact/module/name")
	}
}

// S5 — a moduleless input is workspace-relative, and a relative input resolves
// identically via filepath.Abs semantics.
func TestResolveModuleIdentity_ModulelessIsWorkspaceRelative(t *testing.T) {
	// t.TempDir(), not a "/w" literal: filepath.IsAbs("/w") is FALSE on Windows
	// (an absolute path there needs a volume), so a POSIX literal makes this row
	// measure the host OS rather than the derivation.
	root := t.TempDir()
	got, err := ResolveModuleIdentity(root, filepath.Join(root, "cases", "s.ail"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "cases/s.ail" {
		t.Errorf("got %q, want %q", got, "cases/s.ail")
	}

	// Relative input: filepath.Abs resolves it against the process cwd. We
	// derive the workspace root from that same Abs call so no os.Chdir is
	// needed, and assert the result matches the absolute-input form above.
	// Two Dir calls, not one: Dir(abs) is the `cases/` directory, and a root
	// of `cases/` would make the identity "s.ail" — losing the very path
	// component this row exists to pin.
	abs, err := filepath.Abs("cases/s.ail")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	cwdRoot := filepath.Dir(filepath.Dir(abs))
	gotRel, err := ResolveModuleIdentity(cwdRoot, "cases/s.ail", "")
	if err != nil {
		t.Fatalf("unexpected error for relative input: %v", err)
	}
	if gotRel != "cases/s.ail" {
		t.Errorf("relative input got %q, want %q", gotRel, "cases/s.ail")
	}
}

// S6 — a root the input cannot be made relative to returns a non-nil error and
// an empty string (no silent "" and no basename fallback).
//
// The plan's §5.6 S6 row proposed "root /w, abs input on another volume prefix".
// That trigger does not exist on POSIX: filepath.Rel relates ANY two absolute
// paths by walking up, so Rel(root, sibling) returns a "../"-prefixed path with
// a nil error. Since D3 makes the input absolute before the Rel call, the one
// reachable error shape on every platform is a RELATIVE workspace root — which
// is also what TestConfig.Validate rejects, and which ResolveModuleIdentity must
// refuse rather than paper over, because this value decides a seed.
//
// Paths are built from t.TempDir() rather than "/w" literals so the row measures
// the derivation and not filepath.IsAbs's platform rules (see S5).
func TestResolveModuleIdentity_ErrorsAreLoud(t *testing.T) {
	base := t.TempDir()
	got, err := ResolveModuleIdentity("relative-root", filepath.Join(base, "cases", "s.ail"), "")
	if err == nil {
		t.Fatalf("expected an error, got nil (identity %q)", got)
	}
	if got != "" {
		t.Errorf("expected empty identity on error, got %q", got)
	}

	// An out-of-workspace input is NOT an error: it is workspace-relative like
	// any other, and stays independent of the absolute checkout path (AC9).
	esc, err := ResolveModuleIdentity(filepath.Join(base, "w"), filepath.Join(base, "other", "cases", "s.ail"), "")
	if err != nil {
		t.Fatalf("out-of-workspace input should resolve, got error: %v", err)
	}
	if esc != "../other/cases/s.ail" {
		t.Errorf("got %q, want %q", esc, "../other/cases/s.ail")
	}
}

// S7 — empty root, relative root, and unknown SeedMode each error; a valid
// config does not.
func TestTestConfig_Validate(t *testing.T) {
	// An absolute root on EVERY platform. "/w" is not absolute on Windows, which
	// would make the valid case fail and the bogus-SeedMode case below pass for
	// the wrong reason (rejected on the root, never reaching the mode switch).
	absRoot := t.TempDir()
	valid := TestConfig{WorkspaceRoot: absRoot, SeedMode: SeedModeDerived}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid config returned error: %v", err)
	}

	bad := []TestConfig{
		{WorkspaceRoot: "", SeedMode: SeedModeDerived},
		{WorkspaceRoot: "relative/root", SeedMode: SeedModeDerived},
		{WorkspaceRoot: absRoot, SeedMode: SeedMode("bogus")},
	}
	for i, c := range bad {
		if err := c.Validate(); err == nil {
			t.Errorf("case %d (%+v) expected an error, got nil", i, c)
		}
	}
}

// S8 — an injected failing entropy reader yields a non-nil error, seed 0, and
// no fallback. EntropyReader is restored for the other tests.
func TestNewRandomMasterSeed_InjectedFailure(t *testing.T) {
	orig := EntropyReader
	t.Cleanup(func() { EntropyReader = orig })
	EntropyReader = iotest.ErrReader(errAlways)

	seed, err := NewRandomMasterSeed()
	if err == nil {
		t.Fatalf("expected an error with failing reader, got nil (seed %d)", seed)
	}
	if seed != 0 {
		t.Errorf("expected seed 0 on failure, got %d", seed)
	}
}

var errAlways = io.ErrUnexpectedEOF

// S9 — a reader that yields only 4 bytes errors (io.ReadFull semantics) rather
// than zero-padding to 8.
func TestNewRandomMasterSeed_ShortRead(t *testing.T) {
	orig := EntropyReader
	t.Cleanup(func() { EntropyReader = orig })
	EntropyReader = bytes.NewReader([]byte("abcd"))

	seed, err := NewRandomMasterSeed()
	if err == nil {
		t.Fatalf("expected an error on a 4-byte read, got nil (seed %d)", seed)
	}
	if !strings.Contains(err.Error(), "failed to read 8 bytes of entropy") {
		t.Errorf("unexpected error text: %v", err)
	}
	if seed != 0 {
		t.Errorf("expected seed 0 on short read, got %d", seed)
	}
}
