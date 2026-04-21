package main

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// runCompileCacheClear clears the compilation cache.
func runCompileCacheClear() {
	cs, err := pipeline.NewCacheStore(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	entries, _ := cs.Stats()
	if err := cs.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("Cleared %d cached compilation entries\n", entries)
}

// runCompileCacheStats shows compilation cache statistics.
func runCompileCacheStats() {
	cs, err := pipeline.NewCacheStore(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	entries, totalMs := cs.Stats()
	fmt.Printf("Compilation cache: .ailang/cache/compile/\n")
	fmt.Printf("  Entries: %d\n", entries)
	fmt.Printf("  Total recorded compile time: %dms\n", totalMs)
}
