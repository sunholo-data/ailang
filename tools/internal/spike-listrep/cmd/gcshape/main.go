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

type report struct {
	Arm               string `json:"arm"`
	N                 int    `json:"n"`
	NumGCDelta        uint32 `json:"num_gc_delta"`
	PauseTotalNsDelta uint64 `json:"pause_total_ns_delta"`
	HeapAllocBefore   uint64 `json:"heap_alloc_before"`
	HeapAllocAfter    uint64 `json:"heap_alloc_after"`
}

func main() {
	arm := flag.String("arm", "C0", "C0, C1, C2K8, or C2K32")
	n := flag.Int("n", 12800, "element count")
	flag.Parse()
	list, err := spikelistrep.NewArm(*arm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	element := &eval.IntValue{Value: 1}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range *n {
		list, err = spikelistrep.PrependArm(*arm, element, list)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	runtime.KeepAlive(list)
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(list)
	_ = json.NewEncoder(os.Stdout).Encode(report{*arm, *n, after.NumGC - before.NumGC, after.PauseTotalNs - before.PauseTotalNs, before.HeapAlloc, after.HeapAlloc})
}
