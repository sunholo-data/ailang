package pubsub

import (
	"context"
	"errors"
	"testing"
)

// A test binary must never get a live Pub/Sub client, whatever config it read.
func TestNewClient_RefusesRealProjectFromTestBinary(t *testing.T) {
	c, err := NewClient(context.Background(), "ailang-multivac", "ailang")
	if !errors.Is(err, ErrRealProjectFromTest) {
		t.Fatalf("NewClient from a test binary: err = %v, want ErrRealProjectFromTest", err)
	}
	if c != nil {
		t.Fatalf("NewClient returned a client alongside the refusal")
	}
}
