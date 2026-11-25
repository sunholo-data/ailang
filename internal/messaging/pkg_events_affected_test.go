package messaging

import (
	"reflect"
	"testing"
)

func v2(version string, sigs ...string) PackageVersionInfo {
	return PackageVersionInfo{Name: "sunholo/core", Version: version, InterfaceHash: "h-" + version,
		ContentHash: "c-" + version, Signatures: append(sigs, SignatureContractsMarker)}
}

// TST-PKGPAR-01
// An additive v2 change is B and not breaking, so the cascade can apply it
// deterministically.
func TestUpgradeAvailable_AdditiveIsBNotBreaking(t *testing.T) {
	msg, err := UpgradeAvailableMessage(v2("1.0.0", "m:func:a:int:"), v2("1.1.0", "m:func:a:int:", "m:func:b:int:"), []string{"pkg:x/y"}, "")
	if err != nil || msg == nil {
		t.Fatalf("message: %v %v", msg, err)
	}
	env := mustEnvelope(t, msg)
	if env.Package.ChangeClass != "B" || env.Package.Breaking == nil || *env.Package.Breaking {
		t.Fatalf("class=%q breaking=%v, want B/false", env.Package.ChangeClass, env.Package.Breaking)
	}
	if len(env.Package.AffectedExports) != 0 {
		t.Errorf("an addition affects no existing export: %v", env.Package.AffectedExports)
	}
}

// TST-PKGPAR-02
// A removal is C, and affected_exports names exactly what went away: the
// dependent's agent learns what broke, not only that something did.
func TestUpgradeAvailable_RemovalIsCAndNamed(t *testing.T) {
	old := v2("1.0.0", "m:func:a:int:", "m:func:b:int:", "m:contract:a:requires (n > 0)")
	new := v2("2.0.0", "m:func:a:int:", "m:contract:a:requires (n >= 0)")
	msg, err := UpgradeAvailableMessage(old, new, []string{"pkg:x/y"}, "")
	if err != nil || msg == nil {
		t.Fatalf("message: %v %v", msg, err)
	}
	env := mustEnvelope(t, msg)
	if env.Package.ChangeClass != "C" || env.Package.Breaking == nil || !*env.Package.Breaking {
		t.Fatalf("class=%q breaking=%v, want C/true", env.Package.ChangeClass, env.Package.Breaking)
	}
	want := []string{"m:func:b:int:", "m:contract:a:requires (n > 0)"}
	if !reflect.DeepEqual(env.Package.AffectedExports, want) {
		t.Errorf("affected_exports = %#v, want %#v", env.Package.AffectedExports, want)
	}
	if affectedExports(PackageVersionInfo{}, new) != nil {
		t.Error("a legacy side must leave affected_exports unset")
	}
}

func mustEnvelope(t *testing.T, msg *InboxMessage) *PackageMessageEnvelope {
	t.Helper()
	env, err := ExtractPackageEnvelope(msg)
	if err != nil {
		t.Fatalf("parse envelope: %v", err)
	}
	return env
}
