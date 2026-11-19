package trace

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/eval"
)

// A trace byte budget used to cut a multi-byte character in half, so a
// recorded value could be invalid UTF-8 (pre-existing; M-V1-SHOW-TOTAL review
// item 6). Both bounding sites — eval.ShowTraceBounded at render time and the
// collector's truncateValue — now cut on a rune boundary, and the collector
// still recognises a render-bounded value (no second marker).
func TestTraceBudgetCutsOnRuneBoundaries(t *testing.T) {
	values := []eval.Value{
		&eval.StringValue{Value: strings.Repeat("é中😀a", 40)},
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: strings.Repeat("😀", 50)}, &eval.StringValue{Value: "ü"}}},
		&eval.RecordValue{Fields: map[string]eval.Value{"name": &eval.StringValue{Value: strings.Repeat("中文", 30)}}},
	}
	for vi, v := range values {
		full := eval.Show(v)
		for budget := 1; budget <= 40; budget++ {
			bounded := eval.ShowTraceBounded(v, budget)
			if !utf8.ValidString(bounded) {
				t.Fatalf("value %d @%d: render-bounded value is not valid UTF-8: %q", vi, budget, bounded)
			}
			if got := truncateValue(bounded, budget); got != bounded {
				t.Fatalf("value %d @%d: collector re-truncated a render-bounded value:\n got %q\nwant %q", vi, budget, got, bounded)
			}
			collected := truncateValue(full, budget)
			if !utf8.ValidString(collected) {
				t.Fatalf("value %d @%d: collector-bounded value is not valid UTF-8: %q", vi, budget, collected)
			}
			if collected != bounded {
				t.Fatalf("value %d @%d: the two bounding sites disagree:\n render    %q\n collector %q", vi, budget, bounded, collected)
			}
			if truncateValue(collected, budget) != collected {
				t.Fatalf("value %d @%d: truncateValue is not idempotent: %q", vi, budget, collected)
			}
		}
	}
}
