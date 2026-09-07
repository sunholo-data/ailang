package iteration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeBindingStrictAndReadOnly(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mission-runtime.toml")
	if _, err := LoadBinding(file); err == nil {
		t.Fatal("missing binding accepted")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatal("missing binding created state")
	}
	good := "version = 1\nstate_db = '/tmp/runtime.sqlite'\nworkspace_root = '/tmp/workspaces'\n"
	for name, body := range map[string]string{"valid": good, "dsn_query": strings.Replace(good, "runtime.sqlite", "runtime?x.sqlite", 1), "dsn_fragment": strings.Replace(good, "runtime.sqlite", "runtime#x.sqlite", 1), "unknown": good + "state_dbb = '/tmp/x'\n", "relative": strings.Replace(good, "/tmp/runtime.sqlite", "runtime.sqlite", 1), "version": strings.Replace(good, "version = 1", "version = 2", 1), "missing": "version = 1\n", "duplicate": good + "version = 1\n"} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(file, []byte(body), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := LoadBinding(file)
			if (err == nil) != (name == "valid") {
				t.Fatalf("binding %s: %v", name, err)
			}
		})
	}
}
