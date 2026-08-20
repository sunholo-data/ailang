package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/matrix"
	"github.com/sunholo-data/ailang/tools/internal/spike-listrep/protocol"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {
	out := flag.String("out", "", "JSON output path")
	format := flag.String("format", "json", "json or markdown")
	subset := flag.String("cells", "", "comma-separated cell names")
	e1 := flag.String("e-C1", "unknown", "pass, fail, unknown")
	e8 := flag.String("e-C2K8", "unknown", "pass, fail, unknown")
	e32 := flag.String("e-C2K32", "unknown", "pass, fail, unknown")
	flag.Parse()
	ac, bl := matrix.Sets()
	partial := *subset != ""
	if partial {
		wanted := map[string]bool{}
		for _, s := range strings.Split(*subset, ",") {
			wanted[s] = true
		}
		filter := func(in []matrix.Cell) []matrix.Cell {
			var o []matrix.Cell
			for _, c := range in {
				if wanted[c.Name] {
					o = append(o, c)
				}
			}
			return o
		}
		ac, bl = filter(ac), filter(bl)
	}
	start := time.Now()
	progress := func(k, n int, d time.Duration) {
		fmt.Fprintf(os.Stderr, "cell %d of %d, elapsed %s\n", k, n, d.Round(time.Second))
	}
	matrix.Collect(context.Background(), ac, protocol.Collect, progress)
	matrix.Collect(context.Background(), bl, protocol.Collect, progress)
	r, err := matrix.VerdictWith(context.Background(), ac, bl, map[string]matrix.Encapsulation{"C1": matrix.Encapsulation(*e1), "C2K8": matrix.Encapsulation(*e8), "C2K32": matrix.Encapsulation(*e32)}, partial, protocol.Collect)
	r.Metadata = matrix.Metadata{GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Machine: machine(), Started: start, Ended: time.Now(), Elapsed: time.Since(start).String()}
	var data []byte
	if *format == "markdown" {
		data = []byte(matrix.Markdown(r))
	} else {
		data, _ = json.MarshalIndent(r, "", "  ")
		data = append(data, '\n')
	}
	if *out != "" {
		if e := os.WriteFile(*out, data, 0644); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
	}
	_, _ = os.Stdout.Write(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func machine() string {
	h, e := os.Hostname()
	if e != nil {
		return "unknown"
	}
	return h
}
