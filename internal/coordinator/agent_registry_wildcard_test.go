package coordinator

import (
	"sort"
	"testing"
)

// M-COORDINATOR-INBOX-WILDCARDS acceptance tests.
//
// Origin: an abi 2.1.0 republish triggered 10 dependent cascade tasks that all
// failed silently because no agent was registered for the pkg:sunholo/motoko_ext_*
// inboxes. Each task was created with an empty AgentID, dispatched to a Cloud Run
// Job, and died on arrival with "AILANG_AGENT_ID environment variable is required".
// Family patterns remove the per-package config tax that caused it.

func mustRegister(t *testing.T, r *AgentRegistry, id, inbox string) {
	t.Helper()
	if err := r.Register(&AgentConfig{ID: id, Inbox: inbox, Workspace: "/tmp/t"}); err != nil {
		t.Fatalf("register %s (%s): %v", id, inbox, err)
	}
}

// TestGetAgentForInbox_WildcardFallback covers regression fixture 1 and 3 from
// the design doc: existing exact registrations keep working, and an inbox with
// no match at all still returns nil rather than being swept up by a pattern.
func TestGetAgentForInbox_WildcardFallback(t *testing.T) {
	r := NewAgentRegistry()
	mustRegister(t, r, "pkg-sunholo-auth", "pkg:sunholo/auth")
	mustRegister(t, r, "pkg-sunholo-gcp-auth", "pkg:sunholo/gcp_auth")
	mustRegister(t, r, "pkg-motoko-ext-cascade", "pkg:sunholo/motoko_ext_*")

	t.Run("exact registrations still route to their own agent", func(t *testing.T) {
		for inbox, want := range map[string]string{
			"pkg:sunholo/auth":     "pkg-sunholo-auth",
			"pkg:sunholo/gcp_auth": "pkg-sunholo-gcp-auth",
		} {
			got := r.GetAgentForInbox(inbox)
			if got == nil || got.ID != want {
				t.Errorf("%s → %v, want %s", inbox, got, want)
			}
		}
	})

	t.Run("family member with no explicit entry hits the pattern", func(t *testing.T) {
		// This is the exact case that produced the 10 silent failures.
		for _, inbox := range []string{
			"pkg:sunholo/motoko_ext_abi",
			"pkg:sunholo/motoko_ext_compose",
			"pkg:sunholo/motoko_ext_a_package_that_does_not_exist_yet",
		} {
			got := r.GetAgentForInbox(inbox)
			if got == nil {
				t.Errorf("%s → nil; a family member must reach the pattern agent", inbox)
			} else if got.ID != "pkg-motoko-ext-cascade" {
				t.Errorf("%s → %s, want pkg-motoko-ext-cascade", inbox, got.ID)
			}
		}
	})

	t.Run("unmatched inbox still returns nil with no catch-all", func(t *testing.T) {
		if got := r.GetAgentForInbox("pkg:nonexistent/xyz"); got != nil {
			t.Errorf("pkg:nonexistent/xyz → %s, want nil — a non-matching inbox must not be swept up", got.ID)
		}
	})

	t.Run("pattern does not match its own prefix-minus-family", func(t *testing.T) {
		if got := r.GetAgentForInbox("pkg:sunholo/other"); got != nil {
			t.Errorf("pkg:sunholo/other → %s, want nil", got.ID)
		}
	})
}

// TestGetAgentForInbox_LongestPrefixWins covers regression fixture 2: an explicit
// agent overrides the family glob, and a more specific glob beats a broader one.
func TestGetAgentForInbox_LongestPrefixWins(t *testing.T) {
	r := NewAgentRegistry()
	// Registered deliberately broadest-first, so a correct result cannot come
	// from insertion order alone.
	mustRegister(t, r, "catch-all", "pkg:*")
	mustRegister(t, r, "sunholo-family", "pkg:sunholo/*")
	mustRegister(t, r, "motoko-family", "pkg:sunholo/motoko_ext_*")
	mustRegister(t, r, "abi-explicit", "pkg:sunholo/motoko_ext_abi")

	for inbox, want := range map[string]string{
		"pkg:sunholo/motoko_ext_abi":     "abi-explicit",   // exact beats every pattern
		"pkg:sunholo/motoko_ext_compose": "motoko-family",  // most specific pattern
		"pkg:sunholo/auth":               "sunholo-family", // vendor pattern
		"pkg:othervendor/thing":          "catch-all",      // broadest pattern
	} {
		got := r.GetAgentForInbox(inbox)
		if got == nil || got.ID != want {
			t.Errorf("%s → %v, want %s", inbox, got, want)
		}
	}
}

