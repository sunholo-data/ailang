package main

import (
	"encoding/json"
	"fmt"
	"os"

	ailangErrors "github.com/sunholo-data/ailang/internal/errors"
)

// handleStructuredError outputs structured JSON error reports
func handleStructuredError(err error, compact bool) {
	// Try to extract a structured Report using errors.AsReport
	if rep, ok := ailangErrors.AsReport(err); ok {
		outputJSON(rep, compact)
		return
	}

	// Fallback: wrap in generic error
	generic := ailangErrors.NewGeneric("runtime", err)
	outputJSON(generic, compact)
}

// outputJSON marshals and prints JSON
func outputJSON(v interface{}, compact bool) {
	var data []byte
	var err error

	if compact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		return
	}

	fmt.Println(string(data))
}
