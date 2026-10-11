package runlife

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBootstrapInvalidOwnerRequestFailsReceiptAfterCleanup(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("supported private descriptor platform")
	}
	for _, tc := range []struct {
		name, request string
		good          bool
	}{
		{"valid", `{"version":1,"kind":"stop","remaining_ns":100000000}` + "\n", true},
		{"malformed", "{broken}\n", false},
		{"oversized", strings.Repeat("x", MaxFrame+1) + "\n", false},
		{"unsupported", `{"version":2,"kind":"stop"}` + "\n", false},
		{"wrong-kind", `{"version":1,"kind":"ready"}` + "\n", false},
		{"closed", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requestR, requestW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			receiptR, receiptW, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer requestR.Close()
			defer requestW.Close()
			defer receiptR.Close()
			defer receiptW.Close()
			active.Lock()
			previous := active.s
			active.Unlock()
			defer func() { active.Lock(); active.s = previous; active.Unlock() }()
			exits := make(chan int, 2)
			s, err := Bootstrap(fmt.Sprintf("%d,%d", requestR.Fd(), receiptW.Fd()), func(code int) { exits <- code }, nil)
			if err != nil {
				t.Fatal(err)
			}
			ready, err := readFrame(receiptR)
			if err != nil || ready.Kind != "ready" {
				t.Fatalf("registration: %+v %v", ready, err)
			}
			cleaned := make(chan struct{})
			if err := s.Bind(func(ctx context.Context) error {
				if _, ok := ctx.Deadline(); !ok {
					t.Error("cleanup lacked bounded deadline")
				}
				close(cleaned)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if tc.request != "" {
				if _, err := requestW.WriteString(tc.request); err != nil {
					t.Fatal(err)
				}
			}
			_ = requestW.Close()
			select {
			case <-cleaned:
			case <-time.After(time.Second):
				t.Fatal("invalid request skipped bound cleanup")
			}
			f, err := readFrame(receiptR)
			if err != nil || f.Kind != "receipt" || (f.Error == "") != tc.good {
				t.Fatalf("request %s cleanup proof: %+v %v", tc.name, f, err)
			}
			select {
			case <-exits:
			case <-time.After(time.Second):
				t.Fatal("request cleanup did not exit")
			}
		})
	}
}

func TestSessionStopClosesAdmissionAndCarriesDeadline(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	requestR, requestW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer requestW.Close()
	exits := make(chan int, 2)
	s := &Session{request: requestR, receipt: w, exit: func(code int) { exits <- code }}
	joined := make(chan struct{})
	if err := s.Bind(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 150*time.Millisecond {
			t.Error("remaining deadline was refreshed")
		}
		close(joined)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	s.stop(100*time.Millisecond, false)
	if s.Admit(func() { t.Error("admitted after stop") }) {
		t.Fatal("admission gate remained open")
	}
	<-joined
	if err := s.Bind(func(context.Context) error { t.Error("late owner admitted"); return nil }); err == nil {
		t.Fatal("stop did not close admission")
	}
	f, err := readFrame(r)
	if err != nil || f.Kind != "receipt" || f.Error != "" {
		t.Fatalf("cleanup receipt: %+v %v", f, err)
	}
	if code := <-exits; code != 0 {
		t.Fatalf("cleanup exit: %d", code)
	}
	s.stop(time.Second, false)
	select {
	case <-exits:
		t.Fatal("repeated shutdown")
	default:
	}
}

func TestPrepareAbortReleasesAllDescriptors(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("supported POSIX descriptors")
	}
	before, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Skip("descriptor inventory unavailable")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for range 100 {
		c, err := Prepare(exec.Command(exe, "run"), nil)
		if err != nil {
			t.Fatal(err)
		}
		c.Abort()
	}
	after, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) > len(before) {
		t.Fatalf("preparation rollback leaked descriptors: before=%d after=%d", len(before), len(after))
	}
}

func TestSessionPolicyLifelineAfterReceiptStillStopsOwnGroup(t *testing.T) {
	requestR, requestW, _ := os.Pipe()
	receiptR, receiptW, _ := os.Pipe()
	defer requestR.Close()
	defer requestW.Close()
	defer receiptR.Close()
	defer receiptW.Close()
	forced, exited := 0, 0
	s := &Session{request: requestR, receipt: receiptW, fallback: func() { forced++ }, exit: func(int) { exited++ }}
	s.Complete(nil)
	if f, err := readFrame(receiptR); err != nil || f.Error != "" {
		t.Fatalf("natural cleanup receipt: %+v %v", f, err)
	}
	s.stop(time.Second, true)
	s.stop(time.Second, true)
	if forced != 1 || exited != 1 {
		t.Fatalf("lost lifetime after receipt: group stops=%d exits=%d", forced, exited)
	}
}

func TestBootstrapNaturalReceiptPreservesPolicyLifeline(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("supported private descriptor platform")
	}
	requestR, requestW, _ := os.Pipe()
	receiptR, receiptW, _ := os.Pipe()
	lifelineR, lifelineW, _ := os.Pipe()
	defer requestR.Close()
	defer requestW.Close()
	defer receiptR.Close()
	defer receiptW.Close()
	defer lifelineR.Close()
	defer lifelineW.Close()
	active.Lock()
	previous := active.s
	active.Unlock()
	defer func() { active.Lock(); active.s = previous; active.Unlock() }()
	forced, exited := make(chan struct{}, 1), make(chan int, 1)
	s, err := Bootstrap(fmt.Sprintf("%d,%d,%d", requestR.Fd(), receiptW.Fd(), lifelineR.Fd()), func(code int) { exited <- code }, func() { forced <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	if f, err := readFrame(receiptR); err != nil || f.Kind != "ready" {
		t.Fatalf("registration: %+v %v", f, err)
	}
	s.Complete(nil)
	if f, err := readFrame(receiptR); err != nil || f.Error != "" {
		t.Fatalf("natural cleanup receipt: %+v %v", f, err)
	}
	select {
	case <-forced:
		t.Fatal("locally closed request confused with supervisor loss")
	case <-time.After(50 * time.Millisecond):
	}
	_ = lifelineW.Close()
	select {
	case <-forced:
	case <-time.After(time.Second):
		t.Fatal("supervisor lifeline erased after natural receipt")
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("lost supervisor did not exit completed runtime")
	}
}

func TestSessionPolicyStopWatchdogRetainsOriginalDeadline(t *testing.T) {
	requestR, requestW, _ := os.Pipe()
	receiptR, receiptW, _ := os.Pipe()
	lifelineR, lifelineW, _ := os.Pipe()
	defer requestR.Close()
	defer requestW.Close()
	defer receiptR.Close()
	defer receiptW.Close()
	defer lifelineR.Close()
	defer lifelineW.Close()
	forced, exited := make(chan struct{}, 1), make(chan int, 1)
	s := &Session{request: requestR, receipt: receiptW, lifeline: lifelineR, fallback: func() { forced <- struct{}{} }, exit: func(code int) { exited <- code }}
	started, release := make(chan struct{}), make(chan struct{})
	if err := s.Bind(func(context.Context) error { close(started); <-release; return nil }); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { s.stop(50*time.Millisecond, false); close(done) }()
	<-started
	s.stop(ShutdownTimeout, true)
	select {
	case <-forced:
	case <-time.After(300 * time.Millisecond):
		close(release)
		t.Fatal("lost supervisor restarted the deadline or skipped policy group fallback")
	}
	if code := <-exited; code != 1 {
		t.Fatalf("watchdog exit %d", code)
	}
	close(release)
	<-done
}