// TestGetAgentForInbox_CatchAll covers regression fixture 4.
func TestGetAgentForInbox_CatchAll(t *testing.T) {
	r := NewAgentRegistry()
	mustRegister(t, r, "catch-all", "pkg:*")

	for _, inbox := range []string{"pkg:sunholo/anything", "pkg:newvendor/newpkg", "pkg:x/y"} {
		if got := r.GetAgentForInbox(inbox); got == nil || got.ID != "catch-all" {
			t.Errorf("%s → %v, want catch-all", inbox, got)
		}
	}

	t.Run("catch-all does not capture non-pkg inboxes", func(t *testing.T) {
		// A `pkg:*` catch-all must not swallow agent inboxes like sprint-executor,
		// or every unrouted human message would be dispatched to a package bumper.
		for _, inbox := range []string{"sprint-executor", "user", "public-feedback"} {
			if got := r.GetAgentForInbox(inbox); got != nil {
				t.Errorf("%s → %s, want nil; pkg:* must not capture non-package inboxes", inbox, got.ID)
			}
		}
	})
}

// TestWildcardRegistryBookkeeping asserts the pattern table stays consistent
// under the operations that mutate it — a stale entry would route mail to a
// deregistered agent.
func TestWildcardRegistryBookkeeping(t *testing.T) {
	r := NewAgentRegistry()
	mustRegister(t, r, "motoko-family", "pkg:sunholo/motoko_ext_*")
	mustRegister(t, r, "exact", "pkg:sunholo/auth")

	t.Run("HasInbox agrees with GetAgentForInbox", func(t *testing.T) {
		// Disagreement would let a caller refuse mail the dispatcher accepts.
		for _, inbox := range []string{"pkg:sunholo/motoko_ext_abi", "pkg:sunholo/auth", "pkg:nope/nope"} {
			has := r.HasInbox(inbox)
			got := r.GetAgentForInbox(inbox) != nil
			if has != got {
				t.Errorf("%s: HasInbox=%v but GetAgentForInbox!=nil is %v", inbox, has, got)
			}
		}
	})

	t.Run("ListInboxes includes patterns", func(t *testing.T) {
		got := r.ListInboxes()
		sort.Strings(got)
		want := []string{"pkg:sunholo/auth", "pkg:sunholo/motoko_ext_*"}
		if len(got) != len(want) {
			t.Fatalf("ListInboxes() = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("ListInboxes() = %v, want %v", got, want)
				break
			}
		}
	})

	t.Run("duplicate pattern is refused", func(t *testing.T) {
		err := r.Register(&AgentConfig{ID: "dupe", Inbox: "pkg:sunholo/motoko_ext_*", Workspace: "/tmp/t"})
		if err == nil {
			t.Error("registering a duplicate pattern must fail, not shadow the original")
		}
	})

	t.Run("Unregister removes the pattern", func(t *testing.T) {
		if err := r.Unregister("motoko-family"); err != nil {
			t.Fatalf("unregister: %v", err)
		}
		if got := r.GetAgentForInbox("pkg:sunholo/motoko_ext_abi"); got != nil {
			t.Errorf("after unregister → %s, want nil; stale pattern still routing", got.ID)
		}
		if r.GetAgentForInbox("pkg:sunholo/auth") == nil {
			t.Error("unregistering a pattern must not disturb exact registrations")
		}
	})

	t.Run("Clear drops patterns", func(t *testing.T) {
		r2 := NewAgentRegistry()
		mustRegister(t, r2, "fam", "pkg:sunholo/motoko_ext_*")
		r2.Clear()
		if got := r2.GetAgentForInbox("pkg:sunholo/motoko_ext_abi"); got != nil {
			t.Errorf("after Clear → %s, want nil", got.ID)
		}
	})
}
