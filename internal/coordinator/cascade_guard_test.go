package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-CASCADE-ENVELOPE-HYDRATION. Every fixture below is built with the
// builder `ailang publish` uses, so the payload shape is the production one:
// message_type "notification", package kind only inside the payload.

const legacyDependentInbox = "pkg:sunholo/docparse"

// legacyUpgradeRow is the stored row EmitUpgradeAvailable writes into a
// dependent's inbox: no Source, no cascade envelope.
func legacyUpgradeRow(t *testing.T, id string) *messaging.InboxMessage {
	t.Helper()
	msg, err := messaging.UpgradeAvailableMessage(
		messaging.PackageVersionInfo{Name: "sunholo/ailang_parse", Version: "0.39.3", InterfaceHash: "if-old", ContentHash: "c-old"},
		messaging.PackageVersionInfo{Name: "sunholo/ailang_parse", Version: "0.39.4", InterfaceHash: "if-new", ContentHash: "c-new"},
		[]string{legacyDependentInbox}, "")
	if err != nil || msg == nil {
		t.Fatalf("building the publisher's upgrade-available row: msg=%v err=%v", msg, err)
	}
	msg.ID, msg.MessageID = id, id
	return msg
}

func TestCascadeShaped(t *testing.T) {
	row := legacyUpgradeRow(t, "inbox_1_aaaaaaaa")
	if row.MessageType == string(messaging.PkgMsgUpgradeAvailable) {
		t.Fatalf("fixture premise changed: the stored row's message_type is now %q; "+
			"cascadeShaped's payload check may no longer be the only signal", row.MessageType)
	}
	compat := `{"schema":"` + messaging.PackageMessageSchema + `","kind":"compatibility-request","package":{"name":"x"}}`
	for _, tc := range []struct {
		name, kind, content string
		want                bool
	}{
		{"cascade-topic attribute", "upgrade-available", "", true},
		{"interface change attribute", "interface-change-notice", "anything", true},
		{"stored row: kind only in payload", row.MessageType, row.Payload, true},
		{"ordinary bug report to a package", "notification", "eparse --extract writes nothing for 13.6MB PDFs", false},
		{"package message that is not a bump", "notification", compat, false},
		{"bump kind without the package schema", "notification", `{"kind":"upgrade-available"}`, false},
	} {
		if got := cascadeShaped(tc.kind, tc.content); got != tc.want {
			t.Errorf("%s: cascadeShaped = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestCascadeDispatchRefusal(t *testing.T) {
	row := legacyUpgradeRow(t, "inbox_1_bbbbbbbb")
	empty := &TaskRecord{ID: "task-bbbbbbbb", MessageID: row.ID, Kind: row.MessageType, Content: row.Payload}
	err := cascadeDispatchRefusal(empty, legacyDependentInbox)
	if !errors.Is(err, ErrDispatchPermanent) {
		t.Fatalf("an envelope-less cascade task must be refused permanently, got %v", err)
	}
	for _, want := range []string{"task-bbbbbbbb", row.ID, `source=""`, "root_package", "to_version"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must name %q so the operator can act on it: %v", want, err)
		}
	}

	full := *empty
	full.Source, full.RootPackage, full.ToVersion = pubsub.SourceCascade, "sunholo/ailang_parse@0.39.4", "0.39.4"
	if err := cascadeDispatchRefusal(&full, legacyDependentInbox); err != nil {
		t.Errorf("a cascade task carrying its envelope must dispatch: %v", err)
	}

	// Source alone is not enough: a cascade-topic task with no bump is just as blank.
	sourceOnly := *empty
	sourceOnly.Source = pubsub.SourceCascade
	if err := cascadeDispatchRefusal(&sourceOnly, legacyDependentInbox); err == nil {
		t.Error("a cascade task with Source but no root package/version must be refused")
	}

	bug := &TaskRecord{ID: "task-cccccccc", Kind: "notification", Content: "show --json has no recipients"}
	if err := cascadeDispatchRefusal(bug, legacyDependentInbox); err != nil {
		t.Errorf("a package agent's ordinary bug report must not be gated: %v", err)
	}
	if err := cascadeDispatchRefusal(empty, "coordinator"); err != nil {
		t.Errorf("the gate is for package agents only: %v", err)
	}
}

// Route B, the route that produced the measured flood: the sweep re-enqueued a
// stored upgrade-available row with no Source and no envelope, every pass.
//
// MU: drop the cascadeShaped withhold in backstop_sweep.go and this fails.
func TestBackstopSweepWithholdsEnvelopeLessCascadeRows(t *testing.T) {
	reg := NewAgentRegistry()
	mustRegister(t, reg, "pkg-sunholo-docparse", legacyDependentInbox)

	row := legacyUpgradeRow(t, "inbox_1788309537210_6b9ee299")
	store := &sweepStore{msgs: []messaging.InboxMessage{
		*row,
		{ID: "inbox_bug", ToInbox: legacyDependentInbox, FromAgent: "clients", Title: "bug", MessageType: "notification", Payload: "a real bug"},
	}}
	var logBuf strings.Builder
	adapter := &PubSubInboxAdapter{logger: log.New(io.Discard, "", 0)}
	s := NewBackstopSweep(store, reg, adapter, log.New(&logBuf, "", 0))
	s.mode = BackstopDispatch
	s.SweepOnce(context.Background())

	if len(adapter.buffered) != 1 || adapter.buffered[0].ID != "inbox_bug" {
		t.Fatalf("sweep must recover the bug report and withhold the cascade row, buffered=%v", bufferedIDs(adapter))
	}
	out := logBuf.String()
	if !strings.Contains(out, "WITHHOLDING "+row.ID) {
		t.Errorf("a withheld cascade row must be named loudly, got:\n%s", out)
	}

	// Report mode must say the same thing rather than "WOULD recover" it.
	logBuf.Reset()
	adapter.buffered = nil
	s.mode = BackstopReport
	s.SweepOnce(context.Background())
	if out := logBuf.String(); strings.Contains(out, "WOULD recover "+row.ID) || !strings.Contains(out, "WITHHOLDING "+row.ID) {
		t.Errorf("report mode must report the cascade row as withheld, got:\n%s", out)
	}
}

// Route A: a cascade-topic notification whose data carries no envelope (an
// older publisher, or the legacy notification shape) used to proceed with every
// envelope field blank. It is now acked and logged, never buffered.
//
// MU: drop the post-hydration envelope check in pubsub_adapter.go and this fails.
func TestPubSubAdapter_CascadeWithoutEnvelope_NotDispatched(t *testing.T) {
	store := cascadeStore(t)
	row := legacyUpgradeRow(t, "inbox_1_dddddddd")
	if err := store.InsertInboxMessage(row); err != nil {
		t.Fatal(err)
	}
	var logBuf strings.Builder
	a := NewPubSubInboxAdapter(nil, "sub", "", store, log.New(&logBuf, "", 0))

	data, _ := json.Marshal(pubsub.MessageNotification{MessageID: row.ID}) // no envelope
	attrs := pubsub.MessageAttributes{Inbox: legacyDependentInbox, MessageType: "upgrade-available", Source: pubsub.SourceCascade}.ToMap()
	if err := a.HandleNotification(data, attrs); err != nil {
		t.Fatalf("permanently envelope-less data must be acked, not redelivered: %v", err)
	}
	if n := len(a.buffered); n != 0 {
		t.Fatalf("buffered %d message(s): a cascade with no envelope must not become a task", n)
	}
	if !strings.Contains(logBuf.String(), row.ID) {
		t.Errorf("the refusal must name the message, got:\n%s", logBuf.String())
	}

	// The same cascade WITH its envelope still dispatches, populated.
	full, fullAttrs := cascadeNotification(t, row.ID)
	fullAttrs["inbox"] = legacyDependentInbox
	if err := a.HandleNotification(full, fullAttrs); err != nil {
		t.Fatal(err)
	}
	if len(a.buffered) != 1 || a.buffered[0].RootPackage == "" || a.buffered[0].Source != pubsub.SourceCascade {
		t.Fatalf("a cascade carrying its envelope must dispatch populated, got %+v", a.buffered)
	}
}

// The dispatch gate, end to end on the route the sweep and adapter guards do
// not cover: the stored row's ordinary inbox notification (no cascade source)
// becomes a task for the package agent. It must fail at dispatch — a nil
// publisher panics if it gets further — instead of running the agent.
//
// MU: drop the cascadeDispatchRefusal call in daemon_tasks_exec.go and this panics.
func TestDispatchRefusesEnvelopeLessCascadeTask(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)
	d.ctx = context.Background()
	d.analyzer = NewTaskAnalyzer(0.8)
	d.agentRegistry = NewAgentRegistry()
	mustRegister(t, d.agentRegistry, "pkg-sunholo-docparse", legacyDependentInbox)
	tasks, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	d.taskStore = tasks

	row := legacyUpgradeRow(t, "inbox_1788339964243_6b1657f2")
	if err := d.msgStore.InsertInboxMessage(row); err != nil {
		t.Fatal(err)
	}
	d.cloudInboxAdapter = NewPubSubInboxAdapter(nil, "", "", d.msgStore, d.logger)
	data, _ := json.Marshal(pubsub.MessageNotification{MessageID: row.ID})
	if err := d.cloudInboxAdapter.HandleNotification(data, map[string]string{"inbox": legacyDependentInbox}); err != nil {
		t.Fatal(err)
	}
	if err := d.pollAndProcessTasks(); err != nil {
		t.Fatal(err)
	}
	if err := d.dispatchTasksCloud(); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.GetTask(d.ctx, "task-6b1657f2")
	if err != nil || got == nil {
		t.Fatalf("task not created from the stored row: %v %v", got, err)
	}
	if got.Status != TaskStatusFailed || !strings.Contains(got.Error, "no cascade envelope") {
		t.Fatalf("envelope-less cascade task must fail at dispatch, got status=%s error=%q", got.Status, got.Error)
	}
}

// Both task-creation sites copy the envelope through one helper, so the local
// site can no longer drift from the cloud one.
func TestCopyCascadeEnvelope(t *testing.T) {
	msg := &Message{
		RootPackage: "sunholo/auth@1.3.0", RootChangeClass: "C", FromVersion: "1.2.0", ToVersion: "1.3.0",
		FromInterfaceHash: "if-a", ToInterfaceHash: "if-b", FromContentHash: "c-a", ToContentHash: "c-b",
		EffectsWidened: true, PrevEffectCeiling: []string{"IO"}, NewEffectCeiling: []string{"IO", "Net"},
	}
	task := &TaskRecord{}
	task.copyCascadeEnvelope(msg)
	if task.RootPackage != msg.RootPackage || task.RootChangeClass != "C" || task.FromVersion != "1.2.0" ||
		task.ToVersion != "1.3.0" || task.FromInterfaceHash != "if-a" || task.ToInterfaceHash != "if-b" ||
		task.FromContentHash != "c-a" || task.ToContentHash != "c-b" || !task.EffectsWidened ||
		strings.Join(task.PrevEffectCeiling, ",") != "IO" || strings.Join(task.NewEffectCeiling, ",") != "IO,Net" {
		t.Fatalf("envelope not copied in full: %+v", task)
	}
}

func bufferedIDs(a *PubSubInboxAdapter) []string {
	var ids []string
	for _, m := range a.buffered {
		ids = append(ids, m.ID)
	}
	return ids
}

// The happy path end to end: a cascade-topic delivery's envelope reaches the
// task record, so the dispatcher and the gate both see it. Nothing pinned this
// copy before; dropping it would have turned every cascade into an EMPTY one.
//
// MU: drop copyCascadeEnvelope from the cloud task-creation site and this fails.
func TestCascadeEnvelopeReachesTaskRecord(t *testing.T) {
	d := newTestDaemonWithMsgStore(t)
	d.ctx = context.Background()
	d.analyzer = NewTaskAnalyzer(0.8)
	d.agentRegistry = NewAgentRegistry()
	dependent := messaging.FormatPackageInbox("sunholo/report_gen")
	mustRegister(t, d.agentRegistry, "pkg-sunholo-report-gen", dependent)
	tasks, err := NewSQLiteStore(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	d.taskStore = tasks
	d.cloudInboxAdapter = NewPubSubInboxAdapter(nil, "", "", d.msgStore, d.logger)

	data, attrs := cascadeNotification(t, "msg_20261007_171616_782f344c")
	if err := d.cloudInboxAdapter.HandleNotification(data, attrs); err != nil {
		t.Fatal(err)
	}
	if err := d.pollAndProcessTasks(); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.GetTask(d.ctx, "task-782f344c")
	if err != nil || got == nil {
		t.Fatalf("cascade task not created: %v %v", got, err)
	}
	if got.Source != pubsub.SourceCascade || got.RootPackage != "sunholo/json_tools@0.4.0" || got.RootChangeClass != "B" ||
		got.FromVersion != "0.3.2" || got.ToVersion != "0.4.0" || got.ToInterfaceHash != "if-new" || got.ToContentHash != "c-new" {
		t.Fatalf("cascade envelope did not reach the task record: %+v", got)
	}
	if err := cascadeDispatchRefusal(got, dependent); err != nil {
		t.Errorf("a populated cascade task must pass the dispatch gate: %v", err)
	}
	// The dispatcher reads pending tasks through ListTasks, not GetTask: the
	// SQLite store wrote the envelope and read it back on neither.
	pending, err := tasks.ListTasks(d.ctx, &TaskFilter{Status: []TaskStatus{TaskStatusPending}})
	if err != nil || len(pending) != 1 || pending[0].RootPackage != got.RootPackage || pending[0].ToVersion != "0.4.0" {
		t.Fatalf("ListTasks lost the cascade envelope: %+v %v", pending, err)
	}
}
