package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// The Go emitter is gone. Every spelling that used to reach it fails with the
// removal message and exit 1 (not "unknown command" or "unknown flag"), writes
// nothing to stdout, and never reads its file argument.
func TestCompileRemovedFailsWithTheRemovalMessage(t *testing.T) {
	bin := buildAilang(t)
	for _, args := range [][]string{
		{"compile", "--emit-go", "examples/runnable/hello.ail"},
		{"compile", "--emit-go-v2", "--out", t.TempDir(), "missing.ail"},
		{"compile", "-emit-go", "-out", t.TempDir(), "missing.ail"},
		{"compile"},
		{"compile", "--emit-go", "--out", "gen", "missing.ail"},
		{"dev", "compile", "--emit-go", "missing.ail"},
	} {
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 1 {
			t.Errorf("%v: exit %d, want 1\n%s", args, code, stderr)
		}
		if stdout != "" {
			t.Errorf("%v: wrote to stdout: %q", args, stdout)
		}
		for _, want := range []string{"Go emitter", "were removed", "ailang run", "--bytecode"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("%v: stderr lacks %q:\n%s", args, want, stderr)
			}
		}
		for _, bad := range []string{"unknown command", "flag provided but not defined", "cannot read file"} {
			if strings.Contains(stderr, bad) {
				t.Errorf("%v: stderr says %q instead of the removal message:\n%s", args, bad, stderr)
			}
		}
	}
	// A help flag anywhere in the arguments is answered (exit 0, as
	// everywhere) with the same text on stdout.
	for _, args := range [][]string{
		{"compile", "--help"},
		{"compile", "-h"},
		{"compile", "--emit-go", "--help"},
		{"compile", "--emit-go-v2", "--out", "gen", "x.ail", "-help"},
		{"dev", "compile", "--emit-go", "-h"},
	} {
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 0 || !strings.Contains(stdout, "Go emitter") || !strings.Contains(stdout, "ailang run") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, stdout, stderr)
		}
	}
}

// The stub row is hidden: help no longer advertises a command that only fails.
func TestCompileRemovedIsHidden(t *testing.T) {
	c, ok := lookupCommand("compile")
	if !ok {
		t.Fatal("the compile row is gone: old scripts would get \"unknown command\" instead of the removal message")
	}
	if !c.Hidden {
		t.Error("the compile row is not Hidden")
	}
	var help bytes.Buffer
	printGroupHelp(&help, groupDev)
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*m")
	rows := map[string]bool{}
	for _, line := range strings.Split(ansi.ReplaceAllString(help.String(), ""), "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			rows[f[0]] = true
		}
	}
	if !rows["disasm"] { // control: the scan sees a visible dev row
		t.Fatalf("instrument failure: no disasm row in `ailang dev --help`:\n%s", help.String())
	}
	if rows["compile"] {
		t.Errorf("`ailang dev --help` still lists compile:\n%s", help.String())
	}
}
