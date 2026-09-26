// Package messaging provides the collaboration hub's messaging infrastructure.
// This file implements automatic event emission from package system operations
// for the package messaging graph (M-PKG-MSG).
package messaging

import (
	"fmt"
	"time"
)

// PackageVersionInfo holds the metadata needed to emit package events.
// Callers should populate this from manifests, lockfiles, or registry metadata.
type PackageVersionInfo struct {
	Name          string
	Version       string
	InterfaceHash string
	ContentHash   string
	Effects       []string
	Exports       []string
	// Signatures is the v2 export signature set. Nil or empty means legacy
	// metadata; non-empty means v2.
	//
	// PRECONDITION, relied on by classifyChange: the elements are DEDUPLICATED
	// and INJECTIVE — one string per exported symbol, distinct symbols never
	// colliding. The intended producer, pkg.InterfaceHashV2, satisfies both (it
	// collects behind a seen-set, and iface.SignatureSet escapes each field so
	// `mod:kind:name:sig` cannot collide). classifyChange compares these with SET
	// semantics, so a producer that violated the precondition by emitting a
	// duplicate would classify a real change as "A". Any future producer must
	// satisfy it; do not relax this comment without a test at this boundary.
	Signatures []string
}

// EmitUpgradeAvailable compares old and new package versions and emits an
// upgrade-available message if there are meaningful changes.
// Returns the sent message ID, or empty string if no message was needed.
func EmitUpgradeAvailable(store *Store, old, new PackageVersionInfo, recipients []string) (string, error) {
	if old.Version == new.Version && old.ContentHash == new.ContentHash {
		return "", nil // No change
	}

	changeClass := classifyChange(old, new)

	env := &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      PkgMsgUpgradeAvailable,
		From:      FormatPackageInbox(new.Name),
		To:        recipients,
		Timestamp: time.Now().UTC(),
		Package: PackageRef{
			Name:              new.Name,
			FromVersion:       old.Version,
			ToVersion:         new.Version,
			FromInterfaceHash: old.InterfaceHash,
			ToInterfaceHash:   new.InterfaceHash,
			FromContentHash:   old.ContentHash,
			ToContentHash:     new.ContentHash,
			ChangeClass:       changeClass,
			Breaking:          breakingFlag(old, new, changeClass),
		},
		Status: "open",
	}

	return sendPackageMessage(store, env)
}

// EmitInterfaceChangeNotice emits an interface-change-notice when a package's
// exported API has changed (interface hash delta).
func EmitInterfaceChangeNotice(store *Store, old, new PackageVersionInfo, recipients []string) (string, error) {
	if old.InterfaceHash == new.InterfaceHash {
		return "", nil // No interface change
	}

	changeClass := classifyChange(old, new)
	env := &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      PkgMsgInterfaceChange,
		From:      FormatPackageInbox(new.Name),
		To:        recipients,
		Timestamp: time.Now().UTC(),
		Package: PackageRef{
			Name:              new.Name,
			FromVersion:       old.Version,
			ToVersion:         new.Version,
			FromInterfaceHash: old.InterfaceHash,
			ToInterfaceHash:   new.InterfaceHash,
			ChangeClass:       changeClass,
			Breaking:          breakingFlag(old, new, changeClass),
		},
		Status: "open",
	}

	return sendPackageMessage(store, env)
}

// EmitEffectWideningWarning emits an effect-widening-warning when a package's
// effect ceiling has expanded.
func EmitEffectWideningWarning(store *Store, pkgName string, old, new PackageVersionInfo, recipients []string) (string, error) {
	widened := effectsWidened(old.Effects, new.Effects)
	if !widened {
		return "", nil // No widening
	}

	env := &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      PkgMsgEffectWidening,
		From:      FormatPackageInbox(pkgName),
		To:        recipients,
		Timestamp: time.Now().UTC(),
		Package: PackageRef{
			Name:              pkgName,
			FromVersion:       old.Version,
			ToVersion:         new.Version,
			PrevEffectCeiling: old.Effects,
			NewEffectCeiling:  new.Effects,
			FromContentHash:   old.ContentHash,
			ToContentHash:     new.ContentHash,
		},
		Status: "open",
	}

	return sendPackageMessage(store, env)
}

