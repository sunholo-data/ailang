package spikelistrep_test

import (
	"fmt"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func BenchmarkListRep_BLEN_Length(b *testing.B) {
	for _, arm := range benchmarkArms {
		for _, n := range []int{4096, 65536} {
			b.Run(fmt.Sprintf("arm=%s/n=%d", arm.name, n), func(b *testing.B) {
				list := arm.fromSlice(make([]eval.Value, n))
				b.ResetTimer()
				for range b.N {
					lenSink = list.Len()
				}
			})
		}
	}
}

var lenSink int
