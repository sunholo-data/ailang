package accessapproval

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Service always uses a caller-selected local coordinator SQLite database.
// Its process identity is trusted; HTTP owner/worker authorization belongs to
// the adapter. Workers must not receive this binary/store as a capability.
type Service struct {
	store    *coordinator.SQLiteStore
	now      func() int64
	runDummy func(*Request) map[string]any
}

func Open(dir, source string) (*Service, error) {
	if !filepath.IsAbs(dir) || source != "daneel-local" {
		return nil, fail("unsupported_source", "an absolute --state-dir and --source daneel-local are required; cloud durable approvals are unsupported")
	}
	selection, err := config.StoragePlane()
	if err != nil || selection.Coordinator.Mode != config.StoreLocal {
		return nil, fail("unsupported_source", "cloud durable approvals are unsupported; explicitly configure local coordinator storage")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, fail("unsafe_state_dir", "state directory must be a private directory (0700)")
	}
	path := filepath.Join(dir, "coordinator.db")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err == nil {
		_ = f.Close()
	} else if !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	info, err = os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, fail("unsafe_state_dir", "coordinator.db must be a private regular file (0600)")
	}
	store, err := coordinator.NewAccessApprovalSQLiteStore(path)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, now: func() int64 { return time.Now().Unix() }, runDummy: dummyOperation}, nil
}
func (s *Service) Close() error { return s.store.Close() }
func hash(v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func opaqueID() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) load(ctx context.Context, id string) (*Request, string, error) {
	if err := validateText("id", id, 128); err != nil {
		return nil, "", err
	}
	row, err := s.store.GetApprovalRequest(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", fail("not_found", "approval does not exist")
	}
	if err != nil {
		return nil, "", err
	}
	if row.Type != coordinator.DurableAccessApprovalType {
		return nil, "", fail("not_found", "approval does not exist")
	}
	var r Request
	if err := json.Unmarshal([]byte(row.ContextJSON), &r); err != nil {
		return nil, "", fail("invalid_record", "durable approval is unreadable")
	}
	return &r, row.ContextJSON, nil
}

// mutate retries only record CAS, never an external operation. The callback
// must be pure except for allocating an opaque claim id.
func (s *Service) mutate(ctx context.Context, id string, fn func(*Request) error) (*Request, error) {
	for tries := 0; tries < 32; tries++ {
		r, before, err := s.load(ctx, id)
		if err != nil {
			return nil, err
		}
		s.expire(r)
		// Persist reconciliation even if a requested transition is rejected.
		normalized, _ := json.Marshal(r)
		opErr := fn(r)
		if opErr != nil {
			if string(normalized) == before {
				return nil, opErr
			}
			_ = json.Unmarshal(normalized, r)
		}
		after, _ := json.Marshal(r)
		if string(after) == before {
			return r, opErr
		}
		r.Revision++
		after, _ = json.Marshal(r)
		changed, err := s.store.CompareAndSwapAccessApproval(ctx, id, before, string(after), r.Decision)
		if err != nil {
			return nil, err
		}
		if changed {
			return r, opErr
		}
	}
	return nil, conflict("approval changed concurrently; retry the same command")
}
func (s *Service) expire(r *Request) {
	now := s.now()
	if r.Decision == "pending" && now >= r.ExpiresAt {
		r.Decision = "expired"
	}
	if r.Decision == "approved" && r.Execution.State == "unclaimed" && now >= r.ClaimUntil {
		r.Decision = "expired"
	}
	if (r.Execution.State == "claimed" || r.Execution.State == "started") && now >= r.Execution.ClaimUntil {
		if r.Execution.State == "started" {
			r.Execution.State = "unknown"
			r.Execution.Error = "execution lease expired without a receipt; operator reconciliation required"
		} else {
			r.Execution.State = "failed"
			r.Execution.Error = "claim expired before execution; create a fresh request"
		}
		r.Execution.CompletedAt = now
	}
}

// Handle accepts one versioned command. Receipt is read-only: workers cannot
// submit arbitrary success records to this authority.
func (s *Service) Handle(ctx context.Context, command string, in Input) (any, error) {
	switch command {
	case "request":
		return s.request(ctx, in)
	case "get", "receipt":
		return s.mutate(ctx, in.ID, func(*Request) error { return nil })
	case "list", "reconcile":
		return s.list(ctx, in, command == "reconcile")
	case "decide":
		return s.decide(ctx, in)
	case "claim":
		return s.claim(ctx, in)
	case "execute":
		return s.execute(ctx, in)
	case "cancel":
		return s.cancel(ctx, in)
	default:
		return nil, fail("invalid_command", "expected request, get, list, decide, claim, execute, receipt, cancel, or reconcile")
	}
}