// EmitFromLockfileDiff compares two sets of locked packages and emits
// upgrade-available messages for each changed dependency.
// Returns the number of messages emitted.
func EmitFromLockfileDiff(store *Store, oldPkgs, newPkgs []PackageVersionInfo, workspace string) (int, error) {
	oldMap := make(map[string]PackageVersionInfo)
	for _, p := range oldPkgs {
		oldMap[p.Name] = p
	}

	recipient := FormatWorkspaceInbox(workspace)
	count := 0

	for _, newPkg := range newPkgs {
		oldPkg, existed := oldMap[newPkg.Name]
		if !existed {
			// New dependency added — not an upgrade, skip
			continue
		}
		if oldPkg.ContentHash == newPkg.ContentHash {
			continue // No change
		}

		_, err := EmitUpgradeAvailable(store, oldPkg, newPkg, []string{recipient})
		if err != nil {
			return count, fmt.Errorf("failed to emit upgrade for %s: %w", newPkg.Name, err)
		}
		count++

		// Also emit interface change notice if applicable
		if oldPkg.InterfaceHash != newPkg.InterfaceHash {
			_, err := EmitInterfaceChangeNotice(store, oldPkg, newPkg, []string{recipient})
			if err != nil {
				return count, fmt.Errorf("failed to emit interface change for %s: %w", newPkg.Name, err)
			}
		}

		// Check for effect widening
		if effectsWidened(oldPkg.Effects, newPkg.Effects) {
			_, err := EmitEffectWideningWarning(store, newPkg.Name, oldPkg, newPkg, []string{recipient})
			if err != nil {
				return count, fmt.Errorf("failed to emit effect warning for %s: %w", newPkg.Name, err)
			}
		}
	}

	return count, nil
}

// classifyChange determines the change class from signature sets when both
// sides have them, while preserving hash classification for legacy pairs.
//
//	A = internal only (content changed, interface same)
//	B = additive (new exports, existing interface unchanged)
//	C = contract change (interface hash changed)
//	U = unknown (only one side has signatures)
func classifyChange(old, new PackageVersionInfo) string {
	// Both nil and empty signature slices are legacy. An empty set cannot
	// establish that signature metadata was produced, so len is intentional.
	oldV2 := len(old.Signatures) > 0
	newV2 := len(new.Signatures) > 0
	if !oldV2 && !newV2 {
		return legacyClassify(old, new)
	}
	if !oldV2 && newV2 {
		return "U"
	}
	if oldV2 && !newV2 {
		return "U"
	}
	if sameSignatureSet(old.Signatures, new.Signatures) {
		return "A"
	}
	if isSuperset(new.Signatures, old.Signatures) {
		return "B"
	}
	return "C"
}

func legacyClassify(old, new PackageVersionInfo) string {
	if old.InterfaceHash != new.InterfaceHash {
		return "C"
	}
	return "A"
}

// breakingFlag reports the envelope's `breaking` field, and deliberately leaves
// it UNSET for a wholly-legacy pair.
//
// `breaking` is not merely descriptive: internal/coordinator's ClassifyChange
// short-circuits on it before it ever looks at the message kind, so a non-nil
// true flag routes the cascade to review instead of auto-apply. Setting it from
// the hash-only legacy classification would therefore flip EVERY
// interface-hash-changing publish from auto-apply to review on the day this
// lands — real blast radius, and zero benefit, because no producer emits
// signatures yet. Gating on signature presence keeps the legacy path
// byte-identical to the pre-M5 envelope (nil, omitted by `omitempty`) and lets
// the flag start carrying meaning exactly when a measured signature diff can
// justify it. Pinned by TestEmitUpgradeAvailable (legacy pair, nil) and
// TestEmitUpgradeAvailable_V2PairCarriesBreaking (v2 pair, true), and the
// routing consequence by TestAutonomyRouter_LegacyEnvelopeRoutingUnchanged.
func breakingFlag(old, new PackageVersionInfo, changeClass string) *bool {
	if len(old.Signatures) == 0 && len(new.Signatures) == 0 {
		return nil
	}
	breaking := changeClass == "C"
	return &breaking
}

func sameSignatureSet(a, b []string) bool {
	return isSuperset(a, b) && isSuperset(b, a)
}

func isSuperset(candidate, required []string) bool {
	set := make(map[string]struct{}, len(candidate))
	for _, signature := range candidate {
		set[signature] = struct{}{}
	}
	for _, signature := range required {
		if _, ok := set[signature]; !ok {
			return false
		}
	}
	return true
}

// effectsWidened returns true if newEffects contains effects not in oldEffects.
func effectsWidened(oldEffects, newEffects []string) bool {
	oldSet := make(map[string]bool)
	for _, e := range oldEffects {
		oldSet[e] = true
	}
	for _, e := range newEffects {
		if !oldSet[e] {
			return true
		}
	}
	return false
}

// sendPackageMessage validates and sends a package message through the store.
func sendPackageMessage(store *Store, env *PackageMessageEnvelope) (string, error) {
	if err := ValidatePackageMessage(env); err != nil {
		return "", fmt.Errorf("invalid package message: %w", err)
	}

	msg, err := env.ToInboxMessage()
	if err != nil {
		return "", err
	}

	if err := store.InsertInboxMessage(msg); err != nil {
		return "", fmt.Errorf("failed to send package message: %w", err)
	}

	return msg.MessageID, nil
}
