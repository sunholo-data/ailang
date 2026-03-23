// Package messaging provides the collaboration hub's messaging infrastructure.
// This file defines typed package coordination message schemas for the
// package messaging graph (M-PKG-MSG design doc).
package messaging

import (
	"encoding/json"
	"fmt"
	"time"
)

// PackageMessageSchema is the canonical schema version for package coordination messages.
const PackageMessageSchema = "ailang.package-message/v1"

// PackageMessageKind identifies the type of package coordination event.
type PackageMessageKind string

const (
	PkgMsgUpgradeAvailable    PackageMessageKind = "upgrade-available"
	PkgMsgInterfaceChange     PackageMessageKind = "interface-change-notice"
	PkgMsgEffectWidening      PackageMessageKind = "effect-widening-warning"
	PkgMsgCompatibilityReq    PackageMessageKind = "compatibility-request"
	PkgMsgCompatibilityReport PackageMessageKind = "compatibility-report"
	PkgMsgContractRegression  PackageMessageKind = "contract-regression"
	PkgMsgMigrationRequest    PackageMessageKind = "migration-request"
	PkgMsgDeprecationNotice   PackageMessageKind = "deprecation-notice"
	PkgMsgUpgradeComplete     PackageMessageKind = "upgrade-complete"
	PkgMsgBlocked             PackageMessageKind = "blocked"
	PkgMsgSuperseded          PackageMessageKind = "superseded"
)

// AllPackageMessageKinds lists all valid package message kinds.
var AllPackageMessageKinds = []PackageMessageKind{
	PkgMsgUpgradeAvailable,
	PkgMsgInterfaceChange,
	PkgMsgEffectWidening,
	PkgMsgCompatibilityReq,
	PkgMsgCompatibilityReport,
	PkgMsgContractRegression,
	PkgMsgMigrationRequest,
	PkgMsgDeprecationNotice,
	PkgMsgUpgradeComplete,
	PkgMsgBlocked,
	PkgMsgSuperseded,
}

// PackageMessageEnvelope is the canonical envelope for package coordination messages.
// It is stored as JSON in InboxMessage.Payload.
type PackageMessageEnvelope struct {
	Schema    string             `json:"schema"`
	MessageID string             `json:"message_id,omitempty"`
	Kind      PackageMessageKind `json:"kind"`
	From      string             `json:"from"`
	To        []string           `json:"to"`
	Timestamp time.Time          `json:"timestamp"`
	Package   PackageRef         `json:"package"`

	// Optional fields
	Summary           string          `json:"summary,omitempty"`
	RecommendedAction string          `json:"recommended_action,omitempty"`
	Refs              *PackageRefs    `json:"refs,omitempty"`
	Status            string          `json:"status,omitempty"`
	Supersedes        string          `json:"supersedes,omitempty"`
	RelatedMessages   []string        `json:"related_messages,omitempty"`
	Evidence          *CompatEvidence `json:"evidence,omitempty"`
	BlockReason       string          `json:"block_reason,omitempty"`
}

// PackageRef identifies a package and describes version/hash deltas.
type PackageRef struct {
	Name              string   `json:"name"`
	FromVersion       string   `json:"from_version,omitempty"`
	ToVersion         string   `json:"to_version,omitempty"`
	FromInterfaceHash string   `json:"from_interface_hash,omitempty"`
	ToInterfaceHash   string   `json:"to_interface_hash,omitempty"`
	FromContentHash   string   `json:"from_content_hash,omitempty"`
	ToContentHash     string   `json:"to_content_hash,omitempty"`
	ChangeClass       string   `json:"change_class,omitempty"`
	EffectDelta       []string `json:"effect_delta,omitempty"`
	Breaking          *bool    `json:"breaking,omitempty"`

	// For effect-widening-warning
	PrevEffectCeiling []string `json:"prev_effect_ceiling,omitempty"`
	NewEffectCeiling  []string `json:"new_effect_ceiling,omitempty"`

	// For contract-regression
	AffectedExports []string `json:"affected_exports,omitempty"`
	PrevContract    string   `json:"prev_contract,omitempty"`
	NewContract     string   `json:"new_contract,omitempty"`

	// For compatibility-report
	Result          string `json:"result,omitempty"` // "pass", "fail", "partial"
	TargetWorkspace string `json:"target_workspace,omitempty"`
}

