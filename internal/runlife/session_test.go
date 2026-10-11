package runlife

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

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
