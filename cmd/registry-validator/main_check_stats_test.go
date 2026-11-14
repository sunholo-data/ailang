package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPublishStoresCheckStats(t *testing.T) {
	buildWorktreeAilangOnPath(t)
	var stored []byte
	v := &validator{
		checkPackage: func(string) (bool, string, checkStats, error) {
			return true, "", checkStats{PeakRSS: 123456789, Wall: 987 * time.Millisecond}, nil
		},
		uploadObject: func(_ context.Context, path string, data []byte, _ string) error {
			if strings.HasSuffix(path, "/metadata.json") {
				stored = append([]byte(nil), data...)
			}
			return nil
		},
	}
	w := postFixtureTarball(t, v, "flat_self_import", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var meta pkg.PackageMetadata
	if err := json.Unmarshal(stored, &meta); err != nil {
		t.Fatalf("stored metadata: %v", err)
	}
	if meta.Validation.CheckPeakRSS != 123456789 || meta.Validation.CheckWallMS != 987 {
		t.Fatalf("stats=%+v", meta.Validation)
	}
	for _, tc := range []struct {
		budget bool
		status int
	}{{true, 422}, {false, 400}} {
		v.checkPackage = func(string) (bool, string, checkStats, error) {
			if tc.budget {
				return false, "", checkStats{}, &budgetError{resource: "memory", limit: "384 MiB", observed: "600 MiB"}
			}
			return false, "compile error", checkStats{}, nil
		}
		w = postFixtureTarball(t, v, "flat_self_import", nil)
		if w.Code != tc.status {
			t.Fatalf("status %d want %d: %s", w.Code, tc.status, w.Body.String())
		}
	}
}
