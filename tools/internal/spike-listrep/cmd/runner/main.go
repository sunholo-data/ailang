package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/protocol"
)

func main() {
	kind := flag.String("kind", "benchmark", "benchmark, retained, or gcshape")
	cell := flag.String("cell", "", "one anchored benchmark regex, or arm for a main program")
	control := flag.String("control", "", "anchored control regex, or control arm")
	metric := flag.String("metric", "ns/op", "reported metric")
	threshold := flag.Float64("threshold", math.NaN(), "comparison threshold")
	op := flag.String("op", "<=", "<= or >=")
	flag.Parse()
	if *cell == "" || *control == "" || math.IsNaN(*threshold) || (*op != "<=" && *op != ">=") {
		flag.Usage()
		os.Exit(2)
	}
	if *kind == "benchmark" {
		if _, e := protocol.ValidateCell(*cell); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
		if _, e := protocol.ValidateCell(*control); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(2)
		}
	}
	r := protocol.Adjudicate(context.Background(), protocol.Collect, protocol.Invocation{Kind: *kind, Selector: *cell, Metric: *metric}, protocol.Invocation{Kind: *kind, Selector: *control, Metric: *metric}, *threshold, *op)
	_ = json.NewEncoder(os.Stdout).Encode(r)
	if !r.Valid {
		os.Exit(1)
	}
}
