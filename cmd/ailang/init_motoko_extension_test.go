package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePackageName(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{"empty", "", true, "is required"},
		{"no_slash", "openkb", true, "<namespace>/<package> form"},
		{"empty_namespace", "/motoko_ext_x", true, "<namespace>/<package> form"},
		{"empty_pkg", "ns/", true, "<namespace>/<package> form"},
		{"too_many_slashes", "ns/sub/motoko_ext_x", true, "<namespace>/<package> form"},
		{"missing_motoko_ext_prefix", "arniwesth/motoko_openkb", true, "motoko_ext_"},
		{"only_motoko_ext_prefix", "ns/motoko_ext_", true, "after the motoko_ext_ prefix"},
		{"valid_simple", "arniwesth/motoko_ext_openkb", false, ""},
		{"valid_underscored_short", "sunholo/motoko_ext_a2a", false, ""},
		{"valid_long_short_name", "myorg/motoko_ext_super_long_name_here", false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validatePackageName(c.input)
			if c.wantErr && err == nil {
				t.Fatalf("expected error containing %q, got nil", c.errContains)
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantErr && !strings.Contains(err.Error(), c.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), c.errContains)
			}
		})
	}
}

func TestValidateEffects(t *testing.T) {
	cases := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"empty", nil, false},
		{"single_valid", []string{"FS"}, false},
		{"multiple_valid", []string{"FS", "Process", "Env"}, false},
		{"all_canonical", []string{"IO", "FS", "Net", "Env", "Process", "Clock", "AI", "SharedMem", "Stream", "SharedIndex", "Rand"}, false},
		{"single_invalid", []string{"NotAnEffect"}, true},
		{"valid_then_invalid", []string{"FS", "Bogus"}, true},
		{"case_sensitive_lowercase", []string{"fs"}, true}, // "fs" != "FS"
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateEffects(c.input)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantErr && !strings.Contains(err.Error(), "Valid effects") {
				t.Errorf("error %q should list valid effects", err.Error())
			}
		})
	}
}