// PackageRefs holds optional reference links.
type PackageRefs struct {
	PackageURL   string `json:"package_url,omitempty"`
	ReleaseNotes string `json:"release_notes,omitempty"`
	LockfileRef  string `json:"lockfile_ref,omitempty"`
}

// CompatEvidence holds structured evidence for compatibility reports.
type CompatEvidence struct {
	FailingExports     []string `json:"failing_exports,omitempty"`
	ContractViolations []string `json:"contract_violations,omitempty"`
	LockfileSnapshot   string   `json:"lockfile_snapshot,omitempty"`
	Summary            string   `json:"summary,omitempty"`
}

// ValidatePackageMessage checks that a PackageMessageEnvelope has all required
// fields for its kind. Returns nil if valid.
func ValidatePackageMessage(env *PackageMessageEnvelope) error {
	if env.Schema != PackageMessageSchema {
		return fmt.Errorf("invalid schema: got %q, want %q", env.Schema, PackageMessageSchema)
	}
	if env.Kind == "" {
		return fmt.Errorf("missing required field: kind")
	}
	if !isValidPkgMsgKind(env.Kind) {
		return fmt.Errorf("unknown package message kind: %q", env.Kind)
	}
	if env.From == "" {
		return fmt.Errorf("missing required field: from")
	}
	if len(env.To) == 0 {
		return fmt.Errorf("missing required field: to (must have at least one recipient)")
	}
	if env.Timestamp.IsZero() {
		return fmt.Errorf("missing required field: timestamp")
	}
	if env.Package.Name == "" {
		return fmt.Errorf("missing required field: package.name")
	}

	// Must have at least one delta: version, interface hash, or content hash
	hasVersionDelta := env.Package.FromVersion != "" || env.Package.ToVersion != ""
	hasInterfaceDelta := env.Package.FromInterfaceHash != "" || env.Package.ToInterfaceHash != ""
	hasContentDelta := env.Package.FromContentHash != "" || env.Package.ToContentHash != ""
	if !hasVersionDelta && !hasInterfaceDelta && !hasContentDelta {
		return fmt.Errorf("package must have at least one delta (version, interface hash, or content hash)")
	}

	// Kind-specific validation
	switch env.Kind {
	case PkgMsgUpgradeAvailable:
		return validateUpgradeAvailable(env)
	case PkgMsgInterfaceChange:
		return validateInterfaceChange(env)
	case PkgMsgEffectWidening:
		return validateEffectWidening(env)
	case PkgMsgCompatibilityReport:
		return validateCompatibilityReport(env)
	case PkgMsgContractRegression:
		return validateContractRegression(env)
	case PkgMsgMigrationRequest:
		return validateMigrationRequest(env)
	case PkgMsgCompatibilityReq, PkgMsgDeprecationNotice,
		PkgMsgUpgradeComplete, PkgMsgBlocked, PkgMsgSuperseded:
		// Base validation is sufficient for these kinds
		return nil
	}
	return nil
}

func validateUpgradeAvailable(env *PackageMessageEnvelope) error {
	if env.Package.FromVersion == "" || env.Package.ToVersion == "" {
		return fmt.Errorf("upgrade-available requires from_version and to_version")
	}
	if env.Package.FromInterfaceHash == "" || env.Package.ToInterfaceHash == "" {
		return fmt.Errorf("upgrade-available requires from_interface_hash and to_interface_hash")
	}
	if env.Package.ChangeClass == "" {
		return fmt.Errorf("upgrade-available requires change_class")
	}
	return nil
}

func validateInterfaceChange(env *PackageMessageEnvelope) error {
	if env.Package.FromInterfaceHash == "" || env.Package.ToInterfaceHash == "" {
		return fmt.Errorf("interface-change-notice requires from_interface_hash and to_interface_hash")
	}
	return nil
}

func validateEffectWidening(env *PackageMessageEnvelope) error {
	if len(env.Package.PrevEffectCeiling) == 0 || len(env.Package.NewEffectCeiling) == 0 {
		return fmt.Errorf("effect-widening-warning requires prev_effect_ceiling and new_effect_ceiling")
	}
	return nil
}

