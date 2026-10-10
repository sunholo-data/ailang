package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Extend the existing actual-descriptor terminal controls with a live owned
// worker. This is a subprocess test: it checks exact PID disappearance and
// restored termios state before accepting the CLI signal/return status.
func TestWorkerTerminalPTYLifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native POSIX terminals")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "ailang")
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./cmd/ailang")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	harness := strings.Replace(terminalPTYHarness, "with tempfile.TemporaryDirectory() as work:", `source = source.replace('import std/result', 'import std/process (spawnProcess, closeProcessStdin)\nimport std/result')
source = source.replace('let result = withTerminal', 'let worker = spawnProcess(' + __import__('json').dumps(sys.executable) + ', ["worker.py"]); closeProcessStdin(worker); let result = withTerminal')
with tempfile.TemporaryDirectory() as work:
 with open(os.path.join(work, 'worker.py'), 'w') as f:
  f.write('import os,time\nopen("worker.pid","w").write(str(os.getpid()))\ntime.sleep(60)\n')`, 1)
	harness = strings.Replace(harness, "effects = '{IO @limit=3}' if scenario == 'budget' else '{IO}'", "effects = '{Process, IO @limit=3}' if scenario == 'budget' else '{Process, IO}'", 1)
	harness = strings.Replace(harness, "'--caps', 'IO', '--entry'", "'--caps', 'IO,Process', '--process-allowlist', sys.executable, '--entry'", 1)
	harness = strings.Replace(harness, "  until(b'READY')", `  until(b'READY')
  deadline = time.monotonic() + 5
  pidfile = os.path.join(work, 'worker.pid')
  while not os.path.exists(pidfile) and time.monotonic() < deadline: time.sleep(.01)
  assert os.path.exists(pidfile), 'worker never acknowledged startup'
  worker_pid = int(open(pidfile).read())
  os.kill(worker_pid, 0)`, 1)
	harness = strings.Replace(harness, "  after = termios.tcgetattr(slave)", `  probe = subprocess.run(['ps', '-o', 'stat=', '-p', str(worker_pid)], capture_output=True, text=True)
  assert probe.returncode in (0, 1), ('PID probe failed', probe.returncode, probe.stderr)
  assert not probe.stdout.strip(), ('owned worker remains alive or unreaped', worker_pid, probe.stdout)
  after = termios.tcgetattr(slave)`, 1)

	harness = strings.Replace(harness, " output = b''", " output = b''\n worker_pid = None\n worker_absent = False", 1)
	harness = strings.Replace(harness, "  after = termios.tcgetattr(slave)", "  worker_absent = True\n  after = termios.tcgetattr(slave)", 1)
	harness = strings.Replace(harness, "  os.close(master); os.close(slave)", `  if not worker_absent:
   pidfile = os.path.join(work, 'worker.pid')
   if worker_pid is None and os.path.exists(pidfile):
    try: worker_pid = int(open(pidfile).read())
    except ValueError: pass
   if worker_pid and worker_pid > 1:
    try: os.kill(worker_pid, signal.SIGKILL)
    except ProcessLookupError: pass
  os.close(master); os.close(slave)`, 1)
	for _, backend := range []string{"eval", "strict"} {
		for _, scenario := range []string{"normal", "exit", "error", "sigint", "sigterm", "budget"} {
			if backend == "strict" && scenario == "budget" {
				continue
			}
			t.Run(backend+"/"+scenario, func(t *testing.T) {
				childCtx, done := context.WithTimeout(t.Context(), 30*time.Second)
				defer done()
				command := exec.CommandContext(childCtx, python, "-c", harness, binary, root, backend, scenario, os.Args[0])
				if out, err := command.CombinedOutput(); err != nil {
					t.Fatalf("worker PTY: %v\n%s", err, out)
				}
			})
		}
	}
}
