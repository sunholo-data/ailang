package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestReadFindingsClassifiesModuleOnlyAndReaching(t *testing.T) {
	input := strings.NewReader(`
{"config":{"protocol_version":"v1.0.0"}}
{"finding":{"osv":"GO-2026-5750","trace":[{"module":"github.com/ollama/ollama","function":""}]}}
{"progress":{"message":"Scanning your code..."}}
{"finding":{"osv":"GO-2026-6218","trace":[{"module":"example.com/dependency","function":""},{"module":"example.com/ailang","function":"main.run"}]}}
`)

	reaching, moduleOnly, err := readFindings(input)
	if err != nil {
		t.Fatalf("readFindings() error = %v", err)
	}
	assertStringsEqual(t, "moduleOnly", moduleOnly, []string{"GO-2026-5750"})
	assertStringsEqual(t, "reaching", reaching, []string{"GO-2026-6218"})

	// Mutation killed: reverting readFindings to skip non-function traces
	// (the pre-#703 `Function == ""` continue) empties moduleOnly and reds this test.
}

func assertStringsEqual(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}
