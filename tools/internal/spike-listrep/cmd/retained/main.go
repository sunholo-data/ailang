package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/sunholo-data/ailang/internal/eval"
	spikelistrep "github.com/sunholo-data/ailang/tools/internal/spike-listrep"
)

type counters struct {
	HeapAlloc   uint64 `json:"heap_alloc"`
	HeapObjects uint64 `json:"heap_objects"`
}
type report struct {
	RawEmptyBaseline     int64    `json:"raw_empty_baseline"`
	PostWorkloadCounters counters `json:"post_workload_counters"`
	AdjustedDelta        int64    `json:"adjusted_delta"`
	BytesPerElement      float64  `json:"bytes_per_element"`
}

func snapshot() runtime.MemStats {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

func main() {
	arm := flag.String("arm", "C0", "C0, C1, C2K8, or C2K32")
	n := flag.Int("n", 100000, "element count")
	flag.Parse()
	if *n <= 0 {
		fmt.Fprintln(os.Stderr, "n must be positive")
		os.Exit(2)
	}
	baselineBefore := snapshot()
	baselineAfter := snapshot()
	emptyDelta := int64(baselineAfter.HeapAlloc) - int64(baselineBefore.HeapAlloc)
	before := snapshot()
	singleton := &eval.IntValue{Value: 1}
	list, err := spikelistrep.NewArm(*arm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for range spikelistrep.SharedSingletonElements(*n, singleton) {
		list, err = spikelistrep.PrependArm(*arm, singleton, list)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	runtime.KeepAlive(list)
	after := snapshot()
	runtime.KeepAlive(list)
	rawDelta := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	adjusted := rawDelta - emptyDelta
	_ = json.NewEncoder(os.Stdout).Encode(report{emptyDelta, counters{after.HeapAlloc, after.HeapObjects}, adjusted, float64(adjusted) / float64(*n)})
}
