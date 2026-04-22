package eval_harness

import (
	"os"
	"testing"
)

func TestParseMicroragMode(t *testing.T) {
	cases := map[string]MicroragMode{
		"":         MicroragModeAuto,
		"auto":     MicroragModeAuto,
		"unknown":  MicroragModeAuto,
		"on":       MicroragModeOn,
		"ON":       MicroragModeOn,
		"true":     MicroragModeOn,
		"enabled":  MicroragModeOn,
		"1":        MicroragModeOn,
		"off":      MicroragModeOff,
		"OFF":      MicroragModeOff,
		"disabled": MicroragModeOff,
		"0":        MicroragModeOff,
	}
	for in, want := range cases {
		if got := ParseMicroragMode(in); got != want {
			t.Errorf("ParseMicroragMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestApplyToEnv_OffStripsAndSets(t *testing.T) {
	env := []string{"PATH=/bin", "AILANG_MICRORAG_ENABLED=1", "FOO=bar"}
	out := MicroragModeOff.ApplyToEnv(env)
	hasOff := false
	hasOn := false
	for _, e := range out {
		if e == "AILANG_MICRORAG_ENABLED=0" {
			hasOff = true
		}
		if e == "AILANG_MICRORAG_ENABLED=1" {
			hasOn = true
		}
	}
	if !hasOff || hasOn {
		t.Fatalf("expected only ENABLED=0; got: %v", out)
	}
}

func TestApplyToEnv_AutoLeavesAlone(t *testing.T) {
	env := []string{"AILANG_MICRORAG_ENABLED=1"}
	out := MicroragModeAuto.ApplyToEnv(env)
	if len(out) != 1 || out[0] != "AILANG_MICRORAG_ENABLED=1" {
		t.Fatalf("auto must not mutate env; got %v", out)
	}
}

func TestApplyToEnv_OnStripsExistingOff(t *testing.T) {
	env := []string{"AILANG_MICRORAG_ENABLED=0", "PATH=/bin"}
	out := MicroragModeOn.ApplyToEnv(env)
	for _, e := range out {
		if e == "AILANG_MICRORAG_ENABLED=0" {
			t.Fatalf("on mode left old =0 entry: %v", out)
		}
	}
}

func TestResolvedState_AutoReadsEnv(t *testing.T) {
	t.Setenv("AILANG_MICRORAG_ENABLED", "0")
	if got := MicroragModeAuto.ResolvedState(); got != "off" {
		t.Errorf("auto should reflect env=0 → off, got %q", got)
	}
	t.Setenv("AILANG_MICRORAG_ENABLED", "1")
	if got := MicroragModeAuto.ResolvedState(); got != "on" {
		t.Errorf("auto should reflect env=1 → on, got %q", got)
	}
	_ = os.Unsetenv("AILANG_MICRORAG_ENABLED")
	if got := MicroragModeAuto.ResolvedState(); got != "on" {
		t.Errorf("auto with no env should default → on (engine default), got %q", got)
	}
}

func TestResolvedState_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("AILANG_MICRORAG_ENABLED", "1")
	if got := MicroragModeOff.ResolvedState(); got != "off" {
		t.Errorf("explicit off must override env=1, got %q", got)
	}
}
