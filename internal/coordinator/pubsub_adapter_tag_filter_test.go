package coordinator

import (
	"io"
	"log"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// We avoid implementing the full messaging.MessageStore interface (dozens of
// methods) by using nil msgStore in most tests — HandleNotification guards the
// fetch with `if a.msgStore != nil`. For the no-fetch-on-mismatch assertion
// we use panicOnGetStore which fails the test if GetInboxMessage is reached.

type panicOnGetStore struct {
	messaging.MessageStore // embed nil interface; all methods will panic if called
	t                      *testing.T
}

func (p *panicOnGetStore) GetInboxMessage(_ string) (*messaging.InboxMessage, error) {
	p.t.Fatal("GetInboxMessage was called when tag filter should have rejected the message before the fetch")
	return nil, nil
}

func newSilentLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

// validNotification produces the minimal Pub/Sub data payload (just message_id).
func validNotification(id string) []byte {
	return []byte(`{"message_id":"` + id + `"}`)
}

func TestPubSubAdapter_NoRequiresAttribute_BackwardsCompat(t *testing.T) {
	// Adapter with no advertised tags + message with no `requires` attribute:
	// the pre-tag world's behavior. Must process normally.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())

	attrs := map[string]string{"inbox": "eval-rig", "from_agent": "user"}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Fatalf("HandleNotification with no `requires` returned error: %v", err)
	}
	msgs, _ := a.ListUnread()
	if len(msgs) != 1 {
		t.Errorf("buffered messages = %d, want 1", len(msgs))
	}
}

func TestPubSubAdapter_RequiresMatched_ProcessesAndAcks(t *testing.T) {
	// Adapter advertises ollama:gemma4-26b-ailang. Message requires same.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())
	a.SetWorkerTags("studio.eval-rig", []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Fatalf("matched message should not error: %v", err)
	}
	msgs, _ := a.ListUnread()
	if len(msgs) != 1 {
		t.Errorf("buffered = %d, want 1", len(msgs))
	}
}

func TestPubSubAdapter_RequiresMismatched_NacksWithoutFetch(t *testing.T) {
	// Adapter without ollama tags. Message requires ollama. Must:
	// 1. Return non-nil error (Pub/Sub treats this as NACK -> redelivery)
	// 2. NOT call GetInboxMessage (efficiency — panicOnGetStore enforces this)
	// 3. NOT buffer the message
	store := &panicOnGetStore{t: t}
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", store, newSilentLogger())
	a.SetWorkerTags("laptop.dev", []string{"code", "docs"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang",
	}
	err := a.HandleNotification(validNotification("m1"), attrs)
	if err == nil {
		t.Fatal("mismatch should return non-nil error (nack)")
	}
	if !strings.Contains(err.Error(), "tag filter") {
		t.Errorf("error %q should mention 'tag filter' for observability", err.Error())
	}
	msgs, _ := a.ListUnread()
	if len(msgs) != 0 {
		t.Errorf("buffered = %d, want 0 (mismatch must not buffer)", len(msgs))
	}
}

func TestPubSubAdapter_RequiresMultipleTags_AllMustMatch(t *testing.T) {
	// Adapter has ollama only. Message requires ollama AND gpu. Must reject.
	store := &panicOnGetStore{t: t}
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", store, newSilentLogger())
	a.SetWorkerTags("studio.eval-rig", []string{"ollama:gemma4-26b-ailang"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang,gpu:m4-max",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err == nil {
		t.Fatal("should reject when not all required tags satisfied")
	}
}

func TestPubSubAdapter_RequiresWithGlob_MatchesByFamily(t *testing.T) {
	// Adapter advertises ollama:* (family glob). Message requires specific.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())
	a.SetWorkerTags("studio.eval-rig", []string{"ollama:*"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Errorf("glob family adapter should accept specific required: %v", err)
	}
}

func TestPubSubAdapter_TwoWorkers_OnlyOneClaims(t *testing.T) {
	// The headline scenario: two adapters receive the same Pub/Sub
	// notification. Only the matching one processes; the other nacks.
	studio := NewPubSubInboxAdapter(nil, "sub-studio", "eval-rig", nil, newSilentLogger())
	studio.SetWorkerTags("studio.eval-rig", []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"})

	laptop := NewPubSubInboxAdapter(nil, "sub-laptop", "eval-rig", nil, newSilentLogger())
	laptop.SetWorkerTags("laptop.dev", []string{"code", "docs", "research"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang",
	}

	if err := studio.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Errorf("studio (matching) should have processed; got error: %v", err)
	}
	if err := laptop.HandleNotification(validNotification("m1"), attrs); err == nil {
		t.Error("laptop (non-matching) should have rejected; got nil error")
	}

	if msgs, _ := studio.ListUnread(); len(msgs) != 1 {
		t.Errorf("studio buffered = %d, want 1", len(msgs))
	}
	if msgs, _ := laptop.ListUnread(); len(msgs) != 0 {
		t.Errorf("laptop buffered = %d, want 0", len(msgs))
	}
}

func TestPubSubAdapter_EmptyAdvertisedTags_StillProcessesUntaggedMessages(t *testing.T) {
	// Adapter never had SetWorkerTags called. Message has NO requires attr.
	// Must still process — protects legacy single-host setups.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())

	attrs := map[string]string{"inbox": "eval-rig"}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Errorf("unconfigured adapter + untagged message should process: %v", err)
	}
}

func TestPubSubAdapter_EmptyAdvertisedTags_RejectsRequiresMessage(t *testing.T) {
	// Adapter has no advertised tags. Message HAS requires. Must reject —
	// otherwise unconfigured workers would steal tag-routed messages.
	store := &panicOnGetStore{t: t}
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", store, newSilentLogger())

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "ollama:gemma4-26b-ailang",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err == nil {
		t.Error("untagged adapter must reject tag-routed message; got nil error")
	}
}

func TestPubSubAdapter_RequiresWhitespace_Trimmed(t *testing.T) {
	// `requires: " ollama:gemma4-26b-ailang , gpu:m4-max "` (spaces around commas)
	// Should still parse cleanly.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())
	a.SetWorkerTags("studio.eval-rig", []string{"ollama:gemma4-26b-ailang", "gpu:m4-max"})

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": " ollama:gemma4-26b-ailang , gpu:m4-max ",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Errorf("whitespace in `requires` should be trimmed: %v", err)
	}
}

func TestPubSubAdapter_RequiresEmptyString_NoOp(t *testing.T) {
	// `requires: ""` (empty) is equivalent to no requires — process normally.
	a := NewPubSubInboxAdapter(nil, "sub-x", "eval-rig", nil, newSilentLogger())

	attrs := map[string]string{
		"inbox":    "eval-rig",
		"requires": "",
	}
	if err := a.HandleNotification(validNotification("m1"), attrs); err != nil {
		t.Errorf("empty requires should be treated as no constraint: %v", err)
	}
	msgs, _ := a.ListUnread()
	if len(msgs) != 1 {
		t.Errorf("buffered = %d, want 1 (empty requires = no constraint)", len(msgs))
	}
}