func (s *Service) request(ctx context.Context, in Input) (*Request, error) {
	for name, value := range map[string]string{"owner": in.Owner, "worker": in.Worker, "task": in.Task, "attempt": in.Attempt} {
		if err := validateText(name, value, 256); err != nil {
			return nil, err
		}
	}
	if err := validateText("purpose", in.Purpose, 1024); err != nil {
		return nil, err
	}
	if in.Operation != "dummy" || in.ResourceAlias != "canary" || in.PolicyVersion != "1" || in.Version < 1 {
		return nil, fail("unsupported_operation", "only dummy/canary policy_version 1 with a positive request version is configured; secret resolver is unconfigured")
	}
	if in.Arguments == nil {
		in.Arguments = map[string]any{}
	}
	for key, value := range in.Arguments {
		message, ok := value.(string)
		if key != "message" || !ok || len(message) > 1024 {
			return nil, fail("invalid_input", "dummy arguments permit only an optional message string up to 1024 bytes")
		}
	}
	// First resolve an existing identity so explicit-expiry retries remain
	// idempotent after the deadline. New requests still enforce the TTL bound.
	id := "access-" + hash([]any{"daneel-local", in.Owner, in.Worker, in.Task, in.Attempt, in.Operation, in.Version})
	if existing, _, err := s.load(ctx, id); err == nil {
		if err := matchRequest(existing, in); err != nil {
			return nil, err
		}
		return s.mutate(ctx, id, func(*Request) error { return nil })
	} else {
		var e *Error
		if !errors.As(err, &e) || e.Code != "not_found" {
			return nil, err
		}
	}
	now := s.now()
	expiry := in.ExpiresAt
	if expiry == 0 {
		expiry = now + 7*24*3600
	}
	if expiry <= now || expiry > now+7*24*3600 {
		return nil, fail("invalid_input", "expires_at must be within the next seven days")
	}
	// Tuple encoding prevents delimiter collisions. All authorization-relevant
	// content is subsequently compared, so idempotent replay cannot alter scope.
	r := &Request{ID: id, Source: "daneel-local", Owner: in.Owner, Worker: in.Worker, Task: in.Task, Attempt: in.Attempt, Operation: in.Operation, ResourceAlias: in.ResourceAlias, Purpose: in.Purpose, Arguments: in.Arguments, ArgumentsDigest: hash(in.Arguments), PolicyVersion: in.PolicyVersion, Version: in.Version, Revision: 1, CreatedAt: now, ExpiresAt: expiry, Decision: "pending", Execution: Execution{State: "unclaimed"}}
	raw, _ := json.Marshal(r)
	_, err := s.store.CreateAccessApprovalIfAbsent(ctx, &coordinator.ApprovalRequestRecord{ID: id, TaskID: id, Type: coordinator.DurableAccessApprovalType, Description: "Durable local access approval", ContextJSON: string(raw), Status: "access_pending", CreatedAt: time.Unix(now, 0).UTC()}, in.Owner, in.Worker, now)
	if err != nil {
		return nil, err
	}
	existing, _, err := s.load(ctx, id)
	if err != nil {
		var e *Error
		if errors.As(err, &e) && e.Code == "not_found" {
			return nil, fail("capacity", "worker already has 100 pending requests")
		}
		return nil, err
	}
	if err := matchRequest(existing, in); err != nil {
		return nil, err
	}
	return s.mutate(ctx, id, func(*Request) error { return nil })
}

func matchRequest(existing *Request, in Input) error {
	if existing.ArgumentsDigest != hash(in.Arguments) || existing.Purpose != in.Purpose || existing.ResourceAlias != in.ResourceAlias || existing.PolicyVersion != in.PolicyVersion || (in.ExpiresAt != 0 && existing.ExpiresAt != in.ExpiresAt) {
		return conflict("request identity already exists with different immutable content")
	}
	return nil
}

func (s *Service) list(ctx context.Context, in Input, reconcile bool) (*List, error) {
	if err := validateText("owner", in.Owner, 256); err != nil {
		return nil, err
	}
	if in.Limit == 0 {
		in.Limit = 50
	}
	if in.Limit < 1 || in.Limit > 50 || in.Offset < 0 {
		return nil, fail("invalid_input", "limit must be 1..50 and offset nonnegative")
	}
	if in.Status != "" && in.Status != "all" && in.Status != "pending" && in.Status != "history" {
		return nil, fail("invalid_input", "status must be all, pending, or history")
	}
	status := in.Status
	if reconcile {
		status = "all"
	}
	raws, total, err := s.store.ListAccessApprovalContexts(ctx, in.Owner, in.Worker, status, s.now(), in.Limit, in.Offset)
	if err != nil {
		return nil, err
	}
	records := []*Request{}
	for _, raw := range raws {
		var r Request
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		fresh, err := s.mutate(ctx, r.ID, func(*Request) error { return nil })
		if err != nil {
			return nil, err
		}
		records = append(records, fresh)
	}
	return &List{Requests: records, Total: total, Limit: in.Limit, Offset: in.Offset}, nil
}
