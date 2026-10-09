package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestRunCommand_ReadLineOptEOF(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("examples", "runnable", "io_readline_eof.ail")
	for _, tc := range []struct{ name, input, output string }{
		{"blank preserved", "a\n\nb\n", "a\n\nb\n"},
		{"empty", "", ""},
		{"unterminated", "a\nb", "a\nb\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "run", "--caps", "IO", "--entry", "main", src)
			cmd.Dir = root
			testutil.SetHomeDir(t, t.TempDir())
			cmd.Stdin = strings.NewReader(tc.input)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("run failed (%v): %v\n%s", ctx.Err(), err, &stderr)
			}
			if got := strings.ReplaceAll(stdout.String(), "\r\n", "\n"); got != tc.output {
				t.Fatalf("got %q, want %q; stderr: %s", got, tc.output, &stderr)
			}
		})
	}
}

func TestRunCommand_ReadLineLegacyBlank(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "main.ail")
	const prog = `module main
import std/io (readLine, println)
func loop() -> () ! {IO} {
  let line = readLine(());
  if line == "" then () else { println(line); loop() }
}
export func main() -> () ! {IO} { loop() }
`
	if err := os.WriteFile(src, []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "run", "--relax-modules", "--caps", "IO", "--entry", "main", src)
	testutil.SetHomeDir(t, t.TempDir())
	cmd.Stdin = strings.NewReader("a\n\nb\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("run failed: %v\n%s", err, &stderr)
	}
	if string(got) != "a\n" {
		t.Fatalf("legacy output changed: %q", got)
	}
}
