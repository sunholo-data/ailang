package coordinator

import (
	"testing"
	"time"
)

// M-COORDINATOR-EXECUTION-TRUST M8 (design doc V38/V39).
//
// Cloud Run Jobs were chosen for this architecture BECAUSE they allow long
// runs — up to 24h. We were using 15 minutes of that: 14 of 35 prod agents
// carried `timeout: "15m"`, and a real task filed on 2026-09-02 died on
// "pi exceeded hard timeout (15m0s)" doing ordinary package work.
//
// Raising the wall-clock ceiling is safe because it is NOT the liveness guard.
// `idle_timeout` ("kill if no events for this long") is what detects a hung
// agent, and it stays short. The wall-clock timeout only caps cost and runaway.

// THE ORDERING TRAP, and the reason this file exists.
//
// Cloud Run's own taskTimeout must stay ABOVE the agent's wall-clock timeout. If
// it does not, the platform kills the container before the executor's timeout
// fires — so no completion is ever published, and the task's outcome is lost
// rather than reported. That is exactly the failure class M7 removed
// (work happened, nobody was told), reintroduced through configuration.
//
// The two values live in different repos: the agent timeout in the coordinator's
// config.yaml, the platform timeout in the multivac terraform's `agent_timeout`.
// Nothing connects them, so this asserts the relationship from the side that can.

func TestAgentTimeoutStaysBelowThePlatformCeiling(t *testing.T) {
	if MaxAgentTaskTimeout >= PlatformTaskTimeout {
		t.Fatalf("agent ceiling %v must stay strictly below the platform ceiling %v — "+
			"otherwise Cloud Run kills the container before the executor can report",
			MaxAgentTaskTimeout, PlatformTaskTimeout)
	}
	// A meaningful gap, not a token one: the executor needs room to notice its
	// own timeout, publish a completion and exit.
	if gap := PlatformTaskTimeout - MaxAgentTaskTimeout; gap < time.Hour {
		t.Errorf("gap between agent (%v) and platform (%v) ceilings is %v — "+
			"too tight to guarantee a clean report", MaxAgentTaskTimeout, PlatformTaskTimeout, gap)
	}
}

func TestDefaultTaskTimeoutUsesTheArchitectureItWasChosenFor(t *testing.T) {
	// The old default was 30m and the common agent value 15m, against a platform
	// that allows 24h. A default under an hour makes ordinary package work fail
	// for no reason other than the number.
	if DefaultTaskTimeout < time.Hour {
		t.Errorf("default task timeout %v is under an hour; Cloud Run Jobs allow 24h and "+
			"were chosen for that", DefaultTaskTimeout)
	}
	if DefaultTaskTimeout > MaxAgentTaskTimeout {
		t.Errorf("default %v exceeds the agent ceiling %v", DefaultTaskTimeout, MaxAgentTaskTimeout)
	}
}

// The liveness guard must stay SHORT. It is what makes a long wall-clock safe:
// a hung agent is killed in minutes regardless of how generous the ceiling is.
func TestIdleTimeoutRemainsTheLivenessGuard(t *testing.T) {
	def := (&AgentConfig{}).GetEffectiveIdleTimeout()
	if def > 15*time.Minute {
		t.Errorf("default idle timeout %v is too long to serve as the liveness guard; "+
			"a long wall-clock ceiling is only safe while this stays short", def)
	}
	if def <= 0 {
		t.Fatal("there must be a default idle timeout — it is the only thing that kills a hung agent")
	}
}
