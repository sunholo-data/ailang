//go:build darwin || linux

package effects

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Python only holds the master. The instrumented Go test process owns every
// host operation, so these controls cover the actual native implementation.
const terminalMasterHolder = `
import os, pty, termios, fcntl, struct, json, sys, select
master, slave = pty.openpty()
def resize(columns, rows):
 fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', rows, columns, 0, 0))
resize(80, 24)
print(json.dumps(os.ttyname(slave)), flush=True)
for line in sys.stdin:
 command = json.loads(line)
 op = command[0]
 if op == 'send':
  os.write(master, bytes.fromhex(command[1])); result = True
 elif op == 'resize':
  resize(command[1], command[2]); result = True
 elif op == 'attrs':
  result = termios.tcgetattr(slave)
  result[3] &= ~getattr(termios, 'PENDIN', 0)
  result[6] = [x if isinstance(x, int) else x[0] for x in result[6]]
 elif op == 'output':
  result = b''
  while select.select([master], [], [], 0.03)[0]:
   result += os.read(master, 65536)
  result = result.hex()
 else:
  raise RuntimeError('unknown holder command: ' + op)
 print(json.dumps(result), flush=True)
`

type terminalTestPTY struct {
	t       *testing.T
	input   io.WriteCloser
	replies *bufio.Scanner
	file    *os.File
	stderr  *bytes.Buffer
}

func newTerminalTestPTY(t *testing.T) *terminalTestPTY {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 required for native terminal PTY controls: %v", err)
	}
	runCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	command := exec.CommandContext(runCtx, python, "-u", "-c", terminalMasterHolder)
	input, err := command.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	output, err := command.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stderr := &bytes.Buffer{}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	holder := &terminalTestPTY{t: t, input: input, replies: bufio.NewScanner(output), stderr: stderr}
	t.Cleanup(func() {
		if holder.file != nil {
			_ = holder.file.Close()
		}
		_ = input.Close()
		waitErr := command.Wait()
		cancel()
		if waitErr != nil {
			t.Errorf("PTY holder: %v: %s", waitErr, stderr.String())
		}
	})
	var path string
	holder.reply(&path)
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	holder.file = file
	return holder
}
func (p *terminalTestPTY) reply(value interface{}) {
	p.t.Helper()
	if !p.replies.Scan() {
		p.t.Fatalf("PTY holder response missing: %v", p.replies.Err())
	}
	if err := json.Unmarshal(p.replies.Bytes(), value); err != nil {
		p.t.Fatal(err)
	}
}
func (p *terminalTestPTY) command(result interface{}, command ...interface{}) {
	p.t.Helper()
	data, err := json.Marshal(command)
	if err != nil {
		p.t.Fatal(err)
	}
	if _, err = fmt.Fprintf(p.input, "%s\n", data); err != nil {
		p.t.Fatal(err)
	}
	p.reply(result)
}
func (p *terminalTestPTY) send(data string) {
	var ok bool
	p.command(&ok, "send", fmt.Sprintf("%x", data))
}
func (p *terminalTestPTY) resize(columns, rows int) {
	var ok bool
	p.command(&ok, "resize", columns, rows)
}
func (p *terminalTestPTY) attrs() []interface{} {
	var attrs []interface{}
	p.command(&attrs, "attrs")
	return attrs
}
func (p *terminalTestPTY) context() *EffContext {
	return &EffContext{IOReader: p.file, IOWriter: p.file, TerminalSignalExit: func(code int) { p.t.Errorf("unexpected process signal %d", code) }}
}
func terminalTestOptions(alternate, hidden bool) eval.Value {
	return &eval.RecordValue{Fields: map[string]eval.Value{"alternate_screen": &eval.BoolValue{Value: alternate}, "hide_cursor": &eval.BoolValue{Value: hidden}}}
}
func requireTerminalCase(t *testing.T, value eval.Value, err error, want string) eval.Value {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	result, ok := value.(*eval.TaggedValue)
	if !ok || len(result.Fields) != 1 {
		t.Fatalf("invalid terminal result %v", value)
	}
	if want == "Ok" {
		if result.CtorName != "Ok" {
			t.Fatalf("wanted Ok, got %v", value)
		}
		return result.Fields[0]
	}
	inner, ok := result.Fields[0].(*eval.TaggedValue)
	if result.CtorName != "Err" || !ok || inner.CtorName != want {
		t.Fatalf("wanted Err(%s), got %v", want, value)
	}
	return inner
}
func requireTerminalEvent(t *testing.T, ctx *EffContext, handle eval.Value, timeout int, want string) eval.Value {
	t.Helper()
	value, err := TerminalReadEvent(ctx, []eval.Value{handle, &eval.IntValue{Value: timeout}})
	event := requireTerminalCase(t, value, err, "Ok")
	if want != "" && event.String() != want {
		t.Fatalf("event %v want %s", event, want)
	}
	return event
}

