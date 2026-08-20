package spikelistrep_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	spikelistrep "github.com/sunholo-data/ailang/tools/internal/spike-listrep"
)

type benchmarkArm struct {
	name      string
	fromSlice func([]eval.Value) spikelistrep.List
	cons      func(eval.Value, spikelistrep.List) spikelistrep.List
}

var benchmarkArms = []benchmarkArm{
	{name: "C0", fromSlice: spikelistrep.SliceFromSlice, cons: spikelistrep.SliceCons},
	{name: "C1", fromSlice: spikelistrep.ConsFromSlice, cons: spikelistrep.ConsCons},
	{name: "C2K8", fromSlice: func(v []eval.Value) spikelistrep.List { return spikelistrep.ChunkFromSlice(8, v) }, cons: func(v eval.Value, l spikelistrep.List) spikelistrep.List { return spikelistrep.ChunkCons(8, v, l) }},
	{name: "C2K32", fromSlice: func(v []eval.Value) spikelistrep.List { return spikelistrep.ChunkFromSlice(32, v) }, cons: func(v eval.Value, l spikelistrep.List) spikelistrep.List { return spikelistrep.ChunkCons(32, v, l) }},
}

func BenchmarkListRep_B1_Branching(b *testing.B) {
	for _, arm := range benchmarkArms {
		b.Run("arm="+arm.name, func(b *testing.B) {
			for _, m := range []int{1024, 4096} {
				b.Run(fmt.Sprintf("m=%d", m), func(b *testing.B) {
					for _, length := range []int{1024, 4096, 16384} {
						b.Run(fmt.Sprintf("L=%d", length), func(b *testing.B) {
							benchmarkBranchingPrepend(b, arm, m, length)
						})
					}
				})
			}
		})
	}
}

func benchmarkBranchingPrepend(b *testing.B, arm benchmarkArm, m, length int) {
	element := &eval.IntValue{Value: 1}
	baseElements := make([]eval.Value, length)
	for i := range baseElements {
		baseElements[i] = element
	}
	base := arm.fromSlice(baseElements)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		results := make([]spikelistrep.List, m)
		for i := range results {
			results[i] = arm.cons(element, base)
		}
		runtime.KeepAlive(results)
	}
}
