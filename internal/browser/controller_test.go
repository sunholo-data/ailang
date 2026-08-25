package browser

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeProvider struct {
	createErr, connectionErr, inspectErr, exportErr, stopErr error
	calls                                                    []string
	connection                                               SensitiveConnection
	blockExport                                              bool
	stopSawExpiredContext                                    bool
}

func (f *fakeProvider) Name() string { return "fake" }
func (f *fakeProvider) Create(context.Context, SessionSpec) (Session, error) {
	f.calls = append(f.calls, "create")
	return Session{ID: "session-1", Provider: f.Name(), CreatedAt: time.Unix(10, 0)}, f.createErr
}
func (f *fakeProvider) Connection(context.Context, Session) (SensitiveConnection, error) {
	f.calls = append(f.calls, "connection")
	return f.connection, f.connectionErr
}
func (f *fakeProvider) Inspect(context.Context, Session) (InspectionRef, error) {
	f.calls = append(f.calls, "inspect")
	return InspectionRef{Available: true, Ref: "safe-inspection-ref"}, f.inspectErr
}
func (f *fakeProvider) Export(ctx context.Context, _ Session, _ string) (ArtifactManifest, error) {
	f.calls = append(f.calls, "export")
	if f.blockExport {
		<-ctx.Done()
		return ArtifactManifest{}, NewFailure(FailureArtifactExport, "export", ctx.Err())
	}
	return ArtifactManifest{Complete: f.exportErr == nil}, f.exportErr
}
func (f *fakeProvider) Stop(ctx context.Context, _ Session) (Usage, error) {
	f.calls = append(f.calls, "stop")
	f.stopSawExpiredContext = ctx.Err() != nil
	return Usage{DurationMS: 1000, ActionCount: 2}, f.stopErr
}

func TestControllerLifecycleSuccessAndIdempotentFinish(t *testing.T) {
	fake := &fakeProvider{connection: NewSensitiveConnection(MCPServerSpec{Name: "playwright", Command: "fake"}, nil)}
	c := NewController(fake, ControllerOptions{CleanupTimeout: time.Second, Now: func() time.Time { return time.Unix(20, 0) }})
	run, err := c.Start(context.Background(), SessionSpec{RunID: "run-1", ArtifactDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := run.Finish(context.Background(), TerminationCompleted)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Termination != TerminationCompleted || manifest.Usage.ActionCount != 2 || !manifest.Artifacts.Complete {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if _, err := run.Finish(context.Background(), TerminationCompleted); err != nil {
		t.Fatalf("second finish should be idempotent: %v", err)
	}
	if want := []string{"create", "connection", "inspect", "export", "stop"}; !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("calls = %v, want %v", fake.calls, want)
	}
}

func TestControllerConnectionFailureStillStopsSession(t *testing.T) {
	fake := &fakeProvider{connectionErr: NewFailure(FailureConnect, "connection", errors.New("dial failed"))}
	c := NewController(fake, ControllerOptions{CleanupTimeout: time.Second})
	_, err := c.Start(context.Background(), SessionSpec{RunID: "run-1"})
	if !IsFailure(err, FailureConnect) {
		t.Fatalf("error = %v, want %s", err, FailureConnect)
	}
	if want := []string{"create", "connection", "stop"}; !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("calls = %v, want %v", fake.calls, want)
	}
}

func TestControllerPreservesPrimaryAndReportsCleanupFailure(t *testing.T) {
	fake := &fakeProvider{
		connection: NewSensitiveConnection(MCPServerSpec{Name: "playwright", Command: "fake"}, nil),
		exportErr:  NewFailure(FailureArtifactExport, "export", errors.New("export unavailable")),
		stopErr:    NewFailure(FailureCleanup, "stop", errors.New("release unavailable")),
	}
	c := NewController(fake, ControllerOptions{CleanupTimeout: time.Second})
	run, err := c.Start(context.Background(), SessionSpec{RunID: "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := run.Finish(context.Background(), TerminationExecutorFailed)
	if !IsFailure(err, FailureArtifactExport) {
		t.Fatalf("primary error = %v, want export category", err)
	}
	if manifest.CleanupErrorCategory != FailureCleanup || manifest.ArtifactErrorCategory != FailureArtifactExport {
		t.Fatalf("cleanup/export categories not banked: %#v", manifest)
	}
}

func TestControllerReservesIndependentStopBudgetAfterExportTimeout(t *testing.T) {
	fake := &fakeProvider{
		connection:  NewSensitiveConnection(MCPServerSpec{Name: "playwright", Command: "fake"}, nil),
		blockExport: true,
	}
	c := NewController(fake, ControllerOptions{CleanupTimeout: 20 * time.Millisecond})
	run, err := c.Start(context.Background(), SessionSpec{RunID: "run-1"})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := run.Finish(context.Background(), TerminationExecutorFailed)
	if !IsFailure(err, FailureArtifactExport) || manifest.ArtifactErrorCategory != FailureArtifactExport {
		t.Fatalf("manifest=%#v err=%v, want artifact export failure", manifest, err)
	}
	if fake.stopSawExpiredContext {
		t.Fatal("stop inherited the exhausted export deadline")
	}
	if want := []string{"create", "connection", "inspect", "export", "stop"}; !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("calls = %v, want %v", fake.calls, want)
	}
}

func TestStableFailureCategories(t *testing.T) {
	categories := []FailureCategory{
		FailurePolicyDenied, FailureCapacityExhausted, FailureProvision,
		FailureConnect, FailureActionTimeout, FailureSessionTimeout,
		FailureRemoteDisconnected, FailureArtifactExport, FailureCleanup,
		FailureCostUnknown,
	}
	seen := map[FailureCategory]bool{}
	for _, category := range categories {
		if category == "" || seen[category] {
			t.Fatalf("invalid or duplicate category %q", category)
		}
		seen[category] = true
	}
}
