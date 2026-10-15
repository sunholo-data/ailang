package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProcessSpawnResultFallback(t *testing.T) {
	bin := buildAilang(t)
	for _, typed := range []bool{true, false} {
		t.Run(map[bool]string{true: "result", false: "legacy"}[typed], func(t *testing.T) {
			body := `match cancelProcess(spawnProcess("crew-nonexistent-worker", ["x"])) { _ => () }`
			if typed {
				body = `match spawnProcessResult("crew-nonexistent-worker", ["x"]) {
    Err(NotFound(cmd)) => println("fallback: ${cmd}"),
    Err(_) => println("unexpected failure"),
    Ok(h) => match cancelProcess(h) { _ => () }
   }`
			}
			src := filepath.Join(t.TempDir(), "capprobe.ail")
			code := `module capprobe
import std/io (println)
import std/process (spawnProcess, spawnProcessResult, cancelProcess, NotFound)
import std/result (Ok, Err)
export func main() -> () ! {IO,Process} {
 println("started");
 ` + body + `;
 println("after")
}`
			if err := os.WriteFile(src, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			out, stderr, exit := runAilangBin(t, bin, "run", "--quiet", "--relax-modules", "--caps", "IO,Process", "--process-allowlist", "crew-nonexistent-worker", src)
			if typed {
				if exit != 0 || out != "started\nfallback: crew-nonexistent-worker\nafter\n" {
					t.Fatalf("exit %d stdout %q stderr %s", exit, out, stderr)
				}
			} else if exit == 0 || strings.Contains(out, "after") || !strings.Contains(stderr, "_process_spawn_process: command not found: crew-nonexistent-worker") {
				t.Fatalf("legacy exit %d stdout %q stderr %s", exit, out, stderr)
			}
		})
	}
}
