package main

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func TestOpenChainsReadBackend_DefaultsToLocal(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	backend, closeBackend, err := openChainsReadBackend(context.Background(), "")
	if err != nil {
		t.Fatalf("openChainsReadBackend: %v", err)
	}
	defer closeBackend()
	if _, ok := backend.(*observatory.SQLiteBackend); !ok {
		t.Fatalf("backend type = %T, want *observatory.SQLiteBackend", backend)
	}
}

func TestOpenChainsReadBackend_RemoteRoutesThroughStorageMode(t *testing.T) {
	t.Setenv("AILANG_CHAINS_READ", "")
	t.Setenv("AILANG_CLOUD_PROJECT", "")
	_, _, err := openChainsReadBackend(context.Background(), "gcp")
	if err == nil || !strings.Contains(err.Error(), "AILANG_CLOUD_PROJECT") {
		t.Fatalf("error = %v, want error containing AILANG_CLOUD_PROJECT", err)
	}
}

func TestOpenChainsReadBackend_EnvIsTheFallbackNotTheOverride(t *testing.T) {
	t.Run("env selects remote", func(t *testing.T) {
		t.Setenv("AILANG_CHAINS_READ", "gcp")
		t.Setenv("AILANG_CLOUD_PROJECT", "")
		_, _, err := openChainsReadBackend(context.Background(), "")
		if err == nil || !strings.Contains(err.Error(), "AILANG_CLOUD_PROJECT") {
			t.Fatalf("error = %v, want remote routing error", err)
		}
	})

	t.Run("flag beats env", func(t *testing.T) {
		t.Setenv("AILANG_CHAINS_READ", "gcp")
		backend, closeBackend, err := openChainsReadBackend(context.Background(), "local")
		if err != nil {
			t.Fatalf("openChainsReadBackend: %v", err)
		}
		defer closeBackend()
		if _, ok := backend.(*observatory.SQLiteBackend); !ok {
			t.Fatalf("backend type = %T, want *observatory.SQLiteBackend", backend)
		}
	})
}
