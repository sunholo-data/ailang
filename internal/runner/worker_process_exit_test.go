package runner

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Exact-PID CLI/batch controls exercise the original ownership failure after
// acknowledged child startup. Child sleep is a provider-free Clock effect.
func TestWorkerCLIProcessExitLifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX process-group contract")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
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
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, worker := range []string{"managed", "async"} {
		for _, backend := range []string{"eval", "strict"} {
			for _, scenario := range []string{"normal", "exit", "error", "budget", "sigint", "sigterm", "batch-normal", "batch-exit", "batch-error"} {
				if backend == "strict" && scenario == "budget" {
					continue
				} // VM explicitly rejects annotated budget frames.
				t.Run(worker+"/"+backend+"/"+scenario, func(t *testing.T) {
					probeCtx, stop := context.WithTimeout(t.Context(), 30*time.Second)
					defer stop()
					cmd := exec.CommandContext(probeCtx, python, "-c", workerExitPIDHarness, binary, root, worker, backend, scenario)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("worker exit: %v\n%s", err, output)
					} else {
						t.Log(string(output))
					}
				})
			}
		}
	}
	t.Run("original-select-handler-false", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, python, "-c", workerExitPIDHarness, binary, root, "async", "eval", "select")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("original async: %v\n%s", err, output)
		} else {
			t.Log(string(output))
		}
	})
}

const workerExitPIDHarness = `
import os,sys,subprocess,tempfile,time,json,shlex,signal
binary,root,kind,backend,scenario = sys.argv[1:]
with tempfile.TemporaryDirectory() as work:
 pidfile,ready = os.path.join(work,'worker.pid'),os.path.join(work,'worker.ready')
 childfile = os.path.join(work,'worker.ail')
 with open(childfile,'w') as f:
  f.write('module worker\nimport std/fs (writeFile)\nimport std/clock (sleep)\nimport std/io (println)\nexport func main() -> () ! {FS, Clock, IO} { writeFile('+json.dumps(ready)+',"started"); println("worker started"); sleep(60000); () }\n')
 shell = 'echo $$ > '+shlex.quote(pidfile)+'; exec '+shlex.join([binary,'run','--stdlib-path',os.path.join(root,'std'),'--caps','FS,Clock,IO',childfile])
 args = json.dumps(['-c',shell])
 spawn = 'spawnProcess("/bin/sh",'+args+')' if kind == 'managed' else 'asyncExecProcess("/bin/sh",'+args+',"worker",5,1)'
 close = 'closeProcessStdin(child); ' if kind == 'managed' else ''
 action = '()'
 if scenario.endswith('exit'): action = 'exit(7)'
 if scenario.endswith('error'): action = 'println(show(1 / 0))'
 if scenario == 'budget': action = 'println("OVER_BUDGET")'
 row = '{IO @limit=1, Process}' if scenario == 'budget' and kind == 'managed' else '{IO @limit=1, Stream, Process}' if scenario == 'budget' else '{IO, Stream, Process}'
 imports = 'import std/io (readLine,println,exit)\nimport std/process (spawnProcess,closeProcessStdin)\nimport std/stream (asyncExecProcess,asyncReadStdinLines,selectEvents,StreamEvent,SourceText)\n'
 handler = 'pure func stop(e:StreamEvent) -> bool = match e { SourceText(_,_) => false, _ => true }\n'
 wait = 'let keys=asyncReadStdinLines("keys",10); selectEvents([child,keys],stop); ' if scenario == 'select' else 'let key=readLine(); '
 hostfile = os.path.join(work,'worker_host.ail')
 with open(hostfile,'w') as f: f.write('module worker_host\n'+imports+handler+'export func main() -> () ! '+row+' { let child='+spawn+'; '+wait+close+action+' }\n')
 command = [binary,'run','--stdlib-path',os.path.join(root,'std'),'--quiet','--caps','IO,Stream,Process','--process-allowlist','/bin/sh']
 if backend == 'strict': command += ['--bytecode','--strict-bytecode']
 if scenario.startswith('batch-'): command += ['--batch']
 command += [hostfile]
 if scenario.startswith('batch-'): command += ['input-one']
 process = subprocess.Popen(command,cwd=work,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 worker_pid = None
 worker_absent = False
 try:
  deadline = time.monotonic()+12
  while not os.path.exists(ready) and time.monotonic()<deadline and process.poll() is None: time.sleep(.005)
  assert os.path.exists(ready), ('child did not acknowledge startup',process.poll(),process.communicate(timeout=3) if process.poll() is not None else '')
  with open(pidfile) as f: worker_pid=int(f.read())
  assert worker_pid > 1
  os.kill(worker_pid,0)
  start = time.monotonic()
  if scenario in ('sigint','sigterm'): os.kill(process.pid,signal.SIGINT if scenario=='sigint' else signal.SIGTERM)
  else: process.stdin.write(b'quit\n'); process.stdin.flush()
  stdout,stderr = process.communicate(timeout=4)
  elapsed = time.monotonic()-start
  expected = 130 if scenario=='sigint' else 143 if scenario=='sigterm' else 1 if scenario in ('error','budget','batch-exit','batch-error') else 7 if scenario=='exit' else 0
  assert process.returncode==expected,(process.returncode,expected,stdout,stderr)
  assert elapsed<=2.5,('total shutdown deadline exceeded',elapsed)
  probe=subprocess.run(['ps','-o','pid,ppid,state','-p',str(worker_pid)],capture_output=True,text=True)
  assert probe.returncode in (0,1),(probe.returncode,probe.stderr)
  assert len(probe.stdout.strip().splitlines())<=1,('owned worker or zombie survived host',probe.stdout)
  worker_absent = True
  if scenario=='budget': assert b'OVER_BUDGET' not in stdout,stdout
  print(json.dumps(dict(worker=kind,backend=backend,scenario=scenario,host_pid=process.pid,worker_pid=worker_pid,exit=process.returncode,shutdown_ms=round(elapsed*1000),worker_absent=True)))
 finally:
  if process.poll() is None: process.kill(); process.wait()
  # Only this probe's recorded PID is eligible for failure cleanup.
  if worker_pid is None and os.path.exists(pidfile):
   with open(pidfile) as f: worker_pid=int(f.read())
  if not worker_absent and worker_pid and worker_pid>1:
   try: os.kill(worker_pid,signal.SIGKILL)
   except ProcessLookupError: pass
`
