package coordinator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-CASCADE-ENVELOPE-HYDRATION: a cascade request must never reach a package
// agent without the envelope that makes it actionable.
//
// The cascade template (pkg-update.md) renders {{.RootPackage}}, the bump and
// the change class from the task, and tells the agent that an empty {{.Source}}
// means it was misrouted. Every route that delivered a cascade request without
// its envelope therefore produced the same outcome: a paid agent run that filed
// a "[bug] cascade routing: EMPTY dispatch" report and stopped. Measured on the
// test plane 2026-08-31..09-02: 627 such reports, when the backstop sweep
// re-enqueued unread docparse rows (no Source, no envelope) every pass.
//
// The envelope is carried only by the IAM-restricted cascade topic. A stored
// inbox row has no Source field, and its payload cannot stand in for one:
// any sender can write a package-schema payload to a pkg:* inbox, and the
// payload's change class uses the patch/minor/major labels, not the A/B/C
// taxonomy the wrapper's deterministic bump reads. So these guards refuse
// rather than reconstruct — a wrong Source="cascade" stamp would make the
// template's misroute guard lie.

// cascadeKinds are the package-message kinds that ask a dependent to take a
// dependency bump: the work the cascade template describes.
var cascadeKinds = map[string]bool{
	string(messaging.PkgMsgUpgradeAvailable): true,
	string(messaging.PkgMsgInterfaceChange):  true,
	string(messaging.PkgMsgEffectWidening):   true,
}

// cascadeShaped reports whether a message asks for a dependency bump.
//
// The kind is checked twice because the two routes spell it differently: a
// cascade-topic delivery carries it as the message_type attribute, while the
// stored row `ailang publish` writes has message_type "notification" and keeps
// the package kind only inside its payload. Keying on the attribute alone
// would miss exactly the stored rows the sweep recovers.
func cascadeShaped(kind, content string) bool {
	if cascadeKinds[kind] {
		return true
	}
	var peek struct {
		Schema string `json:"schema"`
		Kind   string `json:"kind"`
	}
	if json.Unmarshal([]byte(content), &peek) != nil {
		return false
	}
	return peek.Schema == messaging.PackageMessageSchema && cascadeKinds[peek.Kind]
}

// missingCascadeEnvelope names what a cascade request lacks, or returns "" when
// it carries what the cascade template needs: the authoritative source and the
// bump it describes.
func missingCascadeEnvelope(source, rootPackage, toVersion string) string {
	var missing []string
	if source != pubsub.SourceCascade {
		missing = append(missing, fmt.Sprintf("source=%q (want %q)", source, pubsub.SourceCascade))
	}
	if rootPackage == "" {
		missing = append(missing, "root_package")
	}
	if toVersion == "" {
		missing = append(missing, "to_version")
	}
	return strings.Join(missing, ", ")
}

// cascadeDispatchRefusal is the dispatch-time gate: a cascade-shaped task for a
// package agent with no envelope behind it is refused before any job exists.
// The adapter and the sweep already refuse the routes known to lose the
// envelope; this catches whichever one is found next, at the last point where
// refusing costs nothing.
//
// Returns nil for every other task, including package agents' ordinary bug and
// feature work, which never claimed to be a cascade.
func cascadeDispatchRefusal(task *TaskRecord, agentInbox string) error {
	if task == nil || !IsPackageInbox(agentInbox) || !cascadeShaped(task.Kind, task.Content) {
		return nil
	}
	missing := missingCascadeEnvelope(task.Source, task.RootPackage, task.ToVersion)
	if missing == "" {
		return nil
	}
	return fmt.Errorf("%w: task %s (message %s) is a cascade bump request for %s with no cascade envelope (missing %s); "+
		"the cascade template would render blank and the agent could only file a routing bug. "+
		"Re-trigger from the original publish, not from the stored row",
		ErrDispatchPermanent, task.ID, task.MessageID, agentInbox, missing)
}

// copyCascadeEnvelope copies the cascade envelope from the message a task is
// created from. Both task-creation sites call it: the local site once copied
// only Source, so a populated message lost its envelope there while the cloud
// site kept it — one concept, two implementations, one of them lossy.
func (t *TaskRecord) copyCascadeEnvelope(msg *Message) {
	t.RootPackage = msg.RootPackage
	t.RootChangeClass = msg.RootChangeClass
	t.FromVersion = msg.FromVersion
	t.ToVersion = msg.ToVersion
	t.FromInterfaceHash = msg.FromInterfaceHash
	t.ToInterfaceHash = msg.ToInterfaceHash
	t.FromContentHash = msg.FromContentHash
	t.ToContentHash = msg.ToContentHash
	t.EffectsWidened = msg.EffectsWidened
	t.PrevEffectCeiling = msg.PrevEffectCeiling
	t.NewEffectCeiling = msg.NewEffectCeiling
}
