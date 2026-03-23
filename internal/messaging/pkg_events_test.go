package messaging

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestEmitUpgradeAvailable(t *testing.T) {
	store := newTestStore(t)

	old := PackageVersionInfo{
		Name:          "sunholo/auth",
		Version:       "0.1.0",
		InterfaceHash: "sha256:aaa",
		ContentHash:   "sha256:ccc",
		Effects:       []string{"io"},
	}
	new := PackageVersionInfo{
		Name:          "sunholo/auth",
		Version:       "0.2.0",
		InterfaceHash: "sha256:bbb",
		ContentHash:   "sha256:ddd",
		Effects:       []string{"io"},
	}

	msgID, err := EmitUpgradeAvailable(store, old, new, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("EmitUpgradeAvailable failed: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected non-empty message ID")
	}

	// Verify the message was stored
	msgs, err := store.ListInboxMessages(InboxListOptions{
		Inbox: "workspace:docparse",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListInboxMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	// Verify payload
	env, err := ExtractPackageEnvelope(&msgs[0])
	if err != nil {
		t.Fatalf("ExtractPackageEnvelope failed: %v", err)
	}
	if env.Kind != PkgMsgUpgradeAvailable {
		t.Errorf("kind: got %q, want upgrade-available", env.Kind)
	}
	if env.Package.ChangeClass != "C" {
		t.Errorf("change_class: got %q, want C (interface hash changed)", env.Package.ChangeClass)
	}
}

func TestEmitUpgradeAvailable_NoChange(t *testing.T) {
	store := newTestStore(t)

	info := PackageVersionInfo{
		Name:        "sunholo/auth",
		Version:     "0.1.0",
		ContentHash: "sha256:same",
	}

	msgID, err := EmitUpgradeAvailable(store, info, info, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "" {
		t.Error("expected no message for identical versions")
	}
}

func TestEmitInterfaceChangeNotice(t *testing.T) {
	store := newTestStore(t)

	old := PackageVersionInfo{
		Name:          "sunholo/auth",
		Version:       "0.1.0",
		InterfaceHash: "sha256:old",
		ContentHash:   "sha256:c1",
	}
	new := PackageVersionInfo{
		Name:          "sunholo/auth",
		Version:       "0.2.0",
		InterfaceHash: "sha256:new",
		ContentHash:   "sha256:c2",
	}

	msgID, err := EmitInterfaceChangeNotice(store, old, new, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected message for interface change")
	}

	// No change should skip
	msgID2, err := EmitInterfaceChangeNotice(store, old, old, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID2 != "" {
		t.Error("expected no message for same interface hash")
	}
}

func TestEmitEffectWideningWarning(t *testing.T) {
	store := newTestStore(t)

	old := PackageVersionInfo{
		Name:        "sunholo/auth",
		Version:     "0.1.0",
		Effects:     []string{"io"},
		ContentHash: "sha256:c1",
	}
	new := PackageVersionInfo{
		Name:        "sunholo/auth",
		Version:     "0.2.0",
		Effects:     []string{"io", "net"},
		ContentHash: "sha256:c2",
	}

	msgID, err := EmitEffectWideningWarning(store, "sunholo/auth", old, new, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if msgID == "" {
		t.Fatal("expected warning for effect widening")
	}

	// No widening should skip
	msgID2, err := EmitEffectWideningWarning(store, "sunholo/auth", old, old, []string{"workspace:docparse"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID2 != "" {
		t.Error("expected no message for same effects")
	}
}

func TestEmitFromLockfileDiff(t *testing.T) {
	store := newTestStore(t)

	oldPkgs := []PackageVersionInfo{
		{Name: "sunholo/auth", Version: "0.1.0", InterfaceHash: "sha256:a1", ContentHash: "sha256:c1", Effects: []string{"io"}},
		{Name: "sunholo/json", Version: "0.3.0", InterfaceHash: "sha256:j1", ContentHash: "sha256:j1"},
	}
	newPkgs := []PackageVersionInfo{
		{Name: "sunholo/auth", Version: "0.2.0", InterfaceHash: "sha256:a2", ContentHash: "sha256:c2", Effects: []string{"io", "net"}},
		{Name: "sunholo/json", Version: "0.3.0", InterfaceHash: "sha256:j1", ContentHash: "sha256:j1"},     // unchanged
		{Name: "sunholo/config", Version: "0.1.0", InterfaceHash: "sha256:cfg", ContentHash: "sha256:cfg"}, // new dep
	}

	count, err := EmitFromLockfileDiff(store, oldPkgs, newPkgs, "docparse")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 upgrade message (auth changed), got %d", count)
	}

	// Verify messages exist for the workspace
	msgs, err := store.ListInboxMessages(InboxListOptions{
		Inbox: "workspace:docparse",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	// Should have: upgrade-available + interface-change-notice + effect-widening-warning
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages (upgrade + interface + effect), got %d", len(msgs))
	}
}

func TestClassifyChange(t *testing.T) {
	tests := []struct {
		name     string
		old, new PackageVersionInfo
		want     string
	}{
		{"interface changed", PackageVersionInfo{InterfaceHash: "a"}, PackageVersionInfo{InterfaceHash: "b"}, "C"},
		{"content only", PackageVersionInfo{InterfaceHash: "same", ContentHash: "a"}, PackageVersionInfo{InterfaceHash: "same", ContentHash: "b"}, "A"},
		{"no change", PackageVersionInfo{InterfaceHash: "same", ContentHash: "same"}, PackageVersionInfo{InterfaceHash: "same", ContentHash: "same"}, "A"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyChange(tc.old, tc.new)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEffectsWidened(t *testing.T) {
	tests := []struct {
		name     string
		old, new []string
		want     bool
	}{
		{"widened", []string{"io"}, []string{"io", "net"}, true},
		{"same", []string{"io", "net"}, []string{"io", "net"}, false},
		{"narrowed", []string{"io", "net"}, []string{"io"}, false},
		{"empty to some", nil, []string{"io"}, true},
		{"some to empty", []string{"io"}, nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectsWidened(tc.old, tc.new); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
