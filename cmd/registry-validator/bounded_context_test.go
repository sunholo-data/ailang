package main

import (
	"strings"
	"testing"
	"time"
)

func TestBudgetContext(t *testing.T) {
	for _, resource := range []string{"memory", "time"} {
		be := &budgetError{resource: resource, limit: "384 MiB", observed: "600 MiB"}
		be.context = refusalContext(896<<20, []string{"GOMEMLIMIT=832MiB", "GOMAXPROCS=2"}, checkStats{PeakRSS: 600 << 20, Wall: time.Second})
		text := be.Error()
		for _, want := range []string{resource + " budget", "600 MiB", "cap=896 MiB", "unavailable", "GOMEMLIMIT=832MiB", "GOMAXPROCS=2", "100ms", "wall=1s"} {
			if !strings.Contains(text, want) {
				t.Errorf("missing %q: %s", want, text)
			}
		}
	}
}

func TestReproduction(t *testing.T) {
	t.Setenv("VALIDATOR_IMAGE", "registry.example/validator:v1")
	text := reproduction("832MiB")
	for _, want := range []string{"registry.example/validator:v1", "--platform linux/amd64", "--cpus 2", "--memory 1g", "GOMEMLIMIT=832MiB", "AILANG_NO_CACHE=1", "cp -a", "ailang lock", "mktemp -d", "gVisor"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q: %s", want, text)
		}
	}
	t.Setenv("VALIDATOR_IMAGE", "")
	if !strings.Contains(reproduction("832MiB"), "image identity unavailable") {
		t.Fatal("missing image must be explicit")
	}
}
