package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFSBytesResultBothEngines runs std/fs readFileRaw / writeFileBytesResult
// end to end on the evaluator and on --bytecode (where FS builtins dispatch
// through the evaluator bridge, like every other FS builtin). Non-UTF-8 bytes
// must survive the write/read round trip unchanged, and a write into a missing
// directory must be Err, not a panic.
func TestFSBytesResultBothEngines(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, backend := range []struct {
		name  string
		extra []string
	}{
		{"evaluator", nil},
		{"bytecode", []string{"--bytecode"}},
	} {
		t.Run(backend.name, func(t *testing.T) {
			dir := t.TempDir()
			// AILANG string literals use forward slashes on every platform.
			base := filepath.ToSlash(dir)
			src := filepath.Join(dir, "fsbytes.ail")
			prog := fmt.Sprintf(`module test/fsbytes

import std/fs (readFileRaw, writeFileBytesResult)
import std/bytes (fromInts, toInts)
import std/io (println)

func w(r: Result[(), string]) -> string = match r {
  Ok(_) => "ok",
  Err(_) => "err"
}

func rd(r: Result[bytes, string]) -> string = match r {
  Ok(b) => show(toInts(b)),
  Err(_) => "err"
}

export func main() -> () ! {IO, FS} {
  let data = fromInts([0, 255, 254, 128, 97, 10])
  let good = writeFileBytesResult("%[1]s/out.bin", data)
  let _ = println("write=${w(good)}")
  let back = readFileRaw("%[1]s/out.bin")
  let _ = println("read=${rd(back)}")
  let bad = writeFileBytesResult("%[1]s/missing/dir/x.bin", data)
  let _ = println("badwrite=${w(bad)}")
  let gone = readFileRaw("%[1]s/absent.bin")
  println("badread=${rd(gone)}")
}
`, base)
			if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"run", "--relax-modules", "--caps", "IO,FS"}, backend.extra...)
			args = append(args, src)
			stdout, stderr, code := runCLI(t, args...)
			if code != 0 {
				t.Fatalf("exit %d\nstdout=%s\nstderr=%s", code, stdout, stderr)
			}
			for _, want := range []string{"write=ok", "read=[0, 255, 254, 128, 97, 10]", "badwrite=err", "badread=err"} {
				if !strings.Contains(stdout, want) {
					t.Errorf("missing %q in output:\n%s", want, stdout)
				}
			}
			got, err := os.ReadFile(filepath.Join(dir, "out.bin"))
			if err != nil {
				t.Fatal(err)
			}
			if want := []byte{0, 255, 254, 128, 97, 10}; !bytes.Equal(got, want) {
				t.Errorf("file bytes = %v, want %v", got, want)
			}
		})
	}
}