func TestTerminalNativePTYInfoEventsAndRestoration(t *testing.T) {
	p := newTerminalTestPTY(t)
	ctx := p.context()
	original := p.attrs()
	value, err := TerminalInfo(ctx, nil)
	info := requireTerminalCase(t, value, err, "Ok").(*eval.RecordValue)
	for _, field := range []string{"input_tty", "output_tty", "raw_supported"} {
		if !info.Fields[field].(*eval.BoolValue).Value {
			t.Fatalf("configured TTY %s false", field)
		}
	}
	measured := info.Fields["size"].(*eval.TaggedValue).Fields[0].(*eval.RecordValue)
	if measured.Fields["columns"].(*eval.IntValue).Value != 80 || measured.Fields["rows"].(*eval.IntValue).Value != 24 {
		t.Fatalf("physical size: %v", measured)
	}
	var escaped eval.Value
	ctx.FnCaller = func(_ eval.Value, handle eval.Value) (eval.Value, error) {
		escaped = handle
		raw := p.attrs()
		if reflect.DeepEqual(raw, original) {
			t.Fatal("native scope never changed termios")
		}
		// No Enter: navigation must be delivered directly by the native descriptor.
		p.send("\x1b[A")
		requireTerminalEvent(t, ctx, handle, -1, "Key(Up)")
		requireTerminalEvent(t, ctx, handle, 5, "Idle")
		p.send("\x1b")
		requireTerminalEvent(t, ctx, handle, -1, "Key(Escape)")
		p.send("\xe2\x82")
		requireTerminalEvent(t, ctx, handle, 0, "Idle")
		p.send("\xac")
		requireTerminalEvent(t, ctx, handle, 100, "Key(Text(€))")
		p.resize(13, 4)
		resize := requireTerminalEvent(t, ctx, handle, 100, "").(*eval.TaggedValue)
		if resize.CtorName != "Resize" {
			t.Fatalf("wanted physical resize: %v", resize)
		}
		size := resize.Fields[0].(*eval.RecordValue)
		if size.Fields["columns"].(*eval.IntValue).Value != 13 || size.Fields["rows"].(*eval.IntValue).Value != 4 {
			t.Fatalf("resize was clamped: %v", size)
		}
		p.send("\x04")
		requireTerminalEvent(t, ctx, handle, 100, "EndOfInput")
		return &eval.IntValue{Value: 17}, nil
	}
	value, err = TerminalWith(ctx, []eval.Value{terminalTestOptions(true, true), &eval.UnitValue{}})
	if result := requireTerminalCase(t, value, err, "Ok"); result.String() != "17" {
		t.Fatalf("callback result changed: %v", result)
	}
	if restored := p.attrs(); !reflect.DeepEqual(original, restored) {
		t.Fatalf("termios not restored: before %v after %v", original, restored)
	}
	var output string
	p.command(&output, "output")
	for _, sequence := range []string{"1b5b3f3130343968", "1b5b3f32356c", "1b5b3f323568", "1b5b3f313034396c"} {
		if !strings.Contains(output, sequence) {
			t.Fatalf("missing acquisition/restoration escape %s in %s", sequence, output)
		}
	}
	value, err = TerminalReadEvent(ctx, []eval.Value{escaped, &eval.IntValue{Value: 0}})
	requireTerminalCase(t, value, err, "InvalidSession")
}

