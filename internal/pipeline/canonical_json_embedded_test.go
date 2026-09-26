package pipeline

import (
	"context"
	"strings"
	"testing"
)

// `ailang iface std/fs` from a checkout with no std/ on disk — every package
// repo the ailang_only lane works in — must serve the embedded stdlib, and
// ONLY std/: a missing user module stays a missing file.
func TestBuildCanonicalJSON_EmbeddedStdFallback(t *testing.T) {
	dir := t.TempDir() // no std/ here
	got, err := BuildCanonicalJSON(context.Background(), dir, "std/fs")
	if err != nil {
		t.Fatalf("std/fs outside a checkout: %v", err)
	}
	for _, want := range []string{`"module": "std/fs"`, `"name": "walk"`, `"name": "glob"`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %s in embedded iface", want)
		}
	}
	if _, err := BuildCanonicalJSON(context.Background(), dir, "std/fs.ail"); err != nil {
		t.Errorf("with .ail suffix: %v", err)
	}
	if _, err := BuildCanonicalJSON(context.Background(), dir, "std/nope"); err == nil || !strings.Contains(err.Error(), "cannot read file") {
		t.Errorf("unknown std module must stay a read error, got %v", err)
	}
	if _, err := BuildCanonicalJSON(context.Background(), dir, "mymod"); err == nil || !strings.Contains(err.Error(), "cannot read file") {
		t.Errorf("user module must not fall back to std, got %v", err)
	}
}
