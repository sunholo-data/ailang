package coordinator

import (
	"log"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-CASCADE-ENVELOPE-HYDRATION.
//
// A cascade directive reaches a pkg: agent through several routes — the Pub/Sub
// push path with an embedded envelope, the same push path with legacy-shaped
// data, and the backstop sweep when push never delivered at all. Only the first
// route populated the cascade envelope; every other route silently produced a
// task whose {{.Source}}, {{.RootPackage}} and version/hash fields rendered
// blank, the pkg-update template's misroute guard fired, and the agent burned a
// full task budget filing a `[bug] cascade routing: EMPTY dispatch` report
// (measured 2026-09-02 → 2026-09-27: 173 such reports in the last 5000 store
// messages alone). This file is the shared fix: hydrate the envelope from data
// that is ALREADY durable, and never dispatch a cascade directive without one.

// cascadeEnvelope is the cascade routing data recoverable for one message,
// gathered from whatever durable evidence exists. Zero value = no evidence.
type cascadeEnvelope struct {
	RootPackage       string
	ChangeClass       string
	FromVersion       string
	ToVersion         string
	FromInterfaceHash string
	ToInterfaceHash   string
	FromContentHash   string
	ToContentHash     string
	EffectsWidened    bool
	PrevEffectCeiling []string
	NewEffectCeiling  []string
}

// cascadeEnvelopeFromPayload parses a PackageMessageEnvelope out of the
// message's durable payload. The legacy inbox path stores the full envelope as
// the payload (messaging.UpgradeAvailableMessage → ToInboxMessage), so every
// bump that ever reached a store can be hydrated from it. Returns a zero
// envelope when the payload is not a package message; a parse error on a
// payload that LOOKS like a package message is logged and treated as no
// evidence rather than trusted.
func cascadeEnvelopeFromPayload(payload string, logger *log.Logger) cascadeEnvelope {
	env := cascadeEnvelope{}
	if payload == "" {
		return env
	}
	pkgEnv, err := messaging.ExtractPackageEnvelope(&messaging.InboxMessage{Payload: payload})
	if err != nil {
		if logger != nil {
			logger.Printf("cascade hydration: payload looks like a package message but does not parse (%v); treating as no envelope evidence", err)
		}
		return env
	}
	if pkgEnv == nil {
		return env
	}
	env.RootPackage = pkgEnv.Package.Name
	env.ChangeClass = pkgEnv.Package.ChangeClass
	env.FromVersion = pkgEnv.Package.FromVersion
	env.ToVersion = pkgEnv.Package.ToVersion
	env.FromInterfaceHash = pkgEnv.Package.FromInterfaceHash
	env.ToInterfaceHash = pkgEnv.Package.ToInterfaceHash
	env.FromContentHash = pkgEnv.Package.FromContentHash
	env.ToContentHash = pkgEnv.Package.ToContentHash
	env.EffectsWidened = effectCeilingsWidened(pkgEnv.Package.PrevEffectCeiling, pkgEnv.Package.NewEffectCeiling)
	env.PrevEffectCeiling = append([]string(nil), pkgEnv.Package.PrevEffectCeiling...)
	env.NewEffectCeiling = append([]string(nil), pkgEnv.Package.NewEffectCeiling...)
	return env
}

// effectCeilingsWidened reports whether newCeiling contains effects absent
// from prevCeiling. Mirrors messaging.effectsWidened (unexported there); the
// payload envelope carries no boolean, so widening is derived from the
// ceilings it does carry.
func effectCeilingsWidened(prevCeiling, newCeiling []string) bool {
	oldSet := make(map[string]bool, len(prevCeiling))
	for _, e := range prevCeiling {
		oldSet[e] = true
	}
	for _, e := range newCeiling {
		if !oldSet[e] {
			return true
		}
	}
	return false
}

// hydrateCascadeFields populates msg's Source and cascade envelope fields from
// durable evidence, in strict evidence order — an embedded envelope the caller
// already decoded beats Pub/Sub attributes beats the stored payload. Fields
// that are already set on msg are NEVER overwritten (the caller's earlier,
// stronger evidence wins); a conflict between two non-empty sources is logged
// loudly rather than silently resolved, because a disagreement means one of
// the two copies is corrupt.
//
// Source is stamped "cascade" ONLY when the routing attributes assert cascade
// authority (the message arrived on the IAM-restricted cascade topic). A
// payload that merely looks like a bump is not authority: stamping Source from
// payload shape alone would make the pkg-update template's misroute guard lie,
// which is worse than an empty Source (M-PKG-AUTONOMOUS-CASCADE-SAFE M1).
//
// Returns msg for call-site convenience; mutation is in place.
func hydrateCascadeFields(msg *Message, attrs pubsub.MessageAttributes, rawAttrs map[string]string, payload string, logger *log.Logger) *Message {
	if msg == nil {
		return nil
	}

	fromPayload := cascadeEnvelopeFromPayload(payload, logger)
	fromAttrs := cascadeEnvelope{
		RootPackage:    rawAttrs["root_package"],
		ChangeClass:    rawAttrs["change_class"],
		FromVersion:    rawAttrs["from_version"],
		ToVersion:      rawAttrs["to_version"],
		EffectsWidened: rawAttrs["effects_widened"] == "true",
	}

	// Evidence order per field: what msg already carries (embedded envelope,
	// decoded by the adapter) wins; then attributes; then the payload. A
	// non-empty existing field is authoritative and any conflicting downstream
	// evidence is logged, never applied.
	set := func(existing *string, attr, payloadVal string, field string) {
		if *existing != "" {
			if logger != nil && attr != "" && attr != *existing {
				logger.Printf("cascade hydration: message %s field %s already set to %q; attribute %q disagrees (existing wins)", msg.ID, field, *existing, attr)
			}
			if logger != nil && payloadVal != "" && payloadVal != *existing {
				logger.Printf("cascade hydration: message %s field %s already set to %q; payload %q disagrees (existing wins)", msg.ID, field, *existing, payloadVal)
			}
			return
		}
		if attr != "" {
			if payloadVal != "" && payloadVal != attr && logger != nil {
				logger.Printf("cascade hydration: message %s field %s: attribute %q and payload %q disagree (attribute wins)", msg.ID, field, attr, payloadVal)
			}
			*existing = attr
			return
		}
		*existing = payloadVal
	}

	set(&msg.RootPackage, fromAttrs.RootPackage, fromPayload.RootPackage, "root_package")
	set(&msg.RootChangeClass, fromAttrs.ChangeClass, fromPayload.ChangeClass, "change_class")
	set(&msg.FromVersion, fromAttrs.FromVersion, fromPayload.FromVersion, "from_version")
	set(&msg.ToVersion, fromAttrs.ToVersion, fromPayload.ToVersion, "to_version")
	set(&msg.FromInterfaceHash, "", fromPayload.FromInterfaceHash, "from_interface_hash")
	set(&msg.ToInterfaceHash, "", fromPayload.ToInterfaceHash, "to_interface_hash")
	set(&msg.FromContentHash, "", fromPayload.FromContentHash, "from_content_hash")
	set(&msg.ToContentHash, "", fromPayload.ToContentHash, "to_content_hash")

	if !msg.EffectsWidened && (fromAttrs.EffectsWidened || fromPayload.EffectsWidened) {
		msg.EffectsWidened = true
	}
	if len(msg.PrevEffectCeiling) == 0 && len(fromPayload.PrevEffectCeiling) > 0 {
		msg.PrevEffectCeiling = fromPayload.PrevEffectCeiling
	}
	if len(msg.NewEffectCeiling) == 0 && len(fromPayload.NewEffectCeiling) > 0 {
		msg.NewEffectCeiling = fromPayload.NewEffectCeiling
	}

	// Authority stamp: only the cascade topic's own assertion counts.
	if attrs.Source == pubsub.SourceCascade && msg.Source == "" {
		msg.Source = pubsub.SourceCascade
	}

	return msg
}

// isCascadeDirectiveKind reports whether a task Kind names a package bump that
// the pkg-update cascade pipeline acts on. The publisher stamps
// MessageType "upgrade-available" on cascade publishes (cmd/ailang/pkg_publish.go);
// interface-change notices ride the same repair pipeline.
func isCascadeDirectiveKind(kind string) bool {
	switch kind {
	case string(messaging.PkgMsgUpgradeAvailable), string(messaging.PkgMsgInterfaceChange):
		return true
	default:
		return false
	}
}

// cascadeDispatchAuthorised reports whether a cascade-shaped task carries the
// authority stamp AND the minimum envelope a cascade directive needs to be
// actionable: a root package to bump toward and a version to bump to.
func cascadeDispatchAuthorised(task *TaskRecord) bool {
	if task == nil {
		return false
	}
	return task.Source == pubsub.SourceCascade && task.RootPackage != "" && task.ToVersion != ""
}
