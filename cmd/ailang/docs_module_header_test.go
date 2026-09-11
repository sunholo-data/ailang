package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// `ailang docs <module>` used to print only the FIRST header line, so the
// std/process header's pointer to std/io.exit — and every module's security
// and semantics notes — were invisible. A CLI author looking in std/process
// for a way to fail non-zero concluded it did not exist (email-parse,
// 2026-09-04) although exit() had shipped in v0.10.1.
func TestDocsModule_HeaderIsShownInFull(t *testing.T) {
	stdlib := filepath.Join(findRepoRootForTest(t), "std")
	var process, io *moduleDoc
	mods := discoverModules(stdlib)
	for i := range mods {
		switch mods[i].Name {
		case "std/process":
			process = &mods[i]
		case "std/io":
			io = &mods[i]
		}
	}
	if process == nil || io == nil {
		t.Fatal("std/process or std/io not discovered")
	}

	// --list stays a one-liner: Description is the first meaningful line only.
	if strings.Contains(process.Description, "\n") || !strings.HasPrefix(process.Description, "Execute external commands") {
		t.Errorf("Description should be the first header line, got %q", process.Description)
	}
	// The module page carries the whole header, including the exit pointer.
	header := strings.Join(process.Header, "\n")
	for _, want := range []string{"--process-allowlist", "import std/io (exit)", "THIS program's own exit code"} {
		if !strings.Contains(header, want) {
			t.Errorf("std/process header lacks %q:\n%s", want, header)
		}
	}
	if strings.Contains(header, "std/process.ail") || strings.Contains(header, "std/process - ") {
		t.Errorf("file-name line leaked into the header:\n%s", header)
	}
	if !strings.Contains(io.Description, "exit(code)") {
		t.Errorf("std/io one-liner should name exit(code): %q", io.Description)
	}
}
