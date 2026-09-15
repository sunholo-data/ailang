package main

import (
	"encoding/json"
	"testing"
)

// getInt used to type-assert float64 only, so a value that arrived as an
// int64 (a Firestore document field) or a json.Number (a decoder with
// UseNumber) printed as 0 — a span count of 0 in `ailang dashboard stats`
// with nothing in the output saying the number was unreadable.
func TestGetIntReadsEveryNumericRepresentation(t *testing.T) {
	m := map[string]interface{}{
		"json":   float64(42),
		"fs":     int64(43),
		"int":    44,
		"number": json.Number("45"),
		"str":    "46",
	}
	for key, want := range map[string]int{"json": 42, "fs": 43, "int": 44, "number": 45, "str": 0, "missing": 0} {
		if got := getInt(m, key); got != want {
			t.Errorf("getInt(%q) = %d, want %d", key, got, want)
		}
	}
	for key, want := range map[string]float64{"json": 42, "fs": 43, "int": 44, "number": 45, "str": 0} {
		if got := getFloat(m, key); got != want {
			t.Errorf("getFloat(%q) = %f, want %f", key, got, want)
		}
	}
}
