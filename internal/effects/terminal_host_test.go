package effects

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
)

type fakeTerminalDevice struct {
	input                []byte
	columns, rows        int
	restores             int
	queryErr, restoreErr error
}

func (d *fakeTerminalDevice) read(b []byte) (int, error) {
	if len(d.input) == 0 {
		return 0, io.EOF
	}
	n := copy(b, d.input)
	d.input = d.input[n:]
	return n, nil
}
func (d *fakeTerminalDevice) poll(int) (bool, error)  { return len(d.input) > 0, nil }
func (d *fakeTerminalDevice) size() (int, int, error) { return d.columns, d.rows, d.queryErr }
func (d *fakeTerminalDevice) restore() error          { d.restores++; return d.restoreErr }
func fakeTerminalContext(input []byte) (*EffContext, *terminalSession, *fakeTerminalDevice) {
	ctx := &EffContext{IOWriter: &bytes.Buffer{}}
	state := ctx.inputState()
	device := &fakeTerminalDevice{input: input, columns: 80, rows: 24}
	session := &terminalSession{id: 123, ctx: ctx, state: state, device: device, output: ctx.IOWriter, columns: 80, rows: 24, stop: make(chan struct{}), release: func() {}, hidden: true, alternate: true}
	state.active = session
	return ctx, session, device
}
func readFakeTerminal(t *testing.T, ctx *EffContext, timeout int) eval.Value {
	t.Helper()
	value, err := TerminalReadEvent(ctx, []eval.Value{terminalTag("TerminalSession", &eval.IntValue{Value: 123}), &eval.IntValue{Value: timeout}})
	if err != nil {
		t.Fatal(err)
	}
	return value
}
func TestTerminalSessionOwnershipAndRestoration(t *testing.T) {
	ctx, session, device := fakeTerminalContext([]byte("\x1b[B"))
	if got := readFakeTerminal(t, ctx.WithBudget(nil), 0).String(); got != "Ok(Key(Down))" {
		t.Fatal(got)
	}
	cloned := ctx.Clone().(*EffContext)
	if got := readFakeTerminal(t, cloned, 0).String(); !strings.Contains(got, "InvalidSession") {
		t.Fatal(got)
	}
	if _, err := ioReadLine(ctx, nil); err == nil {
		t.Fatal("readLine stole leased input")
	}
	if _, err := ioReadLineOpt(ctx, nil); err == nil {
		t.Fatal("readLineOpt stole leased input")
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	if device.restores != 1 {
		t.Fatalf("restored %d times", device.restores)
	}
	if got := ctx.IOWriter.(*bytes.Buffer).String(); got != "\x1b[?25h\x1b[?1049l" {
		t.Fatalf("cleanup bytes %q", got)
	}
	if got := readFakeTerminal(t, ctx, 0).String(); !strings.Contains(got, "InvalidSession") {
		t.Fatal(got)
	}
}
func TestTerminalResizeIdleEOFAndQueryFailure(t *testing.T) {
	ctx, session, device := fakeTerminalContext(nil)
	defer session.close()
	device.columns = 12
	device.rows = 3
	got := readFakeTerminal(t, ctx, 0).(*eval.TaggedValue).Fields[0].(*eval.TaggedValue)
	if got.CtorName != "Resize" {
		t.Fatalf("wanted resize got %v", got)
	}
	if got := readFakeTerminal(t, ctx, 0).String(); got != "Ok(Idle)" {
		t.Fatal(got)
	}
	session.eof = true
	if got := readFakeTerminal(t, ctx, 0).String(); got != "Ok(EndOfInput)" {
		t.Fatal(got)
	}
	session.eof = false
	device.queryErr = errors.New("size query failed")
	if got := readFakeTerminal(t, ctx, 0).String(); !strings.Contains(got, "QueryFailure") {
		t.Fatal(got)
	}
}
func TestTerminalCleanupFailureStillRevokes(t *testing.T) {
	ctx, session, device := fakeTerminalContext(nil)
	device.restoreErr = errors.New("restore failed")
	if err := session.close(); err == nil {
		t.Fatal("restore failure hidden")
	}
	if ctx.inputState().active != nil {
		t.Fatal("failed restore retained live handle")
	}
}
func TestTerminalUnintegratedHostRejected(t *testing.T) {
	ctx := &EffContext{IOReader: strings.NewReader(""), IOWriter: &bytes.Buffer{}}
	opts := &eval.RecordValue{Fields: map[string]eval.Value{"alternate_screen": &eval.BoolValue{}, "hide_cursor": &eval.BoolValue{}}}
	got, err := TerminalWith(ctx, []eval.Value{opts, &eval.UnitValue{}})
	if err != nil || !strings.Contains(got.String(), "Unsupported") {
		t.Fatalf("%v %v", got, err)
	}
}
func TestTerminalDecoderSplitDeadlineAndNavigation(t *testing.T) {
	for _, pair := range [][2]string{{"\x1b[A", "Up"}, {"\x1b[B", "Down"}, {"\x1b[C", "Right"}, {"\x1b[D", "Left"}, {"\x1bOH", "Home"}, {"\x1bOF", "End"}, {"\x1b[5~", "PageUp"}, {"\x1b[6~", "PageDown"}, {"\x1b[3~", "Delete"}, {"\n", "Enter"}, {"\t", "Tab"}, {"\x7f", "Backspace"}} {
		for split := 1; split <= len(pair[0]); split++ {
			d := &terminalDecoder{}
			now := time.Now()
			if err := d.feed([]byte(pair[0][:split]), now); err != nil {
				t.Fatal(err)
			}
			if split < len(pair[0]) {
				if ev, err := d.next(now, false); ev != nil || err != nil {
					t.Fatalf("premature %q split %d: %v %v", pair[0], split, ev, err)
				}
				if err := d.feed([]byte(pair[0][split:]), now.Add(10*time.Millisecond)); err != nil {
					t.Fatal(err)
				}
			}
			ev, err := d.next(now.Add(10*time.Millisecond), false)
			if err != nil || ev.String() != "Key("+pair[1]+")" {
				t.Fatalf("%q split %d: %v %v", pair[0], split, ev, err)
			}
		}
	}
	d := &terminalDecoder{}
	now := time.Now()
	_ = d.feed([]byte("\x1b"), now)
	_ = d.feed([]byte("["), now.Add(20*time.Millisecond))
	if _, err := d.next(now.Add(31*time.Millisecond), false); err == nil {
		t.Fatal("split reads reset escape deadline")
	}
}
func TestTerminalLineReadReservation(t *testing.T) {
	ctx := &EffContext{IOReader: strings.NewReader("\n")}
	release, err := ctx.beginInputRead()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ctx.beginInputRead(); err == nil {
		t.Fatal("concurrent reader accepted")
	}
	release()
	if value, err := ioReadLineOpt(ctx, nil); err != nil || value.String() != "Some()" {
		t.Fatalf("%v %v", value, err)
	}
}

func TestTerminalScopeRestoresOnCallbackOutcomes(t *testing.T) {
	callbackError := errors.New("callback failed")
	budgetError := NewBudgetExhaustedError("IO", 1, 1, "test", 1)
	outcomes := []struct {
		name      string
		call      func(eval.Value, eval.Value) (eval.Value, error)
		wantPanic interface{}
		wantErr   error
	}{
		{name: "normal", call: func(eval.Value, eval.Value) (eval.Value, error) { return &eval.IntValue{Value: 7}, nil }},
		{name: "callback error", call: func(eval.Value, eval.Value) (eval.Value, error) { return nil, callbackError }, wantErr: callbackError},
		{name: "budget exhaustion", call: func(eval.Value, eval.Value) (eval.Value, error) { return nil, budgetError }, wantErr: budgetError},
		{name: "panic", call: func(eval.Value, eval.Value) (eval.Value, error) { panic("callback panic") }, wantPanic: "callback panic"},
		{name: "exit", call: func(eval.Value, eval.Value) (eval.Value, error) { panic(&eval.EvalExitCode{Code: 7}) }, wantPanic: "exit"},
	}
	for _, outcome := range outcomes {
		t.Run(outcome.name, func(t *testing.T) {
			ctx, session, device := fakeTerminalContext(nil)
			ctx.FnCaller = outcome.call
			var result eval.Value
			var callErr error
			var recovered interface{}
			func() { defer func() { recovered = recover() }(); result, callErr = session.run(&eval.UnitValue{}) }()
			if outcome.wantPanic == "exit" {
				code, ok := recovered.(*eval.EvalExitCode)
				if !ok || code.Code != 7 {
					t.Fatalf("exit sentinel changed: %#v", recovered)
				}
			} else if outcome.wantPanic != nil {
				if recovered != outcome.wantPanic {
					t.Fatalf("panic changed: %#v", recovered)
				}
			} else if recovered != nil {
				t.Fatalf("unexpected panic: %#v", recovered)
			}
			if !errors.Is(callErr, outcome.wantErr) {
				t.Fatalf("callback error changed: %v", callErr)
			}
			if outcome.name == "normal" && result.String() != "Ok(7)" {
				t.Fatalf("callback result changed: %v", result)
			}
			if device.restores != 1 || ctx.inputState().active != nil {
				t.Fatal("scope did not restore/revoke exactly once")
			}
			if bytes := ctx.IOWriter.(*bytes.Buffer).String(); !strings.HasSuffix(bytes, "\x1b[?25h\x1b[?1049l") {
				t.Fatalf("cleanup output missing %q", bytes)
			}
		})
	}
}
func TestTerminalScopePreservesCallbackErrorDuringCleanupFailure(t *testing.T) {
	ctx, session, device := fakeTerminalContext(nil)
	callbackError := errors.New("callback primary")
	device.restoreErr = errors.New("cleanup secondary")
	ctx.FnCaller = func(eval.Value, eval.Value) (eval.Value, error) { return nil, callbackError }
	result, err := session.run(&eval.UnitValue{})
	if result != nil || !errors.Is(err, callbackError) || !errors.Is(err, device.restoreErr) {
		t.Fatalf("lost original/secondary errors: %v %v", result, err)
	}
}

func TestTerminalSignalRestoresAllActiveDevices(t *testing.T) {
	_, first, firstDevice := fakeTerminalContext(nil)
	_, second, secondDevice := fakeTerminalContext(nil)
	registerTerminalSession(first)
	registerTerminalSession(second)
	if err := restoreActiveTerminals(); err != nil {
		t.Fatal(err)
	}
	if firstDevice.restores != 1 || secondDevice.restores != 1 {
		t.Fatal("process termination left another terminal active")
	}
}

func TestTerminalDeviceAliasesAndCloneReaderExclusion(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "terminal-device")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	alias, err := os.Open(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer alias.Close()
	first := &EffContext{IOReader: file}
	release, err := first.beginInputRead()
	if err != nil {
		t.Fatal(err)
	}
	defer release() // always relinquish the first reservation, including assertion failures
	second := &EffContext{IOReader: alias}
	if stolenRelease, err := second.beginInputRead(); err == nil {
		stolenRelease()
		t.Fatal("descriptor alias bypassed device lease")
	}
	clone := first.Clone().(*EffContext)
	if stolenRelease, err := clone.beginInputRead(); err == nil {
		stolenRelease()
		t.Fatal("clone bypassed device lease")
	}
	release()
	secondRelease, err := second.beginInputRead()
	if err != nil {
		t.Fatal(err)
	}
	secondRelease()
}
func TestTerminalAsyncStdinOwnsUntilReaderStops(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	ctx := &EffContext{IOReader: input}
	release, err := ctx.beginInputRead()
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan struct{})
	source := newOwnedStdinSource(ctx.GetIOReader(), "stdin", 0, func() { release(); close(finished) })
	source.Close()
	if _, err := ioReadLineOpt(ctx, nil); err == nil {
		t.Fatal("closed blocked async source released stdin prematurely")
	}
	_ = writer.Close()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("reader exit did not release device")
	}
	if value, err := ioReadLineOpt(ctx, nil); err != nil || value.String() != "None" {
		t.Fatalf("%v %v", value, err)
	}
}
func TestTerminalAsyncCannotStealNativeInput(t *testing.T) {
	ctx, session, _ := fakeTerminalContext(nil)
	defer session.close()
	ctx.Stream = NewStreamContext()
	if _, err := StreamAsyncReadStdinLines(ctx, []eval.Value{&eval.StringValue{Value: "stdin"}, &eval.IntValue{Value: 0}}); err == nil {
		t.Fatal("async stdin stole terminal lease")
	}
}

