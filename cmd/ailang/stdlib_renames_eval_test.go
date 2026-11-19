package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every deprecated std spelling and its replacement give the same answer on
// the evaluator, including list.member, whose replacement contains takes its
// arguments the other way round.
func TestStdlibRenamesEvaluatorTranscript(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "renameprobe.ail")
	code := `module renameprobe
import std/list (member, contains)
import std/bytes (fromString, toString, slice, sliceLen)
import std/math (abs_Float, absFloat, abs_Int, absInt)
import std/array (fromList, get, unsafeGet)
import std/option (Option, Some, None)
pure func showOpt(o: Option[bytes]) -> string = match o { Some(b) => toString(b), None => "none" }
export pure func summary() -> string = "${show(member(2, [1, 2, 3]))},${show(member(5, [1, 2, 3]))},${showOpt(slice(fromString("hello"), 1, 3))},${showOpt(sliceLen(fromString("hello"), 1, 3))},${show(abs_Float(0.0 - 2.5))},${show(absFloat(0.0 - 2.5))},${show(contains([1, 2, 3], 2))},${show(contains([1, 2, 3], 5))},${show(abs_Int(0 - 4))},${show(absInt(0 - 4))},${show(unsafeGet(fromList([7, 8, 9]), 2))},${show(get(fromList([7, 8, 9]), 2))}"
`
	if err := os.WriteFile(probe, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	got, stderr, rc := runAilangBin(t, bin, "run", "--stdlib-path", filepath.Join(root, "std"), "--entry", "summary", probe)
	if rc != 0 {
		t.Fatalf("evaluator: exit %d\n%s", rc, stderr)
	}
	if got = strings.TrimSpace(got); got != "true,false,ell,ell,2.5,2.5,true,false,4,4,9,9" {
		t.Fatalf("evaluator transcript changed: %q", got)
	}
}
