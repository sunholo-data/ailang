package coordinator

import (
	"io"
	"log"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// M-CASCADE-ENVELOPE-HYDRATION tests.
//
// 173 prod `[bug] cascade routing: EMPTY dispatch` reports in 5000 store
// messages were each an agent paid to conclude "nothing to repair" because a
// cascade envelope was dropped on one of three coordinator-internal routes.
// These tests pin the hydration helper's evidence order and the
// never-guess-Source rule at the unit level; the adapter, sweep and dispatch
// gate carry their own route-level tests.

// cascadePayload builds a realistic durable payload — exactly what
// messaging.UpgradeAvailableMessage stores — for sunholo/auth 1.2.0 → 1.3.0.
func cascadePayload(t *testing.T) string {
	t.Helper()
	old := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "1.2.0",
		InterfaceHash: "iface-old", ContentHash: "content-old",
	}
	next := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "1.3.0",
		InterfaceHash: "iface-new", ContentHash: "content-new",
	}
	msg, err := messaging.UpgradeAvailableMessage(old, next, []string{"pkg:sunholo/docparse"}, "C")
	if err != nil || msg == nil {
		t.Fatalf("UpgradeAvailableMessage: %v, %v", msg, err)
	}
	return msg.Payload
}

func silentLogger() *log.Logger { return log.New(io.Discard, "", 0) }

func TestHydrateCascadeFields_PayloadOnly_PopulatesEnvelopeWithoutStampingSource(t *testing.T) {
	msg := &Message{ID: "msg-1"}
	got := hydrateCascadeFields(msg, pubsub.MessageAttributes{}, nil, cascadePayload(t), silentLogger())

	if got.RootPackage != "sunholo/auth" {
		t.Errorf("RootPackage = %q, want sunholo/auth (the payload always carried this — the routes just never read it)", got.RootPackage)
	}
	if got.RootChangeClass != "C" {
		t.Errorf("RootChangeClass = %q, want C", got.RootChangeClass)
	}
	if got.FromVersion != "1.2.0" || got.ToVersion != "1.3.0" {
		t.Errorf("versions = %q → %q, want 1.2.0 → 1.3.0", got.FromVersion, got.ToVersion)
	}
	if got.FromInterfaceHash != "iface-old" || got.ToInterfaceHash != "iface-new" {
		t.Errorf("interface hashes = %q → %q", got.FromInterfaceHash, got.ToInterfaceHash)
	}
	if got.FromContentHash != "content-old" || got.ToContentHash != "content-new" {
		t.Errorf("content hashes = %q → %q", got.FromContentHash, got.ToContentHash)
	}
	// The never-guess rule: payload shape alone is NOT cascade authority. A
	// wrong Source=cascade stamp makes the pkg-update template's misroute
	// guard lie; an empty one merely makes the dispatch gate refuse loudly.
	if got.Source != "" {
		t.Errorf("Source = %q, want empty — payload shape must never stamp cascade authority", got.Source)
	}
}

func TestHydrateCascadeFields_AttributesOnly_RequiresCascadeAssertion(t *testing.T) {
	attrs := pubsub.MessageAttributes{Source: pubsub.SourceCascade}
	raw := map[string]string{
		"root_package":    "sunholo/auth@1.3.0",
		"change_class":    "B",
		"from_version":    "1.2.0",
		"to_version":      "1.3.0",
		"effects_widened": "true",
	}
	got := hydrateCascadeFields(&Message{ID: "msg-2"}, attrs, raw, "", silentLogger())

	if got.Source != pubsub.SourceCascade {
		t.Errorf("Source = %q, want cascade — the topic asserted authority", got.Source)
	}
	if got.RootPackage != "sunholo/auth@1.3.0" || got.RootChangeClass != "B" || got.FromVersion != "1.2.0" || got.ToVersion != "1.3.0" {
		t.Errorf("envelope from attributes incomplete: %+v", got)
	}
	if !got.EffectsWidened {
		t.Error("EffectsWidened attribute not honored")
	}
}

