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

// classifyChange determines the change class based on hash comparison.
//
//	A = internal only (content changed, interface same)
//	B = additive (new exports, existing interface unchanged)
//	C = contract change (interface hash changed)
func classifyChange(old, new PackageVersionInfo) string {
	if old.InterfaceHash != new.InterfaceHash {
		return "C"
	}
	if old.ContentHash != new.ContentHash {
		return "A"
	}
	return "A"
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
