package main

import (
	"testing"
)

// M-DX-PI-HARNESS Distribution v2: the managed-file install contract.
// decidePiInstall must never plan a clobber of user-owned content.
func TestDecidePiInstall(t *testing.T) {
	const embeddedContent = "extension content v2"
	const v2 = "v0.35.0"

	tests := []struct {
		name      string
		diskHash  string // "" = absent
		managed   *piManagedFile
		want      string
		suggested bool
	}{
		{name: "absent → install", diskHash: "", managed: nil, want: "install"},
		{name: "identical unmanaged → adopt", diskHash: sha256Hex([]byte(embeddedContent)), managed: nil, want: "adopt"},
		{name: "managed identical, same binary → current",
			diskHash: sha256Hex([]byte(embeddedContent)),
			managed:  &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: v2},
			want:     "current"},
		{name: "managed identical but older binary → update (stamp refresh)",
			diskHash: sha256Hex([]byte(embeddedContent)),
			managed:  &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: "v0.34.0"},
			want:     "update"},
		{name: "managed but user-modified → conflict, preserve", //nolint:dupl // table rows are intentionally parallel
			diskHash:  "deadbeef",
			managed:   &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: v2},
			want:      "conflict-user-modified",
			suggested: true},
		{name: "unmanaged different content → conflict, preserve", //nolint:dupl // sibling row
			diskHash:  "deadbeef",
			managed:   nil,
			want:      "conflict-unmanaged",
			suggested: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action, suggested := decidePiInstall("x.ts", []byte(embeddedContent), tc.diskHash, tc.managed, v2)
			if action != tc.want {
				t.Errorf("decidePiInstall(%s) action = %q, want %q", tc.name, action, tc.want)
			}
			if suggested != tc.suggested {
				t.Errorf("decidePiInstall(%s) suggested = %v, want %v", tc.name, suggested, tc.suggested)
			}
		})
	}
}

// v2 constant mirror for tests (Version is a build-time var)
const v2 = "v0.35.0"
