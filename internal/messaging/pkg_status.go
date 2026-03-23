// Package messaging provides the collaboration hub's messaging infrastructure.
// This file implements message lifecycle management and triage for the
// package messaging graph (M-PKG-MSG).
package messaging

import (
	"fmt"
	"time"
)

// Package message lifecycle states
const (
	PkgStatusOpen         = "open"
	PkgStatusAcknowledged = "acknowledged"
	PkgStatusInProgress   = "in_progress"
	PkgStatusBlocked      = "blocked"
	PkgStatusCompleted    = "completed"
	PkgStatusRejected     = "rejected"
	PkgStatusSuperseded   = "superseded"
)

// validTransitions defines allowed status transitions for package messages.
var validTransitions = map[string][]string{
	PkgStatusOpen:         {PkgStatusAcknowledged, PkgStatusRejected, PkgStatusSuperseded},
	PkgStatusAcknowledged: {PkgStatusInProgress, PkgStatusRejected, PkgStatusSuperseded},
	PkgStatusInProgress:   {PkgStatusCompleted, PkgStatusBlocked, PkgStatusRejected},
	PkgStatusBlocked:      {PkgStatusInProgress, PkgStatusRejected, PkgStatusSuperseded},
	PkgStatusCompleted:    {}, // Terminal
	PkgStatusRejected:     {}, // Terminal
	PkgStatusSuperseded:   {}, // Terminal
}

// UpdatePackageMessageStatus updates the status of a package message,
// enforcing valid lifecycle transitions. The status is stored in the
// message payload's "status" field.
func (s *Store) UpdatePackageMessageStatus(msgID, newStatus string) error {
	msg, err := s.GetInboxMessage(msgID)
	if err != nil {
		return fmt.Errorf("failed to get message %s: %w", msgID, err)
	}

	env, err := ExtractPackageEnvelope(msg)
	if err != nil {
		return fmt.Errorf("failed to extract package envelope: %w", err)
	}
	if env == nil {
		return fmt.Errorf("message %s is not a package message", msgID)
	}

	currentStatus := env.Status
	if currentStatus == "" {
		currentStatus = PkgStatusOpen
	}

	if !isValidTransition(currentStatus, newStatus) {
		return fmt.Errorf("invalid status transition: %s → %s (allowed: %v)",
			currentStatus, newStatus, validTransitions[currentStatus])
	}

	env.Status = newStatus
	payload, err := PackageEnvelopeToJSON(env)
	if err != nil {
		return fmt.Errorf("failed to serialize updated envelope: %w", err)
	}

	_, err = s.db.Exec(
		`UPDATE inbox_messages SET payload = ? WHERE id = ? OR message_id = ?`,
		payload, msgID, msgID,
	)
	return err
}

