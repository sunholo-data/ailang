package riglock

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func requestPriority(t *testing.T, dir, pid string) string {
	t.Helper()
	if err := os.MkdirAll(dir+".priority", 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir+".priority", pid)
	if err := os.WriteFile(path, []byte("test request"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPriorityBlocksNewEvalAndDeadRequesterDoesNot(t *testing.T) {
	dir := isolate(t)
	request := requestPriority(t, dir, strconv.Itoa(os.Getpid()))
	ok, _, err := Acquire(NoWait)
	if err != nil || ok {
		t.Fatalf("live request must block admission: %v %v", ok, err)
	}
	if err := os.Remove(request); err != nil {
		t.Fatal(err)
	}
	dead := requestPriority(t, dir, "999999999")
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatalf("dead requester must not block: %v %v", ok, err)
	}
	defer release()
	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		t.Fatal("dead reservation not cleaned")
	}
}

func TestYieldWaitsForPriorityAndRestoresOwnership(t *testing.T) {
	dir := isolate(t)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatal(err)
	}
	defer release()
	owner := strconv.Itoa(os.Getpid())
	request := requestPriority(t, dir, owner)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Yield(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		if err := os.Mkdir(dir, 0o755); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("eval did not yield")
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "holder"), []byte("priority test"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		t.Fatalf("resumed during priority work: %v", err)
	default:
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	// Reservation between stages must also exclude the evaluator.
	time.Sleep(250 * time.Millisecond)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("eval reacquired between priority stages")
	}
	if err := os.Remove(request); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !ownedBy(owner) {
		t.Fatal("eval did not restore original ownership")
	}
}

func TestYieldCancellationDoesNotDeletePriorityLock(t *testing.T) {
	dir := isolate(t)
	ok, release, err := Acquire(NoWait)
	if err != nil || !ok {
		t.Fatal(err)
	}
	requestPriority(t, dir, strconv.Itoa(os.Getpid()))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Yield(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		if err := os.Mkdir(dir, 0o755); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no yield")
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "holder"), []byte("priority owner"), 0o600); err != nil {
		t.Fatal(err)
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("expected cancellation")
	}
	release()
	if !ownedBy("priority") {
		t.Fatal("eval cleanup deleted the priority task's lock")
	}
}
