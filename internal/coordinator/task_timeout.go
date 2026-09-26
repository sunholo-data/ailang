package coordinator

import "time"

// M-COORDINATOR-EXECUTION-TRUST M8 — the task ceilings, and the relationship
// between them.
//
// Cloud Run Jobs were chosen for this architecture BECAUSE they permit long
// runs: the platform maximum is 24 hours. We were using fifteen minutes of it.
// 14 of the 35 prod agents carried `timeout: "15m"`, and on 2026-09-02 a real
// package task died on `pi exceeded hard timeout (15m0s)` doing ordinary work —
// the ceiling, not the task, was the problem.
//
// Raising the wall-clock ceiling is safe because it is NOT the liveness guard.
// `idle_timeout` — "kill if no events for this long" — is what detects a hung
// agent, and it stays at minutes. The wall-clock timeout only caps cost and
// runaway, so a generous value costs nothing while an agent is making progress.
//
// Ratified by Mark 2026-09-02 (attended): 2h agents, 4h platform.

const (
	// DefaultTaskTimeout applies when an agent declares no timeout of its own.
	// Was 30m, which quietly capped what the plane could be asked to do.
	DefaultTaskTimeout = 2 * time.Hour

	// MaxAgentTaskTimeout is the highest wall-clock an agent may declare.
	MaxAgentTaskTimeout = 2 * time.Hour

	// PlatformTaskTimeout mirrors the multivac terraform's `agent_timeout`, the
	// Cloud Run Job taskTimeout.
	//
	// It MUST stay strictly above MaxAgentTaskTimeout, with room to spare. If the
	// platform ceiling is the lower of the two, Cloud Run kills the container
	// before the executor's own timeout fires — so no completion is ever
	// published and the outcome is LOST rather than reported. That is precisely
	// the failure class M7 removed (work happened, nobody was told), reachable
	// again through configuration alone.
	//
	// The two values live in different repos and nothing connects them, so
	// TestAgentTimeoutStaysBelowThePlatformCeiling asserts the relationship from
	// the side that can. Change this only together with terraform.
	PlatformTaskTimeout = 4 * time.Hour
)
