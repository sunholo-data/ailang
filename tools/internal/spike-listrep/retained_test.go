package spikelistrep_test

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	spikelistrep "github.com/sunholo-data/ailang/tools/internal/spike-listrep"
)

func TestSharedSingletonElements(t *testing.T) {
	singleton := &eval.IntValue{Value: 1}
	values := spikelistrep.SharedSingletonElements(100, singleton)
	for i, value := range values {
		if value != eval.Value(singleton) {
			t.Fatalf("element %d is not the shared singleton", i)
		}
	}
}
