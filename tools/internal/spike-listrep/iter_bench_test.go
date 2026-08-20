package spikelistrep_test

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func BenchmarkListRep_B3_Iteration(b *testing.B) {
	for _, arm := range benchmarkArms {
		for _, n := range []int{4096, 65536} {
			b.Run(fmt.Sprintf("arm=%s/n=%d", arm.name, n), func(b *testing.B) {
				values := make([]eval.Value, n)
				for i := range values {
					values[i] = &eval.IntValue{Value: i}
				}
				list := arm.fromSlice(values)
				b.ReportMetric(0, "ns/element")
				b.ResetTimer()
				for range b.N {
					sum := 0
					for value := range list.All() {
						sum += value.(*eval.IntValue).Value
					}
					iterationSink = sum
				}
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*n), "ns/element")
			})
		}
	}
}

var iterationSink int
