package accessapproval

import "context"

func (s *Service) decide(ctx context.Context, in Input) (*Request, error) {
	if in.Decision != "approved" && in.Decision != "declined" {
		return nil, fail("invalid_input", "decision must be approved or declined")
	}
	if err := validateText("idempotency_key", in.IdempotencyKey, 256); err != nil {
		return nil, err
	}
	return s.mutate(ctx, in.ID, func(r *Request) error {
		if in.Actor != r.Owner || in.Version != r.Version {
			return fail("forbidden", "decision actor or request version does not match")
		}
		if r.Decision == in.Decision && r.DecisionKey == in.IdempotencyKey {
			return nil
		}
		if r.Decision != "pending" {
			return conflict("approval already has a final decision")
		}
		r.Decision = in.Decision
		r.DecisionBy = in.Actor
		r.DecisionAt = s.now()
		r.DecisionKey = in.IdempotencyKey
		if in.Decision == "approved" {
			r.ClaimUntil = min(r.ExpiresAt, s.now()+24*3600)
		}
		return nil
	})
}

func bound(r *Request, in Input) error {
	if r.Worker != in.Worker || r.Task != in.Task || r.Attempt != in.Attempt || r.Operation != in.Operation || r.ArgumentsDigest != in.ArgumentsDigest || r.PolicyVersion != in.PolicyVersion || r.Version != in.Version {
		return fail("forbidden", "worker/task/attempt/operation/digest/policy/version binding does not match")
	}
	return nil
}

func (s *Service) claim(ctx context.Context, in Input) (*Request, error) {
	if err := validateText("idempotency_key", in.IdempotencyKey, 256); err != nil {
		return nil, err
	}
	return s.mutate(ctx, in.ID, func(r *Request) error {
		if err := bound(r, in); err != nil {
			return err
		}
		if r.Decision != "approved" {
			return conflict("request is not approved")
		}
		if r.Execution.ClaimKey == in.IdempotencyKey {
			return nil
		}
		if r.Execution.State != "unclaimed" {
			return conflict("request has already been claimed")
		}
		id, err := opaqueID()
		if err != nil {
			return err
		}
		r.Execution = Execution{State: "claimed", ClaimID: id, ClaimKey: in.IdempotencyKey, ClaimUntil: min(r.ClaimUntil, s.now()+300)}
		return nil
	})
}

// execute is the protected resolve-and-execute seam. Neither references, shell
// commands nor arbitrary worker-written receipts cross it. Only persisted,
// fixed-registry arguments are used. Adding a real resolver requires a separate
// reviewed operation, protected deployment and constrained output policy.
func (s *Service) execute(ctx context.Context, in Input) (*Request, error) {
	ownsStart := false
	r, err := s.mutate(ctx, in.ID, func(r *Request) error {
		ownsStart = false // CAS retries must never inherit ownership of a lost start.
		if err := bound(r, in); err != nil {
			return err
		}
		if in.ClaimID == "" || in.ClaimID != r.Execution.ClaimID {
			return fail("forbidden", "claim id does not match")
		}
		if r.Decision != "approved" {
			return conflict("request is not approved")
		}
		switch r.Execution.State {
		case "completed", "failed", "unknown", "started":
			return nil
		case "claimed":
			// Persist the start before doing anything. An uncertain started record is
			// never blindly executed again, even by the same claimant.
			r.Execution.State = "started"
			r.Execution.StartedAt = s.now()
			ownsStart = true
			return nil
		default:
			return conflict("request has not been claimed")
		}
	})
	if err != nil || !ownsStart {
		return r, err
	}
	result := s.runDummy(r)
	return s.mutate(ctx, r.ID, func(current *Request) error {
		if current.Execution.State != "started" || current.Execution.ClaimID != in.ClaimID {
			return nil
		}
		current.Execution.State = "completed"
		current.Execution.CompletedAt = s.now()
		current.Execution.Result = result
		return nil
	})
}

func (s *Service) cancel(ctx context.Context, in Input) (*Request, error) {
	if len(in.Reason) > 1024 {
		return nil, fail("invalid_input", "reason exceeds 1024 bytes")
	}
	return s.mutate(ctx, in.ID, func(r *Request) error {
		if (in.Actor != r.Owner && in.Actor != r.Worker) || in.Version != r.Version {
			return fail("forbidden", "cancellation actor or version does not match")
		}
		if r.Decision == "cancelled" {
			return nil
		}
		if r.Decision != "pending" && r.Decision != "approved" {
			return conflict("request is already terminal")
		}
		if r.Execution.State == "completed" || r.Execution.State == "failed" || r.Execution.State == "unknown" {
			return conflict("execution is already terminal")
		}
		if r.Execution.State == "started" {
			r.Execution.State = "unknown"
			r.Execution.Error = "cancelled after execution started; operator reconciliation required"
			r.Execution.CompletedAt = s.now()
		}
		r.Decision = "cancelled"
		r.CancelReason = in.Reason
		r.DecisionBy = in.Actor
		r.DecisionAt = s.now()
		return nil
	})
}

// dummyOperation is the sole registered operation. Its internal injection seam
// exists for crash/race tests; no CLI input or environment selects a handler.
func dummyOperation(r *Request) map[string]any {
	result := map[string]any{"operation_id": r.ID, "message": "dummy operation completed"}
	if message, ok := r.Arguments["message"].(string); ok {
		result["message"] = message
	}
	return result
}