func TestTerminalNativePTYOwnership(t *testing.T) {
	p := newTerminalTestPTY(t)
	ctx := p.context()
	callback := &eval.UnitValue{}
	ctx.FnCaller = func(_ eval.Value, handle eval.Value) (eval.Value, error) {
		value, err := TerminalWith(ctx, []eval.Value{terminalTestOptions(false, false), callback})
		requireTerminalCase(t, value, err, "Busy")
		clone := ctx.Clone().(*EffContext)
		value, err = TerminalWith(clone, []eval.Value{terminalTestOptions(false, false), callback})
		requireTerminalCase(t, value, err, "Busy")
		value, err = TerminalReadEvent(clone, []eval.Value{handle, &eval.IntValue{Value: 0}})
		requireTerminalCase(t, value, err, "InvalidSession")
		if _, err := ioReadLine(ctx, nil); err == nil {
			t.Fatal("line reader stole native input")
		}
		return &eval.UnitValue{}, nil
	}
	value, err := TerminalWith(ctx, []eval.Value{terminalTestOptions(false, false), callback})
	requireTerminalCase(t, value, err, "Ok")
	// A line reader's unread input remains protected even in an independent clone.
	p.send("first\nsecond\n")
	// Canonical kernels may return one line per physical read. Explicitly fill
	// the persistent buffer to exercise already-owned read-ahead on both hosts.
	if _, err := ctx.GetIOReader().Peek(len("first\nsecond\n")); err != nil {
		t.Fatal(err)
	}
	if value, err := ioReadLine(ctx, nil); err != nil || value.String() != "first" {
		t.Fatalf("line read %v %v", value, err)
	}
	value, err = TerminalWith(ctx, []eval.Value{terminalTestOptions(false, false), callback})
	requireTerminalCase(t, value, err, "Busy")
	value, err = TerminalWith(ctx.Clone().(*EffContext), []eval.Value{terminalTestOptions(false, false), callback})
	requireTerminalCase(t, value, err, "Busy")
	if value, err := ioReadLine(ctx, nil); err != nil || value.String() != "second" {
		t.Fatalf("buffer drain %v %v", value, err)
	}
	ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) { return &eval.UnitValue{}, nil }
	value, err = TerminalWith(ctx, []eval.Value{terminalTestOptions(false, false), callback})
	requireTerminalCase(t, value, err, "Ok")
}

func TestTerminalNativePTYCallbackFailuresRestore(t *testing.T) {
	for _, outcome := range []string{"error", "panic", "exit"} {
		t.Run(outcome, func(t *testing.T) {
			p := newTerminalTestPTY(t)
			ctx := p.context()
			before := p.attrs()
			primary := errors.New("application callback failure")
			ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) {
				switch outcome {
				case "error":
					return nil, primary
				case "panic":
					panic("application panic")
				default:
					panic(&eval.EvalExitCode{Code: 7})
				}
			}
			var result eval.Value
			var callErr error
			var recovered interface{}
			func() {
				defer func() { recovered = recover() }()
				result, callErr = TerminalWith(ctx, []eval.Value{terminalTestOptions(true, true), &eval.UnitValue{}})
			}()
			switch outcome {
			case "error":
				if result != nil || !errors.Is(callErr, primary) {
					t.Fatalf("callback error replaced: %v %v", result, callErr)
				}
			case "panic":
				if recovered != "application panic" {
					t.Fatalf("panic replaced: %#v", recovered)
				}
			case "exit":
				code, ok := recovered.(*eval.EvalExitCode)
				if !ok || code.Code != 7 {
					t.Fatalf("exit replaced: %#v", recovered)
				}
			}
			if after := p.attrs(); !reflect.DeepEqual(before, after) {
				t.Fatalf("callback %s left terminal attributes changed", outcome)
			}
		})
	}
}

func TestTerminalNativePTYEndpointAndQueryErrors(t *testing.T) {
	p := newTerminalTestPTY(t)
	ctx := p.context()
	options := terminalTestOptions(false, false)
	neverCalled := func(eval.Value, eval.Value) (eval.Value, error) {
		t.Fatal("invalid acquisition invoked callback")
		return nil, nil
	}
	value, err := TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "Unsupported") // missing callback dispatcher
	ctx.FnCaller = neverCalled
	ctx.IOReader = strings.NewReader("")
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "NotTTY")
	input, output, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	defer output.Close()
	ctx.IOReader = input
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "NotTTY")
	ctx.IOReader = p.file
	ctx.IOWriter = &bytes.Buffer{}
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "NotTTY")
	regular, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer regular.Close()
	ctx.IOWriter = regular
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "NotTTY")
	ctx.IOReader = strings.NewReader("")
	value, err = TerminalInfo(ctx, nil)
	info := requireTerminalCase(t, value, err, "Ok").(*eval.RecordValue)
	if info.Fields["input_tty"].(*eval.BoolValue).Value || info.Fields["output_tty"].(*eval.BoolValue).Value || info.Fields["size"].String() != "None" {
		t.Fatalf("redirected endpoints misreported: %v", info)
	}
	ctx.IOReader = p.file
	ctx.IOWriter = p.file
	ctx.TerminalSignalExit = nil
	value, err = TerminalInfo(ctx, nil)
	info = requireTerminalCase(t, value, err, "Ok").(*eval.RecordValue)
	if info.Fields["raw_supported"].(*eval.BoolValue).Value {
		t.Fatal("unintegrated host advertises raw acquisition")
	}
	ctx.TerminalSignalExit = func(int) { t.Fatal("unexpected signal") }
	before := p.attrs()
	p.resize(0, 0)
	value, err = TerminalInfo(ctx, nil)
	requireTerminalCase(t, value, err, "QueryFailure")
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "QueryFailure")
	if after := p.attrs(); !reflect.DeepEqual(before, after) {
		t.Fatal("failed initial size query left terminal raw")
	}
	p.resize(80, 24)
	release, err := ctx.beginInputRead()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	value, err = TerminalWith(ctx, []eval.Value{options, &eval.UnitValue{}})
	requireTerminalCase(t, value, err, "Busy")
	release()
}

