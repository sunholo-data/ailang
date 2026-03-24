package coordinator

import (
	"github.com/sunholo/ailang/internal/messaging"
)

// ChangeClass represents the severity level of a package update.
type ChangeClass int

const (
	// ChangeClassA is a patch/internal change — fully autonomous.
	ChangeClassA ChangeClass = iota
	// ChangeClassB is an additive/minor change — semi-autonomous.
	ChangeClassB
	// ChangeClassC is a breaking/major change — requires human approval.
	ChangeClassC
)

// AdjustAutonomyForChangeClass modifies an agent's approval settings based on
// the incoming package message's change class. Returns a copy of the config
// with adjusted autonomy levels. Non-package messages pass through unchanged.
func AdjustAutonomyForChangeClass(agent *AgentConfig, msg *messaging.InboxMessage) *AgentConfig {
	if agent == nil || msg == nil {
		return agent
	}

	env, err := messaging.ExtractPackageEnvelope(msg)
	if err != nil || env == nil {
		return agent // Not a package message, use defaults
	}

	effective := *agent // Copy
	switch ClassifyChange(env) {
	case ChangeClassA:
		effective.SkipApproval = true
		effective.AutoMerge = true
		effective.AutoApproveHandoffs = true
	case ChangeClassB:
		effective.SkipApproval = false
		effective.AutoMerge = false
		effective.AutoApproveHandoffs = true
	case ChangeClassC:
		effective.SkipApproval = false
		effective.AutoMerge = false
		effective.AutoApproveHandoffs = false
	}
	return &effective
}

// ClassifyChange determines the change class from a package message envelope.
func ClassifyChange(env *messaging.PackageMessageEnvelope) ChangeClass {
	if env.Package.Breaking != nil && *env.Package.Breaking {
		return ChangeClassC
	}

	switch env.Kind {
	case messaging.PkgMsgEffectWidening:
		return ChangeClassC // Effect ceiling changes always need review
	case messaging.PkgMsgContractRegression:
		return ChangeClassC // Broken contracts always need review
	case messaging.PkgMsgInterfaceChange:
		if env.Package.ChangeClass == "major" {
			return ChangeClassC
		}
		return ChangeClassB
	case messaging.PkgMsgUpgradeAvailable:
		if env.Package.ChangeClass == "minor" {
			return ChangeClassB
		}
		return ChangeClassA
	case messaging.PkgMsgDeprecationNotice:
		return ChangeClassB
	case messaging.PkgMsgMigrationRequest:
		return ChangeClassB
	case messaging.PkgMsgCompatibilityReq:
		return ChangeClassA
	case messaging.PkgMsgCompatibilityReport:
		return ChangeClassA
	case messaging.PkgMsgUpgradeComplete:
		return ChangeClassA
	case messaging.PkgMsgBlocked:
		return ChangeClassB
	case messaging.PkgMsgSuperseded:
		return ChangeClassA
	default:
		return ChangeClassB // Conservative default
	}
}
