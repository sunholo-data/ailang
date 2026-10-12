package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

type runnerOwnedWorker struct {
	done chan struct{}
	once sync.Once
}

func (w *runnerOwnedWorker) RequestStop() error { w.once.Do(func() { close(w.done) }); return nil }
func (w *runnerOwnedWorker) Join(ctx context.Context) error {
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (w *runnerOwnedWorker) Done() <-chan struct{} { return w.done }
func (w *runnerOwnedWorker) StdinClosing() bool    { return false }
func TestRunnerWorkerLifecycle_SetupFailureCleansSingleAndBatch(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "batch"}[batch], func(t *testing.T) {
			worker := &runnerOwnedWorker{done: make(chan struct{})}
			opts := Options{Quiet: true, AIHandler: func(ctx *effects.EffContext, _ *ai.AIRoutingPolicy, _ *ai.Attribution) error {
				admission, err := ctx.BeginWorker()
				if err != nil {
					return err
				}
				if err = admission.Complete(worker); err != nil {
					return err
				}
				return errors.New("injected setup failure")
			}}
			if batch {
				if err := ExecuteBatchItem(t.Context(), pipeline.Result{}, "input", opts, nil, nil); err == nil {
					t.Fatal("primary setup error lost")
				}
			} else if code := runSingle(t.Context(), pipeline.Result{}, opts, nil, ""); code != 1 {
				t.Fatal(code)
			}
			select {
			case <-worker.done:
			default:
				t.Fatal("setup failure leaked owned worker")
			}
		})
	}
}

type runnerFailedCleanupWorker struct{ runnerOwnedWorker }

func (w *runnerFailedCleanupWorker) Join(ctx context.Context) error {
	if err := w.runnerOwnedWorker.Join(ctx); err != nil {
		return err
	}
	return errors.New("injected worker cleanup failure")
}
func TestRunnerWorkerLifecycle_CleanupFailureChangesSuccessToFailure(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "worker_noop.ail")
	if err := os.WriteFile(filename, []byte("module worker_noop\nexport func main() -> () { () }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := pipeline.RunWithContext(t.Context(), pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true}, pipeline.Source{Filename: filename})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("fixture compilation: %v %v", err, result.Errors)
	}
	control := Options{Filename: filename, Entry: "main", ArgsJSON: "null", Quiet: true}
	if code := runSingle(t.Context(), result, control, nil, ""); code != -1 {
		t.Fatalf("clean single fixture did not succeed: %d", code)
	}
	if err := ExecuteBatchItem(t.Context(), result, "input", control, nil, nil); err != nil {
		t.Fatalf("clean batch fixture did not succeed: %v", err)
	}
	for _, batch := range []bool{false, true} {
		worker := &runnerFailedCleanupWorker{runnerOwnedWorker{done: make(chan struct{})}}
		opts := Options{Filename: filename, Entry: "main", ArgsJSON: "null", Quiet: true, AIHandler: func(ctx *effects.EffContext, _ *ai.AIRoutingPolicy, _ *ai.Attribution) error {
			admission, err := ctx.BeginWorker()
			if err != nil {
				return err
			}
			return admission.Complete(worker)
		}}
		if batch {
			if err := ExecuteBatchItem(t.Context(), result, "input", opts, nil, nil); err == nil || !strings.Contains(err.Error(), "injected worker cleanup failure") {
				t.Fatalf("batch swallowed cleanup failure: %v", err)
			}
		} else if code := runSingle(t.Context(), result, opts, nil, ""); code != 1 {
			t.Fatalf("successful single execution hid cleanup failure: %d", code)
		}
	}
}