func validateCompatibilityReport(env *PackageMessageEnvelope) error {
	if env.Package.FromVersion == "" || env.Package.ToVersion == "" {
		return fmt.Errorf("compatibility-report requires from_version and to_version")
	}
	if env.Package.TargetWorkspace == "" {
		return fmt.Errorf("compatibility-report requires target_workspace")
	}
	if env.Package.Result == "" {
		return fmt.Errorf("compatibility-report requires result (pass, fail, or partial)")
	}
	validResults := map[string]bool{"pass": true, "fail": true, "partial": true}
	if !validResults[env.Package.Result] {
		return fmt.Errorf("compatibility-report result must be pass, fail, or partial; got %q", env.Package.Result)
	}
	return nil
}

func validateContractRegression(env *PackageMessageEnvelope) error {
	if len(env.Package.AffectedExports) == 0 {
		return fmt.Errorf("contract-regression requires affected_exports")
	}
	if env.Package.PrevContract == "" {
		return fmt.Errorf("contract-regression requires prev_contract")
	}
	return nil
}

func validateMigrationRequest(env *PackageMessageEnvelope) error {
	if env.Package.FromVersion == "" || env.Package.ToVersion == "" {
		return fmt.Errorf("migration-request requires from_version and to_version")
	}
	if env.BlockReason == "" {
		return fmt.Errorf("migration-request requires block_reason")
	}
	return nil
}

func isValidPkgMsgKind(kind PackageMessageKind) bool {
	for _, k := range AllPackageMessageKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// PackageEnvelopeToJSON serializes a PackageMessageEnvelope to JSON.
func PackageEnvelopeToJSON(env *PackageMessageEnvelope) (string, error) {
	data, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("failed to marshal package message: %w", err)
	}
	return string(data), nil
}

// PackageEnvelopeFromJSON deserializes a PackageMessageEnvelope from JSON.
func PackageEnvelopeFromJSON(data string) (*PackageMessageEnvelope, error) {
	var env PackageMessageEnvelope
	if err := json.Unmarshal([]byte(data), &env); err != nil {
		return nil, fmt.Errorf("failed to unmarshal package message: %w", err)
	}
	return &env, nil
}

// ToInboxMessage converts a PackageMessageEnvelope to an InboxMessage
// suitable for storage in the messaging system. The envelope is serialized
// into the Payload field as JSON.
func (env *PackageMessageEnvelope) ToInboxMessage() (*InboxMessage, error) {
	payload, err := PackageEnvelopeToJSON(env)
	if err != nil {
		return nil, err
	}

	msg := &InboxMessage{
		FromAgent:   env.From,
		ToInbox:     env.To[0], // Primary recipient; additional recipients sent separately
		MessageType: InboxTypeNotification,
		Title:       fmt.Sprintf("[%s] %s", env.Kind, env.Package.Name),
		Payload:     payload,
		Category:    CategoryGeneral,
		Status:      InboxStatusUnread,
		CreatedAt:   env.Timestamp,
	}

	// Set category based on message kind
	switch env.Kind {
	case PkgMsgContractRegression, PkgMsgBlocked:
		msg.Category = CategoryBug
	case PkgMsgUpgradeAvailable, PkgMsgInterfaceChange, PkgMsgDeprecationNotice:
		msg.Category = CategoryFeature
	}

	return msg, nil
}

// ExtractPackageEnvelope attempts to parse a PackageMessageEnvelope from
// an InboxMessage's Payload. Returns nil, nil if the payload is not a
// package message (no schema field or wrong schema).
func ExtractPackageEnvelope(msg *InboxMessage) (*PackageMessageEnvelope, error) {
	if msg.Payload == "" {
		return nil, nil
	}

	// Quick check: does it look like a package message?
	var peek struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal([]byte(msg.Payload), &peek); err != nil {
		return nil, nil // Not JSON or not a package message
	}
	if peek.Schema != PackageMessageSchema {
		return nil, nil // Not a package message
	}

	env, err := PackageEnvelopeFromJSON(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("payload has package schema but failed to parse: %w", err)
	}
	return env, nil
}
