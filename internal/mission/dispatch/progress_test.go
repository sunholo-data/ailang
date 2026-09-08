package dispatch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/sunholo-data/ailang/internal/executor"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgressExactBoundedPrivate(t *testing.T) {
	var events []Event
	p := newProgressObserver("digest", func(e Event) error { events = append(events, e); return nil })
	p.OnToolUse("read", "secret-one")
	p.OnToolUse("read", "secret-two")
	p.OnToolUse("read", "secret-one")
	p.OnToolResult("read", "secret-output")
	got := p.snapshot()
	if got.ToolCalls != 3 || got.RepeatedCalls != 1 || got.LastCompletedTool != "read" || got.LastProgressTime.IsZero() {
		t.Fatalf("%+v", got)
	}
	b, _ := json.Marshal(events)
	if strings.Contains(string(b), "secret-") {
		t.Fatal("leaked tool content")
	}
	for i := 0; i < maxTrackedCalls+10; i++ {
		p.OnToolUse("read", strings.Repeat("a", i))
	}
	if len(p.calls) > maxTrackedCalls || !p.snapshot().TrackingTruncated {
		t.Fatal("tracking not bounded")
	}
}

type progressExecutor struct{ *fakeExecutor }

func (e *progressExecutor) ExecuteStreaming(ctx context.Context, task *executor.Task, handler executor.EventHandler) (*executor.Result, error) {
	handler.OnToolUse("read", "private-input")
	handler.OnToolResult("read", "private-output")
	handler.OnToolUse("read", "private-input")
	return e.fakeExecutor.ExecuteStreaming(ctx, task, handler)
}

type progressFactory struct{ executor.Executor }

func (f progressFactory) GetExecutor(string) (executor.Executor, error) { return f.Executor, nil }

func TestProgressSurvivesJournalReopen(t *testing.T) {
	req, runner, f, _ := fixture(t)
	req.Models = []string{"judge"}
	runner.Executors = progressFactory{&progressExecutor{f["pi"]}}
	path := filepath.Join(t.TempDir(), "receipt.jsonl")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	runner.Record = journal.Record
	report, err := runner.Run(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	if report.Progress == nil || report.Progress.RepeatedCalls != 1 {
		t.Fatalf("missing final progress: %+v", report)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "private-input") || strings.Contains(string(body), "private-output") {
		t.Fatal("leaked content")
	}
	var last *Progress
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Kind == "progress" {
			last = event.Report.Progress
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if last == nil || last.ToolCalls != 2 || last.LastCompletedTool != "read" {
		t.Fatalf("progress not durable: %+v", last)
	}
}

func TestProgressRecorderFailureIsVisible(t *testing.T) {
	req, runner, f, _ := fixture(t)
	req.Models = []string{"judge"}
	runner.Executors = progressFactory{&progressExecutor{f["pi"]}}
	runner.Record = func(event Event) error {
		if event.Kind == "progress" {
			return errors.New("progress storage failed")
		}
		return nil
	}
	report, err := runner.Run(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "progress storage failed") || report.Status != "execution_failed" {
		t.Fatalf("silent progress loss: %+v %v", report, err)
	}
}
