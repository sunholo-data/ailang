package runlife

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func testPipeChild(t *testing.T) (*Child, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	requestR, requestW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	c := &Child{request: requestW, receipt: r, received: make(chan struct{}), waited: make(chan struct{})}
	c.AfterStart()
	t.Cleanup(func() { c.Abort(); w.Close(); requestR.Close() })
	return c, w
}

func TestChildRequiresPrivateTerminalReceipt(t *testing.T) {
	for _, body := range []string{"", `{"version":1,"kind":"receipt"}` + "\n", `{"version":1,"kind":"ready"}` + "\n", `{"version":1,"kind":"ready"}` + "\n" + strings.Repeat("x", 1025) + "\n", `{"version":1,"kind":"ready"}` + "\n" + `{"version":1,"kind":"receipt","error":"incomplete"}` + "\n"} {
		c, w := testPipeChild(t)
		_, _ = w.Write([]byte(body))
		_ = w.Close()
		c.AfterWait()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := c.Join(ctx)
		cancel()
		if err == nil {
			t.Errorf("false successful receipt for %q", body)
		}
	}
	c, w := testPipeChild(t)
	if err := writeFrame(w, frame{Kind: "ready"}); err != nil {
		t.Fatal(err)
	}
	if err := writeFrame(w, frame{Kind: "receipt"}); err != nil {
		t.Fatal(err)
	}
	w.Close()
	c.AfterWait()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Join(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestChildForceDeadlineAndSingleStop(t *testing.T) {
	c, w := testPipeChild(t)
	_ = writeFrame(w, frame{Kind: "ready"})
	forced := make(chan struct{}, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	for range 3 {
		c.RequestStop(ctx, func() error { forced <- struct{}{}; return nil })
	}
	if err := c.Join(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing receipt: %v", err)
	}
	select {
	case <-forced:
	case <-time.After(time.Second):
		t.Fatal("deadline did not force owned group")
	}
	select {
	case <-forced:
		t.Fatal("repeated force termination")
	default:
	}
}
