package effects

import (
	"github.com/sunholo-data/ailang/internal/eval"
	"testing"
	"time"
)

func TestSelectEventsClosedSourceDoesNotConsumeOtherSourceOutput(t *testing.T) {
	closed := newMockSource("closed-stdin", 10, 1)
	closed.Close()
	active := newMockSource("worker", 5, 0)
	const messages = 5000
	go func() {
		defer active.Close()
		for i := 0; i < messages; i++ {
			active.Send(streamEvent{kind: "source_text", sourceName: "worker", text: "final-output"})
		}
	}()
	handler := &testHandler{}
	if err := selectEventsLoop([]EventSource{closed, active}, &eval.UnitValue{}, mockFnCaller(handler), time.Second, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if handler.count != messages {
		t.Fatalf("closed-source probing lost output: got %d, want %d", handler.count, messages)
	}
}
