package spikelistrep_test

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func BenchmarkListRep_B5_Materialize(b *testing.B) {
	const n = 4096
	for _, arm := range benchmarkArms {
		b.Run(fmt.Sprintf("arm=%s/n=%d", arm.name, n), func(b *testing.B) {
			values := make([]eval.Value, n)
			list := arm.fromSlice(values)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				materializeSink = list.ToSlice()
			}
		})
	}
}

var materializeSink []eval.Value
