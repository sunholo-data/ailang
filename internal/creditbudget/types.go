// Package creditbudget implements the transactional authority for confirmed API credits.
// It has no provider credentials and never infers renewal from a calendar rollover.
package creditbudget

import (
	"context"
	"errors"
	"time"
)

type MicroUSD int64

const USD MicroUSD = 1_000_000
const MaximumGrant MicroUSD = 200 * USD
const ExpiryBuffer = 5 * time.Minute
const CanaryCeiling MicroUSD = 5 * USD

var ErrBlocked = errors.New("credit budget blocked")
var ErrNotFound = errors.New("credit record not found")
var ErrConflict = errors.New("credit record conflicts with existing state")

// Store must atomically commit all writes or none. Callbacks may be retried;
// they must not perform external side effects. Reads precede all writes.
type Store interface {
	Run(context.Context, string, func(Transaction) error) error
}
type Transaction interface {
	Get(collection, id string, into any) (bool, error)
	Put(collection, id string, value any) error
}

type Authority interface {
	Configure(context.Context, Policy) error
	SetEnabled(context.Context, string, bool, string, string) error
	CompleteCanary(context.Context, string, string, string) error
	Confirm(context.Context, Confirmation) (Status, error)
	Status(context.Context, string) (Status, error)
	AdmitTask(context.Context, Task) error
	ReleaseTask(context.Context, string, string) error
	ReleaseAttempt(context.Context, string, string, string, string) error
	Reserve(context.Context, Reservation) (Request, error)
	MarkForwarded(context.Context, string, string) error
	MarkUnresolved(context.Context, string, string) error
	Settle(context.Context, string, string, MicroUSD, string) error
	ReleaseUnsent(context.Context, string, string) error
}

type Policy struct {
	AccountID        string
	Organization     string
	Workspace        string
	Enabled          bool
	OperatingCeiling MicroUSD
	DailyCeiling     MicroUSD
	TaskCeiling      MicroUSD
	MaxTasks         int
}
type Confirmation struct {
	AccountID       string
	GrantID         string
	Amount          MicroUSD
	Available       MicroUSD
	StartsAt        time.Time
	ExpiresAt       time.Time
	Operator        string
	Evidence        string
	ExpectedGrantID string
}
type Grant struct {
	Confirmation
	ConfirmedAt time.Time
}
type Task struct {
	AccountID   string
	ID          string
	AttemptID   string
	JobIdentity string
	Models      []string
	Ceiling     MicroUSD
	LeaseUntil  time.Time
	Settled     MicroUSD
	Reserved    MicroUSD
	Released    bool
}
type Reservation struct {
	AccountID       string
	TaskID          string
	AttemptID       string
	JobIdentity     string
	Model           string
	RequestID       string
	Amount          MicroUSD
	PricingRevision string
	Deadline        time.Time
}
type Request struct {
	Reservation
	GrantID    string
	Day        string
	State      string
	Actual     MicroUSD
	UpstreamID string
	CreatedAt  time.Time
	Canary     bool
}
type Account struct {
	Policy
	Grant    Grant
	Settled  MicroUSD
	Reserved MicroUSD
	// Forwarding is exposure with durable send intent and no complete result.
	// It includes requests interrupted by a process crash before MarkUnresolved.
	Forwarding             MicroUSD
	Unresolved             MicroUSD
	ReconciliationRequired bool
	ReconciliationReason   string
	Leases                 map[string]time.Time
	CanaryComplete         bool
	CanarySettled          MicroUSD
	CanaryReserved         MicroUSD
}
type Status struct {
	Account
	Available       MicroUSD
	CanaryAvailable MicroUSD
	Eligible        bool
	Blockers        []string
	Day             Day
	ActiveTasks     int
}
type Day struct{ Settled, Reserved MicroUSD }
type Audit struct {
	Operator, Evidence, Action string
	At                         time.Time
	Enabled                    bool
}

type Engine struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Engine {
	if now == nil {
		now = time.Now
	}
	return &Engine{store: store, now: now}
}
