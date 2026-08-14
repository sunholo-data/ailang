package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestChainsReadCommands_UseTheResolver(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	files, err := filepath.Glob(filepath.Join(dir, "chains*.go"))
	if err != nil {
		t.Fatal(err)
	}

	var direct []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") || filepath.Base(name) == "chains_read_backend.go" {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "NewSQLiteBackendFromPath(") {
			direct = append(direct, filepath.Base(name))
		}
	}
	sort.Strings(direct)
	want := []string{"chains.go", "chains_live.go", "chains_post.go", "chains_stats_cvs.go"}
	if strings.Join(direct, "\n") != strings.Join(want, "\n") {
		t.Fatalf("direct local-open files = %v, want exactly %v", direct, want)
	}
}
