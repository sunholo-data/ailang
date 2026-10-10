package runner

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWorkerLifecycleExistingFixtures(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("POSIX process fixtures")
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
	fixtures := []struct {
		name, input string
		expected    []string
	}{
		{"process_stdin_write", "", []string{"wrote line 1", "wrote line 2", "wrote line 3"}},
		{"stream_process_source", "", []string{"[echo] hello from subprocess", "done"}},
		{"stream_multi_source", "hello\n", []string{"stdin: hello", "done"}},
		{"process_demo", "", []string{"stdout: hello from AILANG", "exitCode: 1", "not found: nonexistent_cmd_xyz", "stderr: stderr_msg", "Done!"}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, binary, "run", "--stdlib-path", filepath.Join(root, "std"), "--quiet", "--caps", "IO,Stream,Process", filepath.Join("examples", "runnable", fixture.name+".ail"))
			cmd.Dir = root
			cmd.Stdin = strings.NewReader(fixture.input)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("fixture: %v\n%s", err, output)
			}
			for _, want := range fixture.expected {
				if !strings.Contains(string(output), want) {
					t.Fatalf("missing %q: %s", want, output)
				}
			}
		})
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	for _, backend := range []string{"eval", "strict"} {
		for _, key := range []string{"q", "escape"} {
			t.Run("terminal_keys/"+backend+"/"+key, func(t *testing.T) {
				cmd := exec.CommandContext(ctx, python, "-c", workerTerminalFixtureHarness, binary, root, backend, key)
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("terminal fixture: %v\n%s", err, output)
				}
			})
		}
	}
}

const workerTerminalFixtureHarness = `
import os,sys,pty,termios,subprocess,select,time,fcntl,struct
binary,root,backend,key=sys.argv[1:]
master,slave=pty.openpty()
fcntl.ioctl(slave,termios.TIOCSWINSZ,struct.pack("HHHH",24,80,0,0))
before=termios.tcgetattr(slave)
command=[binary,'run','--stdlib-path',os.path.join(root,'std'),'--quiet','--caps','IO']
if backend=='strict': command+=['--bytecode','--strict-bytecode']
command+=[os.path.join('examples','runnable','terminal_keys.ail')]
process=subprocess.Popen(command,cwd=root,stdin=slave,stdout=slave,stderr=subprocess.PIPE)
output=b''
try:
 deadline=time.monotonic()+12
 while b'Press q or Escape' not in output and time.monotonic()<deadline:
  if select.select([master],[],[],.05)[0]: output+=os.read(master,65536)
  if process.poll() is not None: break
 assert b'Press q or Escape' in output,(output,process.poll(),process.stderr.read() if process.poll() is not None else b'')
 active=termios.tcgetattr(slave)
 assert not active[3]&termios.ICANON and not active[3]&termios.ECHO
 os.write(master,b'q' if key=='q' else b'\x1b')
 process.wait(timeout=4)
 assert process.returncode==0,(process.returncode,output,process.stderr.read())
 after=termios.tcgetattr(slave)
 pending=getattr(termios,'PENDIN',0)
 before[3]&=~pending;after[3]&=~pending
 assert before==after,(before,after)
finally:
 if process.poll() is None: process.kill();process.wait()
 os.close(master);os.close(slave)
`
