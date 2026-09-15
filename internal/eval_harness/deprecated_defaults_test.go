package eval_harness

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// Site 2 (M-V1-SIMPLIFY-S4 M1): DefaultAgentConfig must not name a model.
// cmd/ailang/eval_benchmark_agent.go treats a non-empty ClaudeModel as an
// explicit --agent-model override that skips the models.yml lookup, so a
// "haiku" here was not a fallback but a silent override of the benchmark's
// configured model. Empty routes through the registry like every real run.
func TestDefaultAgentConfig_HasNoModel(t *testing.T) {
	if m := DefaultAgentConfig().ClaudeModel; m != "" {
		t.Fatalf("DefaultAgentConfig().ClaudeModel = %q, want empty: a default model here overrides the registry", m)
	}
}

// Site 7: the grade probe's binary. AILANG_BIN wins; otherwise the PATH
// `ailang` is served as a deprecated default whose warning names the RESOLVED
// path (the stale-binary trap), and AILANG_STRICT_CONFIG=1 refuses it.
func TestResolveAILANGBin_DeprecatedPathFallbackThenStrict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH lookup of a shell stub is POSIX-shaped")
	}
	t.Setenv(config.EnvStrict, "")
	t.Setenv(EnvAILANGBin, "")

	// A fake `ailang` on a PATH we own, so the resolved path is known exactly.
	bin := t.TempDir()
	fake := filepath.Join(bin, "ailang")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	got, err := resolveAILANGBin()
	if err != nil || got != fake {
		t.Fatalf("unset: (%q, %v), want the PATH binary %q served (and named in the warning)", got, err, fake)
	}

	t.Setenv(config.EnvStrict, "1")
	if got, err = resolveAILANGBin(); got != "" || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: (%q, %v), want config.ErrDeprecatedDefault", got, err)
	}

	pinned := filepath.Join(t.TempDir(), "ailang-under-test")
	t.Setenv(EnvAILANGBin, pinned)
	if got, err = resolveAILANGBin(); err != nil || got != pinned {
		t.Fatalf("strict with %s set: (%q, %v)", EnvAILANGBin, got, err)
	}

	// No binary anywhere is an error in both modes, never a bare "ailang".
	t.Setenv(EnvAILANGBin, "")
	t.Setenv(config.EnvStrict, "")
	t.Setenv("PATH", t.TempDir())
	if got, err = resolveAILANGBin(); err == nil || got != "" {
		t.Fatalf("no ailang on PATH: (%q, %v), want an error", got, err)
	}
}
