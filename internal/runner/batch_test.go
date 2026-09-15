package runner

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestRecoverBatchItemExit_Branches pins every branch of the recover directly,
// including the re-panic arm, which no .ail fixture can reach deterministically
// (it needs a genuine Go crash inside the evaluator). Without this, that branch
// would ship unguarded — a batch item that really crashes must stay loud rather
// than being silently downgraded to "this item failed".
func TestRecoverBatchItemExit_Branches(t *testing.T) {
	t.Run("no panic passes the error through unchanged", func(t *testing.T) {
		want := errors.New("ordinary failure")
		if got := RecoverBatchItemExit(func() error { return want }); !errors.Is(got, want) {
			t.Errorf("err = %v, want %v", got, want)
		}
		if got := RecoverBatchItemExit(func() error { return nil }); got != nil {
			t.Errorf("err = %v, want nil", got)
		}
	})

	t.Run("non-zero exit becomes a per-item error", func(t *testing.T) {
		got := RecoverBatchItemExit(func() error {
			panic(&eval.EvalExitCode{Code: 3})
		})
		if got == nil {
			t.Fatal("exit(3) produced no error — the item would count as a success")
		}
		if !strings.Contains(got.Error(), "exit(3)") {
			t.Errorf("err = %q, want it to name exit(3)", got.Error())
		}
	})

	t.Run("exit zero is a success", func(t *testing.T) {
		if got := RecoverBatchItemExit(func() error {
			panic(&eval.EvalExitCode{Code: 0})
		}); got != nil {
			t.Errorf("exit(0) err = %v, want nil", got)
		}
	})

	t.Run("a real crash is re-panicked, not swallowed", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("a non-exit panic was swallowed — genuine crashes would be " +
					"silently reported as a failed batch item")
			}
			if s, ok := r.(string); !ok || s != "genuine crash" {
				t.Errorf("re-panicked value = %v, want the original panic", r)
			}
		}()
		_ = RecoverBatchItemExit(func() error { panic("genuine crash") })
	})
}
