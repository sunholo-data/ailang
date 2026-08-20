package spikelistrep_test

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func BenchmarkListRep_B4_Nth(b *testing.B) {
	const n = 4096
	for _, arm := range benchmarkArms {
		values := make([]eval.Value, n)
		for i := range values {
			values[i] = &eval.IntValue{Value: i}
		}
		list := arm.fromSlice(values)
		for _, index := range []int{0, n / 4, n / 2, n - 1} {
			b.Run(fmt.Sprintf("arm=%s/n=%d/i=%d", arm.name, n, index), func(b *testing.B) {
				b.ResetTimer()
				for range b.N {
					nthSink, _ = list.At(index)
				}
			})
		}
	}
}

var nthSink eval.Value
