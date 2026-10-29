// Command maintenance-feasibility-reap exposes the actual existing cwd reaper
// to the no-model M1 OS canary. It adds no process ownership mechanism.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/proctree"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: maintenance-feasibility-reap <owned-workspace>")
		os.Exit(2)
	}
	found, err := proctree.ReapWorkspace(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if found == nil {
		found = []int{}
	}
	if err := json.NewEncoder(os.Stdout).Encode(found); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
