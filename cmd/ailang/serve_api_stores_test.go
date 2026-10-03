package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/runner"
)

// serve-api must initialise SharedMem/SharedIndex exactly as `ailang run`
// does: granted -> context present; not granted -> absent.
func TestInitServeAPIStores(t *testing.T) {
	granted := effects.NewEffContext(nil)
	if err := runner.GrantCapabilities(granted, "IO,SharedMem,SharedIndex"); err != nil {
		t.Fatal(err)
	}
	initServeAPIStores(granted)
	if granted.SharedMem == nil {
		t.Error("SharedMem granted but its context was not initialised")
	}
	if granted.SharedIndex == nil {
		t.Error("SharedIndex granted but its context was not initialised")
	}

	notGranted := effects.NewEffContext(nil)
	if err := runner.GrantCapabilities(notGranted, "IO"); err != nil {
		t.Fatal(err)
	}
	initServeAPIStores(notGranted)
	if notGranted.SharedMem != nil || notGranted.SharedIndex != nil {
		t.Error("stores initialised without the capability")
	}
}
