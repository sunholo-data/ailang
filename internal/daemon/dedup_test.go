package daemon

import (
	"testing"
	"time"
)

func TestDedup_FirstHitNotSeen(t *testing.T) {
	d := newDedup(60 * time.Second)
	if d.seen("k1") {
		t.Error("first hit should not be seen")
	}
}

func TestDedup_SecondHitWithinWindow(t *testing.T) {
	d := newDedup(60 * time.Second)
	d.seen("k1") // record
	if !d.seen("k1") {
		t.Error("second hit within window should be seen")
	}
}

func TestDedup_HitAfterWindowExpires(t *testing.T) {
	now := time.Now()
	d := newDedup(60 * time.Second)
	d.now = func() time.Time { return now }
	d.seen("k1")
	d.now = func() time.Time { return now.Add(61 * time.Second) }
	if d.seen("k1") {
		t.Error("hit after window expiry should not be seen")
	}
}

func TestDedup_DifferentKeysIndependent(t *testing.T) {
	d := newDedup(60 * time.Second)
	d.seen("k1")
	if d.seen("k2") {
		t.Error("different key should not be seen")
	}
}

func TestDedup_SweepRemovesExpired(t *testing.T) {
	now := time.Now()
	d := newDedup(60 * time.Second)
	d.now = func() time.Time { return now }
	d.seen("k1")
	d.seen("k2")
	d.now = func() time.Time { return now.Add(120 * time.Second) }
	d.sweep()
	if d.size() != 0 {
		t.Errorf("expected 0 after sweep, got %d", d.size())
	}
}