type terminalPTYFailingWriter struct {
	file               *os.File
	writeErr, flushErr error
}

func (w *terminalPTYFailingWriter) Fd() uintptr { return w.file.Fd() }
func (w *terminalPTYFailingWriter) Write(data []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.file.Write(data)
}
func (w *terminalPTYFailingWriter) Flush() error { return w.flushErr }

func TestTerminalNativePTYOutputFailuresRestore(t *testing.T) {
	for _, failure := range []string{"activation write", "activation flush", "cleanup"} {
		t.Run(failure, func(t *testing.T) {
			p := newTerminalTestPTY(t)
			ctx := p.context()
			before := p.attrs()
			problem := errors.New("terminal output unavailable")
			writer := &terminalPTYFailingWriter{file: p.file}
			ctx.IOWriter = writer
			if failure == "activation write" {
				writer.writeErr = problem
			}
			if failure == "activation flush" {
				writer.flushErr = problem
			}
			called := false
			ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) {
				called = true
				writer.writeErr = problem
				return &eval.UnitValue{}, nil
			}
			value, err := TerminalWith(ctx, []eval.Value{terminalTestOptions(true, true), &eval.UnitValue{}})
			expected := "InputFailure"
			if failure == "cleanup" {
				expected = "CleanupFailure"
			}
			requireTerminalCase(t, value, err, expected)
			if called != (failure == "cleanup") {
				t.Fatalf("unexpected callback invocation: %v", called)
			}
			if after := p.attrs(); !reflect.DeepEqual(before, after) {
				t.Fatalf("%s failed to restore terminal", failure)
			}
		})
	}
}

func TestTerminalNativePTYMalformedEvents(t *testing.T) {
	for _, input := range []string{"\xff", "\x1b[99h", "\x1b[" + strings.Repeat("1", 65) + "h"} {
		t.Run(fmt.Sprintf("bytes_%x", input[:min(3, len(input))]), func(t *testing.T) {
			p := newTerminalTestPTY(t)
			ctx := p.context()
			ctx.FnCaller = func(_ eval.Value, handle eval.Value) (eval.Value, error) {
				p.send(input)
				value, err := TerminalReadEvent(ctx, []eval.Value{handle, &eval.IntValue{Value: 100}})
				requireTerminalCase(t, value, err, "InputFailure")
				return &eval.UnitValue{}, nil
			}
			value, err := TerminalWith(ctx, []eval.Value{terminalTestOptions(false, false), &eval.UnitValue{}})
			requireTerminalCase(t, value, err, "Ok")
		})
	}
}

func TestTerminalPOSIXInvalidDescriptorAndPolling(t *testing.T) {
	if _, err := terminalDeviceKey(-1); err == nil {
		t.Fatal("invalid descriptor identity accepted")
	}
	if _, err := terminalActivate(-1, -1); err == nil {
		t.Fatal("invalid descriptor activated")
	}
	if err := terminalRetainSignals(-1); err == nil {
		t.Fatal("invalid descriptor termios accepted")
	}
	p := newTerminalTestPTY(t)
	device, err := terminalActivate(int(p.file.Fd()), int(p.file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = device.restore() }()
	if n, err := device.read(make([]byte, 1)); err != nil || n != 0 {
		t.Fatalf("empty noncanonical descriptor read: %d %v", n, err)
	}
	if err := device.restore(); err != nil {
		t.Fatal(err)
	}
	if err := p.file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := device.poll(0); err == nil {
		t.Fatal("closed descriptor poll accepted")
	}
	if _, err := device.read(make([]byte, 1)); err == nil {
		t.Fatal("closed descriptor read accepted")
	}
}
