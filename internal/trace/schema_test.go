package trace

import "testing"

func TestReadLineOptNonDeterministic(t *testing.T) {
	if !IsNonDeterministic("IO", "readLineOpt") {
		t.Fatal("stdin read must allow nondeterministic trace results")
	}
	if IsNonDeterministic("IO", "println") {
		t.Fatal("output must remain deterministic")
	}
}