func TestTerminalUnreadBufferAcrossContexts(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "terminal-buffer")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.WriteString("first\nsecond\n"); err != nil {
		t.Fatal(err)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	ctx := &EffContext{IOReader: file}
	if _, err = ioReadLine(ctx, nil); err != nil {
		t.Fatal(err)
	}
	clone := ctx.Clone().(*EffContext)
	if buffered, err := terminalBufferedInput(clone.inputEndpoint()); err != nil || !buffered {
		t.Fatalf("clone could bypass buffered line input: %v %v", buffered, err)
	}
	if _, err = ioReadLine(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if buffered, err := terminalBufferedInput(clone.inputEndpoint()); err != nil || buffered {
		t.Fatalf("buffer reservation not released after drain: %v %v", buffered, err)
	}
}

func TestTerminalQueuedSignalSurvivesNormalClose(t *testing.T) {
	// Both select arms are ready before the watcher starts. Repetition catches
	// the former random stop-arm choice swallowing an already-diverted signal.
	for iteration := 0; iteration < 256; iteration++ {
		ctx, session, device := fakeTerminalContext(nil)
		_, other, otherDevice := fakeTerminalContext(nil)
		registerTerminalSession(other)
		exits := make(chan int, 2)
		ctx.TerminalSignalExit = func(code int) { exits <- code }
		session.signals = make(chan os.Signal, 2)
		expected := 130
		if iteration%2 == 0 {
			session.signals <- os.Interrupt
		} else {
			session.signals <- syscall.SIGTERM
			expected = 143
		}
		session.signals <- os.Interrupt
		if err := session.close(); err != nil {
			t.Fatal(err)
		}
		session.watchSignals()
		select {
		case <-session.signalDone:
		case <-time.After(time.Second):
			t.Fatal("signal watcher did not join")
		}
		select {
		case code := <-exits:
			if code != expected {
				t.Fatalf("iteration %d: exit %d want %d", iteration, code, expected)
			}
		default:
			_ = other.close()
			t.Fatalf("iteration %d: queued termination signal swallowed by normal close", iteration)
		}
		select {
		case code := <-exits:
			t.Fatalf("duplicate termination callback: %d", code)
		default:
		}
		if device.restores != 1 || otherDevice.restores != 1 {
			t.Fatalf("termination ran before every device restored: %d/%d", device.restores, otherDevice.restores)
		}
	}
}

func TestTerminalAsyncClosePreservesUnemittedReadAhead(t *testing.T) {
	var input strings.Builder
	for line := 0; line < 102; line++ {
		fmt.Fprintf(&input, "line%d\n", line)
	}
	ctx := &EffContext{IOReader: strings.NewReader(input.String()), Stream: NewStreamContext()}
	handle, err := StreamAsyncReadStdinLines(ctx, []eval.Value{&eval.StringValue{Value: "stdin"}, &eval.IntValue{Value: 0}})
	if err != nil {
		t.Fatal(err)
	}
	id, err := extractSourceID(handle)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := ctx.Stream.GetSource(id)
	deadline := time.Now().Add(time.Second)
	for len(source.Events()) < 100 && time.Now().Before(deadline) {
		runtime.Gosched()
	}
	if len(source.Events()) != 100 {
		source.Close()
		t.Fatal("source never filled event queue")
	}
	source.Close()
	for range source.Events() {
	} // These 100 events were already emitted/source-owned.
	for {
		ctx.inputState().mu.Lock()
		reading := ctx.inputState().reading
		ctx.inputState().mu.Unlock()
		if !reading {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("closed source did not release reader")
		}
		runtime.Gosched()
	}
	for _, want := range []string{"Some(line100)", "Some(line101)", "None"} {
		got, err := ioReadLineOpt(ctx, nil)
		if err != nil || got.String() != want {
			t.Fatalf("unemitted input discarded: %v %v want %s", got, err, want)
		}
	}
}

func TestTerminalOwnedAsyncBlankAndFinalPartial(t *testing.T) {
	ctx := &EffContext{IOReader: strings.NewReader("\nfinal"), Stream: NewStreamContext()}
	handle, err := StreamAsyncReadStdinLines(ctx, []eval.Value{&eval.StringValue{Value: "stdin"}, &eval.IntValue{Value: 0}})
	if err != nil {
		t.Fatal(err)
	}
	id, err := extractSourceID(handle)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := ctx.Stream.GetSource(id)
	var lines []string
	for event := range source.Events() {
		lines = append(lines, event.text)
	}
	if len(lines) != 2 || lines[0] != "" || lines[1] != "final" {
		t.Fatalf("blank/partial lines changed: %#v", lines)
	}
}
func TestTerminalPrependExposesAllExistingBufferedInput(t *testing.T) {
	ctx := &EffContext{IOReader: strings.NewReader("second\nthird\n")}
	reader := ctx.GetIOReader()
	if _, err := reader.Peek(1); err != nil {
		t.Fatal(err)
	}
	ctx.prependIOInput("first\n")
	if got := ctx.GetIOReader().Buffered(); got != len("first\nsecond\nthird\n") {
		t.Fatalf("hidden read-ahead after prepend: %d", got)
	}
	for _, want := range []string{"Some(first)", "Some(second)", "Some(third)", "None"} {
		value, err := ioReadLineOpt(ctx, nil)
		if err != nil || value.String() != want {
			t.Fatalf("%v %v want %s", value, err, want)
		}
	}
}

func TestTerminalHostRejectsInvalidArgumentShapes(t *testing.T) {
	cases := []struct {
		name string
		call func(*EffContext, []eval.Value) (eval.Value, error)
		args []eval.Value
	}{
		{"info arguments", TerminalInfo, []eval.Value{&eval.UnitValue{}}},
		{"scope arguments", TerminalWith, nil},
		{"scope options", TerminalWith, []eval.Value{&eval.UnitValue{}, &eval.UnitValue{}}},
		{"scope missing booleans", TerminalWith, []eval.Value{&eval.RecordValue{Fields: map[string]eval.Value{}}, &eval.UnitValue{}}},
		{"event arguments", TerminalReadEvent, nil},
		{"event timeout", TerminalReadEvent, []eval.Value{&eval.UnitValue{}, &eval.StringValue{Value: "10"}}},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if _, err := item.call(&EffContext{}, item.args); err == nil {
				t.Fatal("invalid host argument shape accepted")
			}
		})
	}
	for _, handle := range []eval.Value{
		&eval.UnitValue{},
		terminalTag("TerminalSession"),
		terminalTag("TerminalSession", &eval.StringValue{Value: "1"}),
		terminalTag("Idle", &eval.IntValue{Value: 1}),
	} {
		value, err := TerminalReadEvent(&EffContext{}, []eval.Value{handle, &eval.IntValue{Value: 0}})
		if err != nil || !strings.Contains(value.String(), "InvalidSession") {
			t.Fatalf("invalid handle %v: %v %v", handle, value, err)
		}
	}
}
