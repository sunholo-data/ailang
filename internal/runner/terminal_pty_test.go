package runner

import (
	"context"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// This exercises the actual CLI boundary, not a mock terminal or direct host call.
// Python's standard-library pty supplies the same harness on macOS and Linux.
func TestTerminalCLIPTYLifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native terminals support macOS/Linux")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 required for native terminal lifecycle controls: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "ailang")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/ailang")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, backend := range []string{"eval", "strict", "host"} {
		for _, scenario := range []string{"normal", "exit", "error", "sigint", "sigterm", "budget", "panic"} {
			if (backend == "host") != (scenario == "panic") {
				continue
			}
			if backend == "strict" && scenario == "budget" {
				continue
			} // VM explicitly rejects annotated budget frames.
			t.Run(backend+"/"+scenario, func(t *testing.T) {
				testCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
				defer done()
				cmd := exec.CommandContext(testCtx, python, "-c", terminalPTYHarness, binary, root, backend, scenario, os.Args[0])
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("PTY: %v\n%s", err, output)
				}
			})
		}
	}
}

const terminalPTYHarness = `
import os, pty, termios, fcntl, struct, subprocess, select, time, signal, sys, tempfile
binary, root, backend, scenario, helper = sys.argv[1:]
action = {
 'normal': 'let first = readEvent(session, -1); match first { Ok(Key(Up)) => println("UP"), _ => println("OTHER") }; let second = readEvent(session, -1); match second { Ok(Resize(size)) => println("RESIZE"), _ => println("OTHER") }; Completed(7)',
 'exit': 'let ev = readEvent(session, -1); exit(7)',
 'error': 'let ev = readEvent(session, -1); println(show(1 / 0))',
 'panic': '()',
 'budget': 'let ev = readEvent(session, -1); println("OVER_BUDGET")',
 'sigint': 'let ev = readEvent(session, -1); ()',
 'sigterm': 'let ev = readEvent(session, -1); ()',
}[scenario]
ending = 'match result { Ok(Completed(7)) => println("DONE"), _ => println("ERROR") }' if scenario == 'normal' else 'match result { Ok(_) => println("DONE"), Err(_) => println("ERROR") }'
effects = '{IO @limit=3}' if scenario == 'budget' else '{IO}'
source = 'module terminal_probe\nimport std/terminal (withTerminal, readEvent, Key, Up, Resize)\nimport std/result (Ok, Err)\nimport std/io (println, exit)\ntype Outcome = Completed(int)\nexport func main() -> () ! ' + effects + ' = { let result = withTerminal({alternate_screen: true, hide_cursor: true}, \\session. { println("READY"); ' + action + ' }); ' + ending + ' }\n'
with tempfile.TemporaryDirectory() as work:
 file = os.path.join(work, 'terminal_probe.ail')
 with open(file, 'w') as f: f.write(source)
 master, slave = pty.openpty()
 fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 24, 80, 0, 0))
 before = termios.tcgetattr(slave)
 command = [binary, 'run', '--stdlib-path', os.path.join(root, 'std'), '--caps', 'IO', '--entry', 'main']
 if backend == 'strict': command += ['--bytecode', '--strict-bytecode']
 command += [file]
 environment = dict(os.environ)
 if scenario == 'panic':
  command = [helper, '-test.run=^TestTerminalPTYPanicChild$']
  environment['AILANG_TEST_TERMINAL_PANIC_CHILD'] = '1'
 process = subprocess.Popen(command, cwd=work, env=environment, stdin=slave, stdout=slave, stderr=subprocess.PIPE)
 output = b''
 def until(needle):
  global output
  deadline = time.monotonic() + 12
  while needle not in output and time.monotonic() < deadline:
   if select.select([master], [], [], .05)[0]: output += os.read(master, 65536)
   if process.poll() is not None: break
  assert needle in output, (needle, output, process.poll(), process.stderr.read() if process.poll() is not None else b'')
 try:
  until(b'READY')
  active = termios.tcgetattr(slave)
  assert not active[3] & termios.ICANON and not active[3] & termios.ECHO
  assert active[3] & termios.ISIG
  if scenario == 'normal':
   os.write(master, b'\x1b[A'); until(b'UP')
   fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 8, 20, 0, 0)); until(b'RESIZE')
  elif scenario in ('sigint', 'sigterm'):
   os.kill(process.pid, signal.SIGINT if scenario == 'sigint' else signal.SIGTERM)
  else: os.write(master, b'x')
  process.wait(timeout=12)
  while select.select([master], [], [], .05)[0]: output += os.read(master, 65536)
  expected = {'normal': 0, 'exit': 7, 'error': 1, 'sigint': 130, 'sigterm': 143, 'panic': '()',
 'budget': 1, 'panic': 0}[scenario]
  assert process.returncode == expected, (process.returncode, expected, output, process.stderr.read())
  if scenario == 'normal': assert b'DONE' in output, output
  if scenario == 'budget': assert b'OVER_BUDGET' not in output, output
  after = termios.tcgetattr(slave)
  # macOS sets transient PENDIN when restoring canonical processing. It is
  # kernel input-reprocessing bookkeeping, not a changed user terminal mode.
  pending = getattr(termios, 'PENDIN', 0)
  before[3] &= ~pending; after[3] &= ~pending
  assert before == after, (before, after)
  assert b'\x1b[?25h' in output and b'\x1b[?1049l' in output, output
 finally:
  if process.poll() is None: process.kill(); process.wait()
  os.close(master); os.close(slave)
`

// Subprocess callback panic proves restoration on a real descriptor before the
// outer host catches the panic. AILANG runtime failures otherwise return errors.
func TestTerminalPTYPanicChild(t *testing.T) {
	if os.Getenv("AILANG_TEST_TERMINAL_PANIC_CHILD") != "1" {
		return
	}
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.IOReader, ctx.IOWriter = os.Stdin, os.Stdout
	ctx.TerminalSignalExit = os.Exit
	ctx.FnCaller = func(_ eval.Value, handle eval.Value) (eval.Value, error) {
		fmt.Fprintln(os.Stdout, "READY")
		if _, err := effects.TerminalReadEvent(ctx, []eval.Value{handle, &eval.IntValue{Value: -1}}); err != nil {
			t.Fatal(err)
		}
		panic("terminal panic fixture")
	}
	var caught any
	func() {
		defer func() { caught = recover() }()
		result, err := effects.TerminalWith(ctx, []eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{"alternate_screen": &eval.BoolValue{Value: true}, "hide_cursor": &eval.BoolValue{Value: true}}}, &eval.UnitValue{}})
		t.Fatalf("panic scope returned: %v %v", result, err)
	}()
	if caught != "terminal panic fixture" {
		t.Fatalf("panic = %v", caught)
	}
	fmt.Fprintln(os.Stdout, "RECOVERED")
}
