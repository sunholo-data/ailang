package spikelistrep_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	spikelistrep "github.com/sunholo-data/ailang/tools/internal/spike-listrep"
)

func BenchmarkListRep_B2_LinearBuild(b *testing.B) {
	for _, arm := range benchmarkArms {
		b.Run("arm="+arm.name, func(b *testing.B) {
			for _, n := range []int{1600, 3200, 6400, 12800} {
				b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
					benchmarkLinearBuild(b, arm, n)
				})
			}
		})
	}
}

func benchmarkLinearBuild(b *testing.B, arm benchmarkArm, n int) {
	element := &eval.IntValue{Value: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var list spikelistrep.List = arm.fromSlice(nil)
		for range n {
			list = arm.cons(element, list)
		}
		runtime.KeepAlive(list)
	}
}