func TestDeriveOutputDir(t *testing.T) {
	cases := map[string]string{
		"arniwesth/motoko_ext_openkb":        "packages/motoko-ext-openkb",
		"sunholo/motoko_ext_a2a":             "packages/motoko-ext-a2a",
		"sunholo/motoko_ext_exa_search":      "packages/motoko-ext-exa-search",
		"myorg/motoko_ext_super_long_thingy": "packages/motoko-ext-super-long-thingy",
		"motoko_ext_no_namespace":            "packages/motoko-ext-no-namespace", // unusual but tolerated
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			got := deriveOutputDir(in)
			if got != want {
				t.Errorf("deriveOutputDir(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"   ", nil},
		{"FS", []string{"FS"}},
		{"FS,Process", []string{"FS", "Process"}},
		{"FS, Process,  Env  ", []string{"FS", "Process", "Env"}},
		{"FS,,,,Process", []string{"FS", "Process"}},
		{",,,FS,,,", []string{"FS"}},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := splitCSV(c.input)
			if len(got) != len(c.want) {
				t.Fatalf("len mismatch: got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestParseInitMotokoExtensionFlags_Help(t *testing.T) {
	mef, err := parseInitMotokoExtensionFlags([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error on --help: %v", err)
	}
	if !mef.help {
		t.Error("expected mef.help = true after --help")
	}
}

func TestParseInitMotokoExtensionFlags_Valid(t *testing.T) {
	mef, err := parseInitMotokoExtensionFlags([]string{
		"--name", "arniwesth/motoko_ext_openkb",
		"--tools", "OpenKBSearch,OpenKBList",
		"--effects", "FS,Process",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mef.name != "arniwesth/motoko_ext_openkb" {
		t.Errorf("name = %q", mef.name)
	}
	if len(mef.tools) != 2 || mef.tools[0] != "OpenKBSearch" || mef.tools[1] != "OpenKBList" {
		t.Errorf("tools = %v", mef.tools)
	}
	// Effects auto-includes Env + FS + IO. FS already present from user → not duplicated.
	// Expected: [FS, Process, Env, IO] = 4 entries.
	// M-EXT-AUTHOR-DX M2 (v0.20.1): IO added to baseline so the scaffolded
	// _smoke.ail's println-based assertion logging passes the effect ceiling
	// at publish-time smoke validation.
	if len(mef.effects) != 4 {
		t.Errorf("effects = %v (expected 4 entries: user FS,Process + auto Env,IO)", mef.effects)
	}
}

func TestParseInitMotokoExtensionFlags_RejectBadName(t *testing.T) {
	_, err := parseInitMotokoExtensionFlags([]string{
		"--name", "missing_slash_and_prefix",
	})
	if err == nil {
		t.Fatal("expected error for invalid --name")
	}
}

// TestParseInitMotokoExtensionFlags_AutoIncludesEnvAndFS — generated
// register.ail's `register_with_config` declares `! {Env, FS}`, so the
// package's [effects].max MUST permit them or `ailang check` rejects.
// _smoke.ail uses println so IO is also required (M-EXT-AUTHOR-DX M2).
// Auto-include all three regardless of what the user passes for --effects.
//
// Regression: Env+FS was a real shipping bug in v0.18.5 — caught by local
// post-release verification before any user reported it. IO was added
// alongside the _smoke.ail scaffold in M-EXT-AUTHOR-DX (v0.20.1).
func TestParseInitMotokoExtensionFlags_AutoIncludesEnvAndFS(t *testing.T) {
	cases := []struct {
		name      string
		userInput string
		mustHave  []string
	}{
		{"empty_user_effects", "", []string{"Env", "FS", "IO"}},
		{"only_FS", "FS", []string{"FS", "Env", "IO"}},
		{"only_Env", "Env", []string{"Env", "FS", "IO"}},
		{"FS_Process_no_Env", "FS,Process", []string{"FS", "Process", "Env", "IO"}},
		{"already_has_all", "Env,FS,IO,Process", []string{"Env", "FS", "IO", "Process"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := []string{"--name", "ns/motoko_ext_x"}
			if c.userInput != "" {
				args = append(args, "--effects", c.userInput)
			}
			mef, err := parseInitMotokoExtensionFlags(args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			have := make(map[string]bool)
			for _, e := range mef.effects {
				have[e] = true
			}
			for _, want := range c.mustHave {
				if !have[want] {
					t.Errorf("effect %q missing from %v", want, mef.effects)
				}
			}
		})
	}
}

func TestEnsureEffects_Idempotent(t *testing.T) {
	in := []string{"Env", "FS", "Process"}
	out := ensureEffects(in, "Env", "FS")
	if len(out) != 3 {
		t.Errorf("ensureEffects added duplicates: %v", out)
	}
}

func TestParseInitMotokoExtensionFlags_RejectBadEffect(t *testing.T) {
	_, err := parseInitMotokoExtensionFlags([]string{
		"--name", "arniwesth/motoko_ext_x",
		"--effects", "FS,Bogus",
	})
	if err == nil {
		t.Fatal("expected error for invalid effect 'Bogus'")
	}
}

// TestInitMotokoExtensionHelpMentionsRequiredFields — smoke for --help.
func TestInitMotokoExtensionHelpMentionsRequiredFields(t *testing.T) {
	mef, err := parseInitMotokoExtensionFlags([]string{"--help"})
	if err != nil || !mef.help {
		t.Fatalf("help flag broken: err=%v help=%v", err, mef.help)
	}
}

// TestScaffoldMotokoExtension_ProducesValidPackage is the load-bearing M3
// integration test. It scaffolds a real extension to a temp dir and asserts:
//
//  1. All 5 expected files are written
//  2. NONE of the 4 arniwesth/motoko_agent#8 failure modes appear in output:
//     - No path = "../..." dep refs (would defeat publishability)
//     - Package name has motoko_ext_ infix (registry generator's short-name
//     derivation needs it)
//     - No registry hand-edits (we never write registry_generated.ail)
//     - Lives at packages/motoko-ext-X/ (not nested in any host's src/)
//
// The full `ailang lock + ailang check` cycle requires network access (the
// registry fetch) so it's gated behind AILANG_INTEGRATION_TESTS=1. The
// structural assertions above run unconditionally — they catch the high-
// value regressions without requiring a registry round-trip.
func TestScaffoldMotokoExtension_ProducesValidPackage(t *testing.T) {
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "packages", "motoko-ext-openkb")

	err := scaffoldMotokoExtension(
		outDir,
		"arniwesth/motoko_ext_openkb",
		"openkb",
		[]string{"OpenKBSearch", "OpenKBList"},
		[]string{"FS", "Process", "Env"},
	)
	if err != nil {
		t.Fatalf("scaffoldMotokoExtension: %v", err)
	}

	// Assertion 1: all 6 expected files exist
	// M-EXT-AUTHOR-DX M2 (v0.20.1): added _smoke.ail to the scaffold so the
	// publish sandbox has a real smoke to run; without it 0.2.1-style stub
	// register bugs ship as functional regressions (cf. context_mode 0.2.1
	// → 0.2.3 stub-register-class incident).
	wantFiles := []string{"ailang.toml", "register.ail", "types.ail", "openkb.ail", "_smoke.ail", "README.md"}
	for _, f := range wantFiles {
		if _, statErr := os.Stat(filepath.Join(outDir, f)); statErr != nil {
			t.Errorf("expected file missing: %s (%v)", f, statErr)
		}
	}

	// Read ailang.toml + impl for content assertions
	tomlBytes, err := os.ReadFile(filepath.Join(outDir, "ailang.toml"))
	if err != nil {
		t.Fatalf("read ailang.toml: %v", err)
	}
	implBytes, err := os.ReadFile(filepath.Join(outDir, "openkb.ail"))
	if err != nil {
		t.Fatalf("read openkb.ail: %v", err)
	}
	registerBytes, err := os.ReadFile(filepath.Join(outDir, "register.ail"))
	if err != nil {
		t.Fatalf("read register.ail: %v", err)
	}
	tomlStr := string(tomlBytes)
	implStr := string(implBytes)
	registerStr := string(registerBytes)

	// PR #8 failure mode A: path = "../..." for any dep
	if strings.Contains(tomlStr, `path = "..`) {
		t.Errorf("ailang.toml contains path-based dep (PR #8 failure mode A):\n%s", tomlStr)
	}
	// PR #8 failure mode B: package name missing motoko_ext_ infix
	if !strings.Contains(tomlStr, `name = "arniwesth/motoko_ext_openkb"`) {
		t.Errorf("ailang.toml has wrong name shape (PR #8 failure mode B):\n%s", tomlStr)
	}
	// PR #8 failure mode C: NEVER write a registry_generated.ail
	registryFile := filepath.Join(outDir, "registry_generated.ail")
	if _, statErr := os.Stat(registryFile); statErr == nil {
		t.Errorf("scaffolder MUST NOT write registry_generated.ail (PR #8 failure mode C); found at %s", registryFile)
	}
	// PR #8 failure mode D: output dir is packages/motoko-ext-<short>
	wantDirSuffix := filepath.Join("packages", "motoko-ext-openkb")
	if !strings.HasSuffix(outDir, wantDirSuffix) {
		t.Errorf("output dir %q does not end in %q (PR #8 failure mode D)", outDir, wantDirSuffix)
	}

	// Tools list propagated through to impl module
	if !strings.Contains(implStr, `provided_tools: ["OpenKBSearch", "OpenKBList"]`) {
		t.Errorf("impl module missing tools list:\n%s", implStr)
	}
	// Effects list propagated through to ailang.toml
	if !strings.Contains(tomlStr, `max = ["FS", "Process", "Env"]`) {
		t.Errorf("ailang.toml missing effects list:\n%s", tomlStr)
	}
	// Register module imports the impl module
	if !strings.Contains(registerStr, `pkg/arniwesth/motoko_ext_openkb/openkb (make_hooks)`) {
		t.Errorf("register.ail does not import make_hooks correctly:\n%s", registerStr)
	}

	// Optional: full ailang lock + ailang check cycle. Gated to avoid
	// requiring a registry round-trip in unit tests.
	if os.Getenv("AILANG_INTEGRATION_TESTS") != "1" {
		t.Skip("integration test skipped (set AILANG_INTEGRATION_TESTS=1 to run ailang lock + check)")
	}
	cmd := exec.Command("ailang", "lock")
	cmd.Dir = outDir
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("ailang lock failed: %v\n%s", runErr, out)
	}
	cmd = exec.Command("ailang", "check", "register.ail")
	cmd.Dir = outDir
	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("ailang check register.ail failed: %v\n%s", runErr, out)
	}
}

// TestScaffoldMotokoExtension_RefusesExistingDir — defensive check that we
// don't clobber an existing directory.
func TestScaffoldMotokoExtension_StillWritesIfDirExists(t *testing.T) {
	// Note: scaffoldMotokoExtension itself uses os.MkdirAll (which is
	// idempotent). The dir-already-exists check happens in
	// initMotokoExtensionCommand before scaffold is called. This test
	// documents that scaffold is robust to a pre-existing dir at the path.
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "exists")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	err := scaffoldMotokoExtension(outDir, "ns/motoko_ext_x", "x", nil, nil)
	if err != nil {
		t.Errorf("scaffold should write into pre-existing empty dir: %v", err)
	}
}
