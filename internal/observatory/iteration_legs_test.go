package observatory

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-MISSION-LOOP-UNIFIED-TELEMETRY M3 — node-generic dual-write.
//
// The ratified decision is dual-write, NODE-GENERIC ("this server, laptop, cloud,
// other nodes in the future") and NEVER-BLOCK when a destination is unreachable.
// Never-block is already satisfied structurally by the bounded+loud spool, so
// these tests assert the EXTENSION of that spool to a second leg rather than a
// new fail-soft policy.
//
// The subtle requirement: each leg spools SEPARATELY. A shared spool would replay
// a post that the local leg already stored, duplicating the chain on every flush.

func testLeg(t *testing.T, name string, sink IterationSink, warn *bytes.Buffer) IterationLeg {
	t.Helper()
	sp := NewSpool(filepath.Join(t.TempDir(), name+"-spool.jsonl"))
	sp.SetWarnWriter(warn)
	return IterationLeg{Name: name, Sink: sink, Spool: sp}
}

func TestPostToLegs_DualWriteReachesBothStores(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	cloud := newFakeSink() // Firestore: no eval_assessment support

	legs := []IterationLeg{
		testLeg(t, "local", local, &warn),
		testLeg(t, "cloud", cloud, &warn),
	}
	PostToLegs(context.Background(), legs, iter190Post(), &warn)

	for _, sink := range []*fakeSink{local.fakeSink, cloud} {
		if len(sink.chains) != 1 {
			t.Errorf("sink wrote %d chains, want 1", len(sink.chains))
		}
		if len(sink.stages) != 4 {
			t.Errorf("sink wrote %d stages, want 4", len(sink.stages))
		}
	}
	for _, leg := range legs {
		if n := leg.Spool.Len(); n != 0 {
			t.Errorf("leg %q spooled %d posts after a successful write, want 0", leg.Name, n)
		}
	}
}

// TestPostToLegs_CloudUnreachableNeverBlocks is the ratified never-block
// requirement, asserted against a deliberately-broken cloud leg.
func TestPostToLegs_CloudUnreachableNeverBlocks(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	cloud := newFakeSink()
	cloud.failOn = "CreateChain" // the store is there but refusing writes

	legs := []IterationLeg{
		testLeg(t, "local", local, &warn),
		testLeg(t, "cloud", cloud, &warn),
	}
	PostToLegs(context.Background(), legs, iter190Post(), &warn)

	if len(local.chains) != 1 {
		t.Errorf("local leg wrote %d chains, want 1 — a cloud outage must not stop the local write", len(local.chains))
	}
	if n := legs[1].Spool.Len(); n != 1 {
		t.Errorf("cloud spool holds %d posts, want 1", n)
	}
	// The local leg succeeded, so it must NOT be replayed later — that would
	// duplicate the chain it already stored.
	if n := legs[0].Spool.Len(); n != 0 {
		t.Errorf("local spool holds %d posts, want 0 (it succeeded)", n)
	}
	if got := warn.String(); !strings.Contains(got, "cloud") {
		t.Errorf("no loud notice naming the failed leg; got:\n%s", got)
	}
}

// TestPostToLegs_ReportsWhereTheDataWent: the caller prints this, so claiming
// delivery to a leg that only buffered would make a cloud outage invisible in
// the loop's own output.
func TestPostToLegs_ReportsWhereTheDataWent(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	cloud := newFakeSink()
	cloud.failOn = "CreateChain"

	legs := []IterationLeg{
		testLeg(t, "local", local, &warn),
		testLeg(t, "cloud", cloud, &warn),
	}
	delivered, spooled := PostToLegs(context.Background(), legs, iter190Post(), &warn)

	if len(delivered) != 1 || delivered[0] != "local" {
		t.Errorf("delivered = %v, want [local]", delivered)
	}
	if len(spooled) != 1 || spooled[0] != "cloud" {
		t.Errorf("spooled = %v, want [cloud]", spooled)
	}

	// An invalid post is neither delivered nor buffered.
	delivered, spooled = PostToLegs(context.Background(), legs, &IterationPost{Source: "x"}, &warn)
	if len(delivered) != 0 || len(spooled) != 0 {
		t.Errorf("invalid post reported delivered=%v spooled=%v, want both empty", delivered, spooled)
	}
}

// TestPostToLegs_UnavailableLegSpoolsWithoutASink covers the connect-time
// failure: the destination could not be opened at all, so there is no sink to
// call. The post must still be buffered rather than dropped.
func TestPostToLegs_UnavailableLegSpoolsWithoutASink(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	cloud := testLeg(t, "cloud", nil, &warn)
	cloud.Err = context.DeadlineExceeded

	legs := []IterationLeg{testLeg(t, "local", local, &warn), cloud}
	PostToLegs(context.Background(), legs, iter190Post(), &warn)

	if n := cloud.Spool.Len(); n != 1 {
		t.Errorf("cloud spool holds %d posts, want 1 (an unopenable leg still buffers)", n)
	}
	if len(local.chains) != 1 {
		t.Errorf("local leg wrote %d chains, want 1", len(local.chains))
	}
}

