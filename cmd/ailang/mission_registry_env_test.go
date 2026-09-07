package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissionRegistryExplicitPath(t *testing.T) {
	dir := t.TempDir()
	work := t.TempDir()
	body := fmt.Sprintf("name = \"canary\"\nrepo = \"example/project\"\nworkdir = %q\ndoc = \"README.md\"\n[schedule]\nmode = \"interval\"\ninterval_seconds = 21600\nboot_offset = 840\n", filepath.ToSlash(work))
	if err := os.WriteFile(filepath.Join(dir, "canary.toml"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_MISSION_REGISTRY", dir)
	reg, err := loadMissionRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := reg.Get("canary")
	if !ok || filepath.Clean(m.Workdir) != filepath.Clean(work) {
		t.Fatalf("explicit registry not used: %+v", reg.Names())
	}
}
func TestMissionRegistryExplicitInvalidDoesNotFallback(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"relative-missions", filepath.Join(t.TempDir(), "missing"), file} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("AILANG_MISSION_REGISTRY", value)
			_, err := loadMissionRegistry()
			if err == nil || !strings.Contains(err.Error(), "AILANG_MISSION_REGISTRY") {
				t.Fatalf("expected explicit registry error, got %v", err)
			}
		})
	}
}
