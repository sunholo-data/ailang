//go:build !windows

package agy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// A killed agy must be proven gone before a cloud job may write back or release
// its rotating token. A grandchild that escaped the process group (its own session)
// and still holds stdout is "not proven gone": the run returns the shared sentinel,
// which the cloud entry point maps to quarantine.
func TestTimeout_EscapedGrandchildIsTerminationUnconfirmed(t *testing.T) {
	setupHomes(t)
	old := terminationGrace
	terminationGrace = 500 * time.Millisecond
	t.Cleanup(func() { terminationGrace = old })

	// perl setsid(): a new session, so the group kill cannot reach it; it inherits
	// stdout and sleeps on it.
	escape := `perl -e 'use POSIX qw(setsid); setsid(); sleep 20' &
sleep 30`
	e := newExec(t, newMockAgy(t, ExpectedVersion, "", 0, escape))
	task := &executor.Task{ID: "t1", Directive: "x", Workspace: t.TempDir(), Model: "gemini-4-neon-high", Timeout: 1500 * time.Millisecond}
	res, err := e.Execute(context.Background(), task)
	if !errors.Is(err, executor.ErrProcessTerminationUnconfirmed) {
		t.Fatalf("escaped grandchild: err=%v res=%+v, want ErrProcessTerminationUnconfirmed", err, res)
	}
	if res == nil || res.Success || res.FinishReason != executor.FinishTimeout {
		t.Fatalf("result = %+v, want a failed timeout result alongside the error", res)
	}
}

// The control: a plain hang is killed, reaped, and reported as a timeout with no
// termination error, so the credential can be written back and released normally.
func TestTimeout_KilledAndReapedIsConfirmed(t *testing.T) {
	setupHomes(t)
	e := newExec(t, newMockAgy(t, ExpectedVersion, "", 0, "sleep 30"))
	task := &executor.Task{ID: "t1", Directive: "x", Workspace: t.TempDir(), Model: "gemini-4-neon-high", Timeout: 1500 * time.Millisecond}
	res, err := e.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("plain hang: err=%v, want nil (killed and reaped)", err)
	}
	if res.Success || res.FinishReason != executor.FinishTimeout {
		t.Fatalf("result = %+v, want a timeout", res)
	}
}
