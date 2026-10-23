package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaintenanceManifestRejectsCaseFoldedSchemaAliases(t *testing.T) {
	for _, key := range []string{"schema", "settingsDigest", "requirementDigest"} {
		t.Run(key, func(t *testing.T) {
			path, m := maintenanceManifestFixture(t)
			if key == "schema" {
				writeMaintenanceManifest(t, path, m)
				raw, _ := os.ReadFile(path)
				raw = []byte(strings.Replace(string(raw), `"schema":`, `"Schema":`, 1))
				_ = os.WriteFile(path, raw, 0600)
			} else {
				pp := filepath.Join(filepath.Dir(path), m.Plan.Path)
				raw, _ := os.ReadFile(pp)
				raw = []byte(strings.Replace(string(raw), `"`+key+`":`, `"`+strings.ToUpper(key[:1])+key[1:]+`":`, 1))
				_ = os.WriteFile(pp, raw, 0600)
				sum := sha256.Sum256(raw)
				m.Plan.SHA256 = hex.EncodeToString(sum[:])
				writeMaintenanceManifest(t, path, m)
			}
			if _, err := LoadMaintenanceExecutionManifest(path); err == nil {
				t.Fatal("accepted non-schema casefold key")
			}
		})
	}
}
