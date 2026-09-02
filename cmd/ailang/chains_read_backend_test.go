package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// hermeticLocalObservatory points DefaultDatabasePath() at a throwaway home and
// creates the state directory sqlite needs.
//
// Without it, any arm exercising the LOCAL branch of openChainsReadBackend reads
// the developer's real ~/.ailang/state/observatory.db — so it passes on a machine
// that has one and fails on a clean runner with "unable to open database file: no
// such file or directory". That is not hypothetical: it red-lighted Build
// macos-latest on this very PR while the identical command was rc=0 locally, and
// the only variable was $HOME. Same class as iteration 195's Windows finding, one
// axis over: there the env var NAME differed per platform, here the env var VALUE
// differs per machine. setHomeDir (chains_post_dualwrite_test.go) sets all three
// variables os.UserHomeDir consults, so this stays honest on windows and plan9 too.
func hermeticLocalObservatory(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	if err := os.MkdirAll(filepath.Join(home, ".ailang", "state"), 0o755); err != nil {
		t.Fatalf("create hermetic state dir: %v", err)
	}
}

func TestRemoteReadRefusal_LocalOnlySurfaces(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	tests := []struct {
		command string
		reason  string
	}{
		{"chains live", "Store().DB()"},
		{"chains journey", "GetChainJourney"},
		{"chains stats --cost-per-verified-success", "Store"},
		{"chains find --task", "GetTaskSpanSummary"},
		{"observatory_*", "DB"},
	}
	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			err := refuseRemoteReadForLocalOnlySurface(test.command, "gcp")
			if err == nil || !strings.Contains(err.Error(), test.command) || !strings.Contains(err.Error(), test.reason) {
				t.Fatalf("error = %v, want command %q and reason %q", err, test.command, test.reason)
			}
		})
	}
	for _, command := range []string{"chains view", "chains list", "chains tree"} {
		if err := refuseRemoteReadForLocalOnlySurface(command, "gcp"); err != nil {
			t.Errorf("swappable command %q refused: %v", command, err)
		}
	}
	if err := refuseRemoteReadForLocalOnlySurface("chains live", "local"); err != nil {
		t.Errorf("explicit local mode refused: %v", err)
	}
	t.Setenv("AILANG_CHAINS_READ", "gcp")
	if err := refuseRemoteReadForLocalOnlySurface("chains journey", ""); err == nil {
		t.Fatal("environment-selected remote mode was not refused")
	}
}

func TestEvalRemoteReadIsRefused(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	commands := []struct {
		command string
		args    []string
	}{
		{"eval", nil}, {"eval-analyze", nil}, {"eval-compare", nil},
		{"eval-paired", nil}, {"eval-matrix", nil}, {"eval-sweet-spot", nil},
		{"eval-summary", nil}, {"eval-report", nil}, {"eval-suite", nil},
		{"eval-elo", nil}, {"eval-trend", nil}, {"eval-publish", nil},
		{"eval-chains", nil}, {"eval-chains", []string{"list"}},
	}
	for _, test := range commands {
		name := strings.Join(append([]string{test.command}, test.args...), " ")
		t.Run(name, func(t *testing.T) {
			args := append(append([]string{}, test.args...), "--remote", "gcp")
			err := guardEvalRemoteRead(test.command, args, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "D-15") || !strings.Contains(err.Error(), "#698 part 1") || !strings.Contains(err.Error(), test.command) {
				t.Fatalf("error = %v, want command and revisit trigger", err)
			}
		})
	}
	for _, command := range []string{"chains", "chains view", "messages"} {
		if err := guardEvalRemoteRead(command, []string{"--remote", "gcp"}, &bytes.Buffer{}); err != nil {
			t.Errorf("non-eval command %q refused: %v", command, err)
		}
	}
}