// TestPostToLegs_NoCloudLegIsIdenticalToToday: with no cloud configured the
// caller passes one leg, and nothing about the local path changes — in
// particular no second spool file is created.
func TestPostToLegs_NoCloudLegIsIdenticalToToday(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	leg := testLeg(t, "local", local, &warn)

	PostToLegs(context.Background(), []IterationLeg{leg}, iter190Post(), &warn)

	if len(local.chains) != 1 {
		t.Fatalf("local leg wrote %d chains, want 1", len(local.chains))
	}
	entries, err := os.ReadDir(filepath.Dir(leg.Spool.Path))
	if err != nil {
		t.Fatalf("read spool dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("single-leg post created %d file(s) in the spool dir, want 0: %v", len(entries), entries)
	}
	if warn.Len() != 0 {
		t.Errorf("single-leg success warned on stderr: %s", warn.String())
	}
}

// TestPostToLegs_CloudSpoolStaysBounded: adding a second leg must not weaken the
// spool's existing caps.
func TestPostToLegs_CloudSpoolStaysBounded(t *testing.T) {
	var warn bytes.Buffer
	cloud := newFakeSink()
	cloud.failOn = "CreateChain"
	leg := testLeg(t, "cloud", cloud, &warn)

	for i := 0; i < DefaultSpoolMaxEntries+25; i++ {
		PostToLegs(context.Background(), []IterationLeg{leg}, iter190Post(), &warn)
	}
	if n := leg.Spool.Len(); n != DefaultSpoolMaxEntries {
		t.Errorf("cloud spool holds %d posts, want the %d-entry cap", n, DefaultSpoolMaxEntries)
	}
	if !strings.Contains(warn.String(), "OVERFLOW") {
		t.Error("spool overflowed without a loud OVERFLOW notice")
	}
}

func TestFlushLegs_ReplaysOnlyItsOwnSpool(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	cloud := newFakeSink()
	cloud.failOn = "CreateChain"

	legs := []IterationLeg{
		testLeg(t, "local", local, &warn),
		testLeg(t, "cloud", cloud, &warn),
	}
	PostToLegs(context.Background(), legs, iter190Post(), &warn)

	// Cloud recovers; the next invocation flushes ONLY the cloud backlog.
	cloud.failOn = ""
	FlushLegs(context.Background(), legs, &warn)

	if len(cloud.chains) != 1 {
		t.Errorf("cloud holds %d chains after flush, want 1", len(cloud.chains))
	}
	if len(local.chains) != 1 {
		t.Errorf("local holds %d chains, want 1 — a cloud flush must not re-post the local leg", len(local.chains))
	}
	if n := legs[1].Spool.Len(); n != 0 {
		t.Errorf("cloud spool holds %d posts after a successful flush, want 0", n)
	}
}

func TestFlushLegs_StillBrokenLegRebuffers(t *testing.T) {
	var warn bytes.Buffer
	cloud := newFakeSink()
	cloud.failOn = "CreateChain"
	legs := []IterationLeg{testLeg(t, "cloud", cloud, &warn)}

	PostToLegs(context.Background(), legs, iter190Post(), &warn)
	FlushLegs(context.Background(), legs, &warn) // still broken

	if n := legs[0].Spool.Len(); n != 1 {
		t.Errorf("cloud spool holds %d posts, want 1 (a failed replay must re-buffer, not drop)", n)
	}
}

// TestPostToLegs_InvalidPostIsNotSpooled: a malformed post is a caller bug, not
// an outage. Buffering it would replay the same rejection on every future
// iteration until it aged out of the cap, evicting recoverable posts.
func TestPostToLegs_InvalidPostIsNotSpooled(t *testing.T) {
	var warn bytes.Buffer
	local := &modelFakeSink{newFakeSink()}
	leg := testLeg(t, "local", local, &warn)

	PostToLegs(context.Background(), []IterationLeg{leg}, &IterationPost{
		Source: "mission:v1/iter-191",
		Stages: []IterationStage{{Role: "controller", Status: "finished"}},
	}, &warn)

	if n := leg.Spool.Len(); n != 0 {
		t.Errorf("spool holds %d posts, want 0 (an invalid post must not be retried forever)", n)
	}
	if !strings.Contains(warn.String(), "invalid") {
		t.Errorf("invalid post rejected without a loud notice; got:\n%s", warn.String())
	}
}