func TestHydrateCascadeFields_NoEvidence_TouchesNothing(t *testing.T) {
	msg := &Message{ID: "msg-3", Source: "messages"}
	got := hydrateCascadeFields(msg, pubsub.MessageAttributes{Source: "messages"}, nil, "just a plain message body", silentLogger())

	if got.RootPackage != "" || got.ToVersion != "" || got.FromInterfaceHash != "" {
		t.Errorf("plain message gained cascade fields it has no evidence for: %+v", got)
	}
	if got.Source != "messages" {
		t.Errorf("Source changed: %q", got.Source)
	}
}

func TestHydrateCascadeFields_EmbeddedEnvelopeWinsOverDownstreamEvidence(t *testing.T) {
	msg := &Message{ID: "msg-4", RootPackage: "sunholo/auth@1.3.0", ToVersion: "1.3.0"}
	raw := map[string]string{"root_package": "sunholo/auth@9.9.9"}
	got := hydrateCascadeFields(msg, pubsub.MessageAttributes{Source: pubsub.SourceCascade}, raw, cascadePayload(t), silentLogger())

	if got.RootPackage != "sunholo/auth@1.3.0" {
		t.Errorf("RootPackage = %q — the embedded envelope must win over attributes and payload", got.RootPackage)
	}
	if got.ToVersion != "1.3.0" {
		t.Errorf("ToVersion = %q", got.ToVersion)
	}
}

func TestHydrateCascadeFields_NilMessageIsSafe(t *testing.T) {
	if got := hydrateCascadeFields(nil, pubsub.MessageAttributes{}, nil, "", silentLogger()); got != nil {
		t.Errorf("nil message must stay nil, got %+v", got)
	}
}

func TestIsCascadeDirectiveKind(t *testing.T) {
	for _, kind := range []string{"upgrade-available", "interface-change-notice"} {
		if !isCascadeDirectiveKind(kind) {
			t.Errorf("kind %q must be recognised as a cascade directive", kind)
		}
	}
	for _, kind := range []string{"", "directive", "feedback", "completion", "question"} {
		if isCascadeDirectiveKind(kind) {
			t.Errorf("kind %q must NOT be recognised as a cascade directive", kind)
		}
	}
}

func TestCascadeDispatchAuthorised(t *testing.T) {
	full := &TaskRecord{Source: pubsub.SourceCascade, RootPackage: "sunholo/auth", ToVersion: "1.3.0", Kind: "upgrade-available"}
	if !cascadeDispatchAuthorised(full) {
		t.Error("a fully populated cascade task must be authorised")
	}
	for name, mutate := range map[string]func(*TaskRecord){
		"no source":       func(t *TaskRecord) { t.Source = "" },
		"public source":   func(t *TaskRecord) { t.Source = "messages" },
		"no root package": func(t *TaskRecord) { t.RootPackage = "" },
		"no to version":   func(t *TaskRecord) { t.ToVersion = "" },
	} {
		broken := *full
		mutate(&broken)
		if cascadeDispatchAuthorised(&broken) {
			t.Errorf("%s: must not be authorised", name)
		}
	}
	if cascadeDispatchAuthorised(nil) {
		t.Error("nil task must not be authorised")
	}
}

// The misroute guard fired on empty rendering, so the payload that produced
// the report must hydrate with EVERY field the template interpolates.
func TestHydrateCascadeFields_TemplateVariablesAllPopulated(t *testing.T) {
	got := hydrateCascadeFields(&Message{ID: "msg-5"}, pubsub.MessageAttributes{Source: pubsub.SourceCascade}, nil, cascadePayload(t), silentLogger())

	for name, v := range map[string]string{
		"RootPackage":        got.RootPackage,
		"RootChangeClass":    got.RootChangeClass,
		"FromVersion":        got.FromVersion,
		"ToVersion":          got.ToVersion,
		"FromInterfaceHash":  got.FromInterfaceHash,
		"ToInterfaceHash":    got.ToInterfaceHash,
		"Source":             got.Source,
	} {
		if strings.TrimSpace(v) == "" {
			t.Errorf("%s rendered empty — the exact failure that produced 173 bug reports", name)
		}
	}
}