func TestEvalRemoteReadEnvWarnsAndProceeds(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "gcp")
	var warning bytes.Buffer
	if err := guardEvalRemoteRead("eval-paired", nil, &warning); err != nil {
		t.Fatalf("environment arm returned error: %v", err)
	}
	got := warning.String()
	for _, token := range []string{"eval-paired", "AILANG_CHAINS_READ", "D-15", "#698 part 1"} {
		if !strings.Contains(got, token) {
			t.Errorf("warning %q does not contain %q", got, token)
		}
	}
}

func TestOpenChainsReadBackend_DefaultsToLocal(t *testing.T) {
	hermeticLocalObservatory(t)
	t.Setenv("AILANG_CHAINS_READ", "")
	backend, closeBackend, err := openChainsReadBackend(context.Background(), "")
	if err != nil {
		t.Fatalf("openChainsReadBackend: %v", err)
	}
	defer closeBackend()
	if _, ok := backend.(*observatory.SQLiteBackend); !ok {
		t.Fatalf("backend type = %T, want *observatory.SQLiteBackend", backend)
	}
}

func TestOpenChainsReadBackend_RemoteRoutesThroughStorageMode(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	t.Setenv("AILANG_CLOUD_PROJECT", "")
	_, _, err := openChainsReadBackend(context.Background(), "gcp")
	if err == nil || !strings.Contains(err.Error(), "AILANG_CLOUD_PROJECT") {
		t.Fatalf("error = %v, want error containing AILANG_CLOUD_PROJECT", err)
	}
}

func TestOpenChainsReadBackend_EnvIsTheFallbackNotTheOverride(t *testing.T) {
	t.Run("env selects remote", func(t *testing.T) {
		t.Setenv("AILANG_CHAINS_READ", "gcp")
		t.Setenv("AILANG_CLOUD_PROJECT", "")
		_, _, err := openChainsReadBackend(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "AILANG_CLOUD_PROJECT") {
			t.Fatalf("error = %v, want remote routing error", err)
		}
	})

	t.Run("flag beats env", func(t *testing.T) {
		hermeticLocalObservatory(t)
		t.Setenv("AILANG_CHAINS_READ", "gcp")
		backend, closeBackend, err := openChainsReadBackend(context.Background(), "local")
		if err != nil {
			t.Fatalf("openChainsReadBackend: %v", err)
		}
		defer closeBackend()
		if _, ok := backend.(*observatory.SQLiteBackend); !ok {
			t.Fatalf("backend type = %T, want *observatory.SQLiteBackend", backend)
		}
	})
}

// TestEvalRemoteReadIsRefused_AllFlagSpellings pins the four argv spellings Go's
// flag package treats as the SAME flag. Before this, the guard matched only the
// double-dash forms, so `-remote gcp` fell through to the subcommand's own
// FlagSet and died with a generic "flag provided but not defined" -- losing the
// D-15 text, which is the entire signalling mechanism the D-15 ruling chose
// `view` in order to get. Found by the iteration-198 evaluator.
func TestEvalRemoteReadIsRefused_AllFlagSpellings(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	for _, args := range [][]string{
		{"--remote", "gcp"},
		{"--remote=gcp"},
		{"-remote", "gcp"},
		{"-remote=gcp"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			err := guardEvalRemoteRead("eval-paired", args, io.Discard)
			if err == nil {
				t.Fatalf("guardEvalRemoteRead(%q) = nil, want a refusal", args)
			}
			for _, token := range []string{"eval-paired", "D-15", "#698 part 1"} {
				if !strings.Contains(err.Error(), token) {
					t.Errorf("refusal %q does not contain %q", err, token)
				}
			}
		})
	}
}

// Positive control for the above: tokens that merely LOOK flag-shaped, or that
// name a different flag, must NOT be swallowed by the normalizer.
func TestEvalRemoteReadIsRefused_DoesNotOverMatch(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	for _, args := range [][]string{
		{"--remotely", "gcp"},
		{"--baseline", "v1.0"},
		{"--"},
		{"remote"},
		{"-"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if err := guardEvalRemoteRead("eval-paired", args, io.Discard); err != nil {
				t.Fatalf("guardEvalRemoteRead(%q) = %v, want nil (must not over-match)", args, err)
			}
		})
	}
}
