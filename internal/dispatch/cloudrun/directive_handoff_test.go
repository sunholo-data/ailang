package cloudrun

import (
	"context"
	"errors"
	"strings"
	"testing"

	runpb "cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// memDirectives is an in-memory DirectiveWriter.
type memDirectives struct {
	stored map[string]string
	err    error
}

func (m *memDirectives) PutDirective(_ context.Context, taskID, directive string) error {
	if m.err != nil {
		return m.err
	}
	if m.stored == nil {
		m.stored = map[string]string{}
	}
	m.stored[taskID] = directive
	return nil
}

// newDispatcherWithClient creates a Dispatcher with a custom client and an
// in-memory directive store (for testing).
func newDispatcherWithClient(client jobRunner, projectID, region, prefix string) *Dispatcher {
	return &Dispatcher{
		client:     client,
		directives: &memDirectives{},
		projectID:  projectID,
		region:     region,
		prefix:     prefix,
	}
}

func envOf(req *runpb.RunJobRequest) map[string]string {
	out := map[string]string{}
	for _, e := range req.GetOverrides().GetContainerOverrides()[0].GetEnv() {
		out[e.GetName()] = e.GetValue()
	}
	return out
}

// The 2026-09-30 request: 43 KB, over Cloud Run's 32,768-byte env cap. It must
// dispatch, with the directive in the store and NOT in the env.
func TestDispatchLargeDirectiveGoesThroughTheStore(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "p", "r", "ailang")
	big := strings.Repeat("x", 43*1024)

	if err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task-big", AgentID: "a", Directive: big}); err != nil {
		t.Fatalf("a 43 KB directive must dispatch: %v", err)
	}
	if got := d.directives.(*memDirectives).stored["task-big"]; got != big {
		t.Fatalf("store holds %d bytes, want the full %d", len(got), len(big))
	}
	env := envOf(mock.lastReq)
	if env[config.EnvDirectiveSource] != config.DirectiveSourceFirestore {
		t.Errorf("%s = %q, want %q", config.EnvDirectiveSource, env[config.EnvDirectiveSource], config.DirectiveSourceFirestore)
	}
	if _, ok := env[config.EnvDirective]; ok {
		t.Errorf("a directive over %d bytes must not ride the env", inlineDirectiveMax)
	}
}

// A small directive is stored AND sent inline, so a job image that predates
// the Firestore read still runs it while images roll.
func TestDispatchSmallDirectiveAlsoInline(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "p", "r", "ailang")
	if err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task-s", AgentID: "a", Directive: "fix it"}); err != nil {
		t.Fatal(err)
	}
	if d.directives.(*memDirectives).stored["task-s"] != "fix it" {
		t.Error("small directive not stored")
	}
	if envOf(mock.lastReq)[config.EnvDirective] != "fix it" {
		t.Error("small directive not sent inline for older job images")
	}
}

// Any env value over the cap is refused BEFORE the API, as permanent, and
// nothing is dispatched.
func TestDispatchOversizedEnvIsPermanent(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "p", "r", "ailang")
	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID: "task-p", AgentID: "a", PolicyTOML: strings.Repeat("y", maxEnvValueBytes+1),
	})
	if !errors.Is(err, coordinator.ErrDispatchPermanent) {
		t.Fatalf("err = %v, want ErrDispatchPermanent", err)
	}
	if mock.lastReq != nil {
		t.Error("RunJob was called for a request that cannot succeed")
	}
}

func TestDispatchDirectiveOverStoreLimitIsPermanent(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "p", "r", "ailang")
	err := d.Dispatch(context.Background(), coordinator.DispatchParams{
		TaskID: "task-h", AgentID: "a", Directive: strings.Repeat("z", coordinator.MaxDirectiveBytes+1),
	})
	if !errors.Is(err, coordinator.ErrDispatchPermanent) {
		t.Fatalf("err = %v, want ErrDispatchPermanent", err)
	}
	if len(d.directives.(*memDirectives).stored) != 0 || mock.lastReq != nil {
		t.Error("an oversized directive was stored or dispatched")
	}
}

// A store outage is transient: not permanent, and the job is not started
// without its directive.
func TestDispatchStoreFailureIsTransient(t *testing.T) {
	mock := &mockJobRunner{}
	d := newDispatcherWithClient(mock, "p", "r", "ailang")
	d.directives = &memDirectives{err: errors.New("firestore unavailable")}
	err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task-t", AgentID: "a", Directive: "d"})
	if err == nil || errors.Is(err, coordinator.ErrDispatchPermanent) {
		t.Fatalf("err = %v, want a transient error", err)
	}
	if mock.lastReq != nil {
		t.Error("job started before its directive was stored")
	}
}

func TestDispatchRunJobErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		code      codes.Code
		permanent bool
	}{
		{codes.InvalidArgument, true},
		{codes.Unavailable, false},
		{codes.ResourceExhausted, false},
	} {
		mock := &mockJobRunner{err: status.Error(tc.code, "boom")}
		d := newDispatcherWithClient(mock, "p", "r", "ailang")
		err := d.Dispatch(context.Background(), coordinator.DispatchParams{TaskID: "task-c", AgentID: "a"})
		if got := errors.Is(err, coordinator.ErrDispatchPermanent); got != tc.permanent {
			t.Errorf("%s: permanent = %v, want %v (err %v)", tc.code, got, tc.permanent, err)
		}
	}
}
