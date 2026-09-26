package iteration

import (
	"context"
	"testing"
)

func TestRuntimeRequiresDependencies(t *testing.T) {
	s := &Service{}
	if _, err := s.Run(context.Background(), validSpec()); err == nil {
		t.Fatal("runtime accepted missing dependencies")
	}
}