// SupersedeOlderMessages marks older upgrade-available messages for the same
// package as superseded when a newer version is published.
func (s *Store) SupersedeOlderMessages(pkgName, newVersion string) (int, error) {
	// Find all unresolved messages for this package
	msgs, err := s.ListInboxMessages(InboxListOptions{
		Inbox: FormatPackageInbox(pkgName),
		Limit: 100,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list messages for %s: %w", pkgName, err)
	}

	count := 0
	for _, msg := range msgs {
		env, err := ExtractPackageEnvelope(&msg)
		if err != nil || env == nil {
			continue
		}

		// Only supersede upgrade-available and interface-change-notice
		if env.Kind != PkgMsgUpgradeAvailable && env.Kind != PkgMsgInterfaceChange {
			continue
		}

		// Don't supersede messages about the current version
		if env.Package.ToVersion == newVersion {
			continue
		}

		// Only supersede open or acknowledged messages
		if env.Status != PkgStatusOpen && env.Status != PkgStatusAcknowledged && env.Status != "" {
			continue
		}

		if err := s.UpdatePackageMessageStatus(msg.ID, PkgStatusSuperseded); err != nil {
			continue // Best effort
		}
		count++
	}

	return count, nil
}

// TriageActionability classifies what action a package message requires.
type TriageActionability string

const (
	TriageNoAction    TriageActionability = "no_action"
	TriageVerifyLocal TriageActionability = "verify_local"
	TriageMigrate     TriageActionability = "migrate"
	TriageEscalate    TriageActionability = "escalate"
	TriagePolicyBlock TriageActionability = "policy_block"
)

// TriageResult holds the triage classification for a package message.
type TriageResult struct {
	Action      TriageActionability `json:"action"`
	Reason      string              `json:"reason"`
	MessageKind PackageMessageKind  `json:"message_kind"`
	PackageName string              `json:"package_name"`
}

// TriagePackageMessage classifies the actionability of a package message.
func TriagePackageMessage(env *PackageMessageEnvelope) TriageResult {
	result := TriageResult{
		MessageKind: env.Kind,
		PackageName: env.Package.Name,
	}

	switch env.Kind {
	case PkgMsgUpgradeAvailable:
		if env.Package.ChangeClass == "A" {
			result.Action = TriageNoAction
			result.Reason = "Internal-only change; no downstream impact"
		} else if env.Package.ChangeClass == "C" {
			result.Action = TriageMigrate
			result.Reason = "Contract change; downstream verification required"
		} else {
			result.Action = TriageVerifyLocal
			result.Reason = "Content change; local verification recommended"
		}

	case PkgMsgInterfaceChange:
		result.Action = TriageVerifyLocal
		result.Reason = "Interface hash changed; check export compatibility"

	case PkgMsgEffectWidening:
		result.Action = TriagePolicyBlock
		result.Reason = "Effect ceiling widened; policy review required"

	case PkgMsgContractRegression:
		result.Action = TriageEscalate
		result.Reason = "Previously working contract broken; escalate to maintainer"

	case PkgMsgCompatibilityReport:
		if env.Package.Result == "pass" {
			result.Action = TriageNoAction
			result.Reason = "Compatibility verified"
		} else {
			result.Action = TriageMigrate
			result.Reason = fmt.Sprintf("Compatibility check result: %s", env.Package.Result)
		}

	case PkgMsgBlocked:
		result.Action = TriageEscalate
		result.Reason = "Migration blocked"

	case PkgMsgDeprecationNotice:
		result.Action = TriageMigrate
		result.Reason = "API deprecated; migration planning needed"

	case PkgMsgSuperseded:
		result.Action = TriageNoAction
		result.Reason = "Superseded by newer message"

	default:
		result.Action = TriageVerifyLocal
		result.Reason = "Standard coordination event"
	}

	return result
}

// DeduplicatePackageReports finds and marks duplicate compatibility reports
// for the same package + version combination.
func (s *Store) DeduplicatePackageReports(pkgName string) (int, error) {
	msgs, err := s.ListInboxMessages(InboxListOptions{
		Limit: 200,
	})
	if err != nil {
		return 0, err
	}

	// Group compatibility reports by package + version pair
	type reportKey struct {
		fromVersion string
		toVersion   string
		workspace   string
	}
	groups := make(map[reportKey][]string) // key → message IDs

	for _, msg := range msgs {
		env, err := ExtractPackageEnvelope(&msg)
		if err != nil || env == nil {
			continue
		}
		if env.Package.Name != pkgName || env.Kind != PkgMsgCompatibilityReport {
			continue
		}

		key := reportKey{
			fromVersion: env.Package.FromVersion,
			toVersion:   env.Package.ToVersion,
			workspace:   env.Package.TargetWorkspace,
		}
		groups[key] = append(groups[key], msg.ID)
	}

	// Mark duplicates (keep first, mark rest)
	deduped := 0
	for _, ids := range groups {
		if len(ids) <= 1 {
			continue
		}
		for _, dupID := range ids[1:] {
			_, err := s.db.Exec(
				`UPDATE inbox_messages SET dup_of = ? WHERE id = ?`,
				ids[0], dupID,
			)
			if err == nil {
				deduped++
			}
		}
	}

	return deduped, nil
}

// PackageMessageStats returns aggregate stats for package messages.
type PackageMessageStats struct {
	TotalMessages     int            `json:"total_messages"`
	ByKind            map[string]int `json:"by_kind"`
	ByStatus          map[string]int `json:"by_status"`
	OpenUpgrades      int            `json:"open_upgrades"`
	BlockedMigrations int            `json:"blocked_migrations"`
	LastActivity      *time.Time     `json:"last_activity,omitempty"`
}

// GetPackageMessageStats returns statistics about package messages for a given package.
func (s *Store) GetPackageMessageStats(pkgName string) (*PackageMessageStats, error) {
	inbox := FormatPackageInbox(pkgName)
	msgs, err := s.ListInboxMessages(InboxListOptions{
		Inbox: inbox,
		Limit: 500,
	})
	if err != nil {
		return nil, err
	}

	stats := &PackageMessageStats{
		ByKind:   make(map[string]int),
		ByStatus: make(map[string]int),
	}

	for _, msg := range msgs {
		stats.TotalMessages++
		if stats.LastActivity == nil || msg.CreatedAt.After(*stats.LastActivity) {
			t := msg.CreatedAt
			stats.LastActivity = &t
		}

		env, err := ExtractPackageEnvelope(&msg)
		if err != nil || env == nil {
			continue
		}

		stats.ByKind[string(env.Kind)]++
		status := env.Status
		if status == "" {
			status = PkgStatusOpen
		}
		stats.ByStatus[status]++

		if env.Kind == PkgMsgUpgradeAvailable && (status == PkgStatusOpen || status == PkgStatusAcknowledged) {
			stats.OpenUpgrades++
		}
		if status == PkgStatusBlocked {
			stats.BlockedMigrations++
		}
	}

	return stats, nil
}

func isValidTransition(from, to string) bool {
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}
