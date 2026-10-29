package pkg_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestReadPublishKeyFile_PublicProtectedFiles(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "")
	t.Setenv(config.EnvRegistryAPIKey, "unused-dummy-env-key")
	cases := []struct {
		name, input string
		mode        os.FileMode
		valid       bool
	}{
		{"readonly-trim", " \tdummy-private-value\n", 0400, true},
		{"readwrite-trim", " dummy-private-value\n", 0600, true},
		{"empty", " \n", 0600, false},
		{"multiline", "dummy-private-value\nNEXT", 0600, false},
		{"control", "dummy-private-value\x00SUFFIX", 0600, false},
		{"oversized", strings.Repeat("Z", 16*1024+1), 0600, false},
		{"public-mode", "dummy-private-value", 0644, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "publisher-key")
			if err := os.WriteFile(path, []byte(tt.input), tt.mode); err != nil {
				t.Fatal(err)
			}
			key, err := pkg.ReadPublishKeyFile(path)
			if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
				if !errors.Is(err, pkg.ErrPublishKeyFileUnsupported) || key != "" {
					t.Fatalf("unsupported host accepted protected file: %v", err)
				}
				return
			}
			if tt.valid {
				if err != nil || key != "dummy-private-value" {
					t.Fatalf("protected trimmed file rejected: %v", err)
				}
			} else if err == nil || key != "" || strings.Contains(err.Error(), "dummy-private-value") || strings.Contains(err.Error(), "unused-dummy-env-key") {
				t.Fatalf("invalid file accepted, fell back, or exposed contents: %v", err)
			}
		})
	}
}

func TestReadPublishKeyFile_PublicMissingAndAuthority(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "")
	path := filepath.Join(t.TempDir(), "absent-key")
	key, err := pkg.ReadPublishKeyFile(path)
	if err == nil || key != "" {
		t.Fatal("missing file accepted")
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing cause lost: %v", err)
		}
	} else if !errors.Is(err, pkg.ErrPublishKeyFileUnsupported) {
		t.Fatalf("unsupported host probed file: %v", err)
	}
	t.Setenv(config.EnvAgentPolicy, "private-policy")
	key, err = pkg.ReadPublishKeyFile(path)
	var refusal *pkg.ConfinedError
	if key != "" || !errors.As(err, &refusal) || refusal.Authority != pkg.AuthNetwork || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("authority did not precede file access: %v", err)
	}
}
