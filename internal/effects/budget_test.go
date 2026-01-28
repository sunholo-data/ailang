package effects

import (
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestNewBudgetContext(t *testing.T) {
	t.Run("nil limits creates empty budget", func(t *testing.T) {
		bc := NewBudgetContext(nil)
		if bc.HasBudget("IO") {
			t.Error("expected no budget for IO")
		}
	})

	t.Run("limits with nil value are skipped", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{
			"IO": nil,
			"FS": intPtr(5),
		})
		if bc.HasBudget("IO") {
			t.Error("expected no budget for IO (nil limit)")
		}
		if !bc.HasBudget("FS") {
			t.Error("expected budget for FS")
		}
	})

	t.Run("limits are stored correctly", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{
			"IO": intPtr(5),
			"FS": intPtr(10),
		})

		limit, ok := bc.Limit("IO")
		if !ok || limit != 5 {
			t.Errorf("expected IO limit 5, got %d (ok=%v)", limit, ok)
		}

		limit, ok = bc.Limit("FS")
		if !ok || limit != 10 {
			t.Errorf("expected FS limit 10, got %d (ok=%v)", limit, ok)
		}
	})
}

func TestBudgetContext_CheckAndConsume(t *testing.T) {
	t.Run("unlimited effect always succeeds", func(t *testing.T) {
		bc := NewBudgetContext(nil)

		for i := 0; i < 100; i++ {
			if err := bc.CheckAndConsume("IO", ""); err != nil {
				t.Errorf("unexpected error on call %d: %v", i, err)
			}
		}
	})

	t.Run("limited effect succeeds until exhausted", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{"IO": intPtr(3)})

		// First 3 calls should succeed
		for i := 0; i < 3; i++ {
			if err := bc.CheckAndConsume("IO", ""); err != nil {
				t.Errorf("call %d should succeed: %v", i, err)
			}
		}

		// 4th call should fail
		err := bc.CheckAndConsume("IO", "test.ail:10:5")
		if err == nil {
			t.Error("expected BudgetExhaustedError on 4th call")
		}

		budgetErr, ok := err.(*BudgetExhaustedError)
		if !ok {
			t.Errorf("expected *BudgetExhaustedError, got %T", err)
		} else {
			if budgetErr.Effect != "IO" {
				t.Errorf("expected effect IO, got %s", budgetErr.Effect)
			}
			if budgetErr.Limit != 3 {
				t.Errorf("expected limit 3, got %d", budgetErr.Limit)
			}
			if budgetErr.Used != 3 {
				t.Errorf("expected used 3, got %d", budgetErr.Used)
			}
			if budgetErr.Position != "test.ail:10:5" {
				t.Errorf("expected position 'test.ail:10:5', got %s", budgetErr.Position)
			}
		}
	})

	t.Run("zero budget fails immediately", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{"IO": intPtr(0)})

		err := bc.CheckAndConsume("IO", "")
		if err == nil {
			t.Error("expected BudgetExhaustedError on first call with zero budget")
		}
	})

	t.Run("different effects have separate budgets", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{
			"IO": intPtr(2),
			"FS": intPtr(3),
		})

		// Use all IO budget
		bc.CheckAndConsume("IO", "")
		bc.CheckAndConsume("IO", "")
		if err := bc.CheckAndConsume("IO", ""); err == nil {
			t.Error("expected IO budget exhausted")
		}

		// FS should still have budget
		for i := 0; i < 3; i++ {
			if err := bc.CheckAndConsume("FS", ""); err != nil {
				t.Errorf("FS call %d should succeed: %v", i, err)
			}
		}
	})
}

func TestBudgetContext_Used(t *testing.T) {
	bc := NewBudgetContext(map[string]*int{"IO": intPtr(5)})

	if bc.Used("IO") != 0 {
		t.Error("expected 0 used initially")
	}

	bc.CheckAndConsume("IO", "")
	bc.CheckAndConsume("IO", "")

	if bc.Used("IO") != 2 {
		t.Errorf("expected 2 used, got %d", bc.Used("IO"))
	}
}

func TestBudgetContext_Remaining(t *testing.T) {
	t.Run("unlimited effect returns -1", func(t *testing.T) {
		bc := NewBudgetContext(nil)
		if bc.Remaining("IO") != -1 {
			t.Errorf("expected -1 for unlimited, got %d", bc.Remaining("IO"))
		}
	})

	t.Run("limited effect returns remaining count", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{"IO": intPtr(5)})

		if bc.Remaining("IO") != 5 {
			t.Errorf("expected 5 remaining, got %d", bc.Remaining("IO"))
		}

		bc.CheckAndConsume("IO", "")
		bc.CheckAndConsume("IO", "")

		if bc.Remaining("IO") != 3 {
			t.Errorf("expected 3 remaining, got %d", bc.Remaining("IO"))
		}
	})

	t.Run("exhausted effect returns 0", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{"IO": intPtr(1)})
		bc.CheckAndConsume("IO", "")

		if bc.Remaining("IO") != 0 {
			t.Errorf("expected 0 remaining, got %d", bc.Remaining("IO"))
		}
	})
}

func TestBudgetContext_Clone(t *testing.T) {
	bc := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
	bc.CheckAndConsume("IO", "")
	bc.CheckAndConsume("IO", "")

	clone := bc.Clone()

	// Clone should have same limits
	if limit, ok := clone.Limit("IO"); !ok || limit != 5 {
		t.Errorf("expected clone to have IO limit 5, got %d", limit)
	}

	// Clone should have fresh usage counters
	if clone.Used("IO") != 0 {
		t.Errorf("expected clone to have 0 used, got %d", clone.Used("IO"))
	}

	// Original should be unaffected
	if bc.Used("IO") != 2 {
		t.Errorf("expected original to have 2 used, got %d", bc.Used("IO"))
	}
}

func TestBudgetContext_Reset(t *testing.T) {
	bc := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
	bc.CheckAndConsume("IO", "")
	bc.CheckAndConsume("IO", "")

	bc.Reset()

	if bc.Used("IO") != 0 {
		t.Errorf("expected 0 used after reset, got %d", bc.Used("IO"))
	}

	// Limits should still be in place
	if limit, ok := bc.Limit("IO"); !ok || limit != 5 {
		t.Errorf("expected limit 5 after reset, got %d", limit)
	}
}

func TestBudgetContext_Merge(t *testing.T) {
	t.Run("merge with nil returns clone", func(t *testing.T) {
		bc := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
		merged := bc.Merge(nil)

		if limit, ok := merged.Limit("IO"); !ok || limit != 5 {
			t.Errorf("expected merged to have IO limit 5, got %d", limit)
		}
	})

	t.Run("merge sums matching limits", func(t *testing.T) {
		a := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
		b := NewBudgetContext(map[string]*int{"IO": intPtr(3)})

		merged := a.Merge(b)

		if limit, ok := merged.Limit("IO"); !ok || limit != 8 {
			t.Errorf("expected merged IO limit 8, got %d", limit)
		}
	})

	t.Run("merge preserves non-overlapping limits", func(t *testing.T) {
		a := NewBudgetContext(map[string]*int{"IO": intPtr(5)})
		b := NewBudgetContext(map[string]*int{"FS": intPtr(3)})

		merged := a.Merge(b)

		if limit, ok := merged.Limit("IO"); !ok || limit != 5 {
			t.Errorf("expected merged IO limit 5, got %d", limit)
		}
		if limit, ok := merged.Limit("FS"); !ok || limit != 3 {
			t.Errorf("expected merged FS limit 3, got %d", limit)
		}
	})
}

func TestBudgetExhaustedError(t *testing.T) {
	t.Run("error message without position", func(t *testing.T) {
		// M-DX25: Now includes physical count
		err := NewBudgetExhaustedError("IO", 5, 5, "", 10)
		expected := "effect 'IO' budget exhausted: semantic limit=5, used=5 (physical: 10)\nHint: Increase the budget with @limit=N or refactor to use fewer IO operations"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("error message with position", func(t *testing.T) {
		err := NewBudgetExhaustedError("IO", 5, 5, "file.ail:10:5", 15)
		expected := "effect 'IO' budget exhausted: semantic limit=5, used=5 (physical: 15) at file.ail:10:5\nHint: Increase the budget with @limit=N or refactor to use fewer IO operations"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})
}

func TestEffContext_RequireCapWithBudget(t *testing.T) {
	t.Run("fails without capability", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(5)}))

		err := ctx.RequireCapWithBudget("IO", "")
		if err == nil {
			t.Error("expected CapabilityError")
		}
		if _, ok := err.(*CapabilityError); !ok {
			t.Errorf("expected *CapabilityError, got %T", err)
		}
	})

	t.Run("succeeds with capability and budget", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(2)}))

		// First two calls should succeed
		if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
			t.Errorf("first call should succeed: %v", err)
		}
		if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
			t.Errorf("second call should succeed: %v", err)
		}

		// Third call should fail with budget error
		err := ctx.RequireCapWithBudget("IO", "test.ail:5:3")
		if err == nil {
			t.Error("expected BudgetExhaustedError on third call")
		}
		if _, ok := err.(*BudgetExhaustedError); !ok {
			t.Errorf("expected *BudgetExhaustedError, got %T", err)
		}
	})

	t.Run("succeeds with capability and no budget", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		// No budget set

		for i := 0; i < 10; i++ {
			if err := ctx.RequireCapWithBudget("IO", ""); err != nil {
				t.Errorf("call %d should succeed: %v", i, err)
			}
		}
	})
}

func TestEffContext_WithBudget(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))

	budget := NewBudgetContext(map[string]*int{"IO": intPtr(3)})
	ctxWithBudget := ctx.WithBudget(budget)

	// Original should not have budget
	if ctx.Budget != nil {
		t.Error("original should not have budget")
	}

	// New context should have budget
	if ctxWithBudget.Budget == nil {
		t.Error("new context should have budget")
	}

	// Both should share capabilities
	if !ctxWithBudget.HasCap("IO") {
		t.Error("new context should have IO capability")
	}
}

// M-DX25: Scoped Caller Charging Tests

func TestBudgetContext_ChargeSemanticOnly(t *testing.T) {
	bc := NewBudgetContext(map[string]*int{"IO": intPtr(10)})

	// Initial state
	if bc.Used("IO") != 0 {
		t.Errorf("expected 0 used, got %d", bc.Used("IO"))
	}
	if bc.PhysicalUsed("IO") != 0 {
		t.Errorf("expected 0 physical, got %d", bc.PhysicalUsed("IO"))
	}

	// ChargeSemanticOnly should only increment semantic
	bc.ChargeSemanticOnly("IO", 5)

	if bc.Used("IO") != 5 {
		t.Errorf("expected 5 used, got %d", bc.Used("IO"))
	}
	if bc.PhysicalUsed("IO") != 0 {
		t.Errorf("expected 0 physical (not incremented by ChargeSemanticOnly), got %d", bc.PhysicalUsed("IO"))
	}

	// Remaining should reflect semantic usage
	if bc.Remaining("IO") != 5 {
		t.Errorf("expected 5 remaining, got %d", bc.Remaining("IO"))
	}
}

func TestEffContext_WithBudgetLimits_StoresCallerReference(t *testing.T) {
	caller := NewEffContext(nil)
	caller.Grant(NewCapability("IO"))
	caller.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

	// Create callee context with declared limits
	calleeRaw := caller.WithBudgetLimits(map[string]int{"IO": 3})
	callee, ok := calleeRaw.(*EffContext)
	if !ok {
		t.Fatalf("expected *EffContext, got %T", calleeRaw)
	}

	// Callee should have CallerContext pointing to caller
	if callee.CallerContext != caller {
		t.Error("callee.CallerContext should reference caller")
	}

	// Callee should have DeclaredBudgets
	if callee.DeclaredBudgets["IO"] != 3 {
		t.Errorf("expected declared IO=3, got %d", callee.DeclaredBudgets["IO"])
	}
}

func TestEffContext_PopScopeAndChargeCaller(t *testing.T) {
	t.Run("charges caller semantic budget with declared amount", func(t *testing.T) {
		caller := NewEffContext(nil)
		caller.Grant(NewCapability("IO"))
		caller.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

		// Create callee context with declared limit of 3 IO operations
		calleeRaw := caller.WithBudgetLimits(map[string]int{"IO": 3})
		callee := calleeRaw.(*EffContext)

		// Simulate callee making actual builtin calls (physical usage)
		callee.Budget.CheckAndConsume("IO", "")
		callee.Budget.CheckAndConsume("IO", "")
		// Callee made 2 actual calls but declared @limit=3

		// Verify callee's physical count
		if callee.Budget.PhysicalUsed("IO") != 2 {
			t.Errorf("expected callee physical=2, got %d", callee.Budget.PhysicalUsed("IO"))
		}

		// Before PopScopeAndChargeCaller, caller should have 0 semantic usage
		if caller.Budget.Used("IO") != 0 {
			t.Errorf("expected caller used=0 before pop, got %d", caller.Budget.Used("IO"))
		}

		// Pop scope and charge caller
		callee.PopScopeAndChargeCaller()

		// Caller should be charged declared amount (3), not actual (2)
		if caller.Budget.Used("IO") != 3 {
			t.Errorf("expected caller used=3 (declared), got %d", caller.Budget.Used("IO"))
		}

		// Caller's physical count should NOT be affected
		if caller.Budget.PhysicalUsed("IO") != 0 {
			t.Errorf("expected caller physical=0, got %d", caller.Budget.PhysicalUsed("IO"))
		}
	})

	t.Run("no-op when CallerContext is nil (pass-through)", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

		// No CallerContext (pass-through mode)
		ctx.PopScopeAndChargeCaller() // Should be no-op

		// Budget should be unchanged
		if ctx.Budget.Used("IO") != 0 {
			t.Errorf("expected 0 used, got %d", ctx.Budget.Used("IO"))
		}
	})

	t.Run("no-op when DeclaredBudgets is empty", func(t *testing.T) {
		caller := NewEffContext(nil)
		caller.Grant(NewCapability("IO"))
		caller.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

		callee := caller.WithBudget(caller.Budget.Clone())
		callee.CallerContext = caller
		callee.DeclaredBudgets = nil // Empty declared budgets

		callee.PopScopeAndChargeCaller() // Should be no-op

		if caller.Budget.Used("IO") != 0 {
			t.Errorf("expected 0 used, got %d", caller.Budget.Used("IO"))
		}
	})
}

func TestScopedBudget_NestedCalls(t *testing.T) {
	// Simulate: main (limit 10) -> foo (limit 5) -> bar (limit 2)
	// Each function may make fewer actual calls than declared

	main := NewEffContext(nil)
	main.Grant(NewCapability("IO"))
	main.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

	// main calls foo (declared @limit=5)
	fooRaw := main.WithBudgetLimits(map[string]int{"IO": 5})
	foo := fooRaw.(*EffContext)

	// foo calls bar (declared @limit=2)
	barRaw := foo.WithBudgetLimits(map[string]int{"IO": 2})
	bar := barRaw.(*EffContext)

	// bar makes 1 actual call
	bar.Budget.CheckAndConsume("IO", "")

	// bar returns -> charges foo 2 (declared)
	bar.PopScopeAndChargeCaller()
	if foo.Budget.Used("IO") != 2 {
		t.Errorf("expected foo charged 2, got %d", foo.Budget.Used("IO"))
	}

	// foo makes 1 more actual call (total physical: bar's 1 + foo's 1 = 2, but foo's budget tracked separately)
	foo.Budget.CheckAndConsume("IO", "")

	// foo returns -> charges main 5 (declared)
	foo.PopScopeAndChargeCaller()
	if main.Budget.Used("IO") != 5 {
		t.Errorf("expected main charged 5, got %d", main.Budget.Used("IO"))
	}

	// main has 5 remaining
	if main.Budget.Remaining("IO") != 5 {
		t.Errorf("expected main remaining 5, got %d", main.Budget.Remaining("IO"))
	}
}

// M-DX25 M4: Minimum budget tests

func TestNewBudgetContextWithMin(t *testing.T) {
	t.Run("nil inputs create empty budget", func(t *testing.T) {
		bc := NewBudgetContextWithMin(nil, nil)
		if len(bc.minLimits) != 0 {
			t.Errorf("expected empty minLimits, got %v", bc.minLimits)
		}
	})

	t.Run("min limits are stored correctly", func(t *testing.T) {
		limits := map[string]*int{"IO": intPtr(5)}
		mins := map[string]*int{"IO": intPtr(2)}
		bc := NewBudgetContextWithMin(limits, mins)

		if min, ok := bc.Minimum("IO"); !ok || min != 2 {
			t.Errorf("expected min 2, got %d (ok=%v)", min, ok)
		}
	})

	t.Run("nil min values are skipped", func(t *testing.T) {
		mins := map[string]*int{"IO": nil, "FS": intPtr(3)}
		bc := NewBudgetContextWithMin(nil, mins)

		if _, ok := bc.Minimum("IO"); ok {
			t.Error("IO should not have a minimum (nil value)")
		}
		if min, ok := bc.Minimum("FS"); !ok || min != 3 {
			t.Errorf("expected FS min 3, got %d", min)
		}
	})
}

func TestBudgetContext_HasMinimum(t *testing.T) {
	mins := map[string]*int{"IO": intPtr(2)}
	bc := NewBudgetContextWithMin(nil, mins)

	if !bc.HasMinimum("IO") {
		t.Error("expected HasMinimum(IO) to be true")
	}
	if bc.HasMinimum("FS") {
		t.Error("expected HasMinimum(FS) to be false")
	}
}

func TestBudgetContext_MinLimitsMap(t *testing.T) {
	mins := map[string]*int{"IO": intPtr(2), "FS": intPtr(3)}
	bc := NewBudgetContextWithMin(nil, mins)

	m := bc.MinLimitsMap()
	if len(m) != 2 {
		t.Errorf("expected 2 min limits, got %d", len(m))
	}
	if m["IO"] != 2 {
		t.Errorf("expected IO min 2, got %d", m["IO"])
	}
	if m["FS"] != 3 {
		t.Errorf("expected FS min 3, got %d", m["FS"])
	}
}

func TestBudgetContext_CheckMinimum(t *testing.T) {
	t.Run("passes when minimum is met", func(t *testing.T) {
		mins := map[string]*int{"IO": intPtr(2)}
		bc := NewBudgetContextWithMin(map[string]*int{"IO": intPtr(10)}, mins)

		// Make 2 calls (physical usage)
		bc.CheckAndConsume("IO", "")
		bc.CheckAndConsume("IO", "")

		err := bc.CheckMinimum("")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("passes when usage exceeds minimum", func(t *testing.T) {
		mins := map[string]*int{"IO": intPtr(2)}
		bc := NewBudgetContextWithMin(map[string]*int{"IO": intPtr(10)}, mins)

		// Make 5 calls
		for i := 0; i < 5; i++ {
			bc.CheckAndConsume("IO", "")
		}

		err := bc.CheckMinimum("")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("fails when minimum not met", func(t *testing.T) {
		mins := map[string]*int{"IO": intPtr(3)}
		bc := NewBudgetContextWithMin(map[string]*int{"IO": intPtr(10)}, mins)

		// Make only 2 calls when 3 required
		bc.CheckAndConsume("IO", "")
		bc.CheckAndConsume("IO", "")

		err := bc.CheckMinimum("test.ail:10:5")
		if err == nil {
			t.Fatal("expected BudgetUnderrunError")
		}

		underrun, ok := err.(*BudgetUnderrunError)
		if !ok {
			t.Fatalf("expected *BudgetUnderrunError, got %T", err)
		}
		if underrun.Effect != "IO" {
			t.Errorf("expected effect IO, got %s", underrun.Effect)
		}
		if underrun.Min != 3 {
			t.Errorf("expected min 3, got %d", underrun.Min)
		}
		if underrun.Actual != 2 {
			t.Errorf("expected actual 2, got %d", underrun.Actual)
		}
	})

	t.Run("multiple effects - one fails", func(t *testing.T) {
		mins := map[string]*int{"IO": intPtr(2), "FS": intPtr(3)}
		limits := map[string]*int{"IO": intPtr(10), "FS": intPtr(10)}
		bc := NewBudgetContextWithMin(limits, mins)

		// IO: 2 calls (meets min of 2)
		bc.CheckAndConsume("IO", "")
		bc.CheckAndConsume("IO", "")

		// FS: 1 call (below min of 3)
		bc.CheckAndConsume("FS", "")

		err := bc.CheckMinimum("")
		if err == nil {
			t.Fatal("expected error for FS underrun")
		}

		underrun := err.(*BudgetUnderrunError)
		if underrun.Effect != "FS" {
			t.Errorf("expected FS to fail, got %s", underrun.Effect)
		}
	})
}

func TestEffContext_SetMinBudgets(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("IO"))
	ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))

	// Set min budgets
	ctx.SetMinBudgets(map[string]int{"IO": 2})

	if !ctx.Budget.HasMinimum("IO") {
		t.Error("expected IO to have minimum after SetMinBudgets")
	}
	if min, _ := ctx.Budget.Minimum("IO"); min != 2 {
		t.Errorf("expected IO min 2, got %d", min)
	}
}

func TestEffContext_CheckMinimums(t *testing.T) {
	t.Run("passes when minimums met", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))
		ctx.SetMinBudgets(map[string]int{"IO": 2})

		// Make 2 calls
		ctx.RequireCapWithBudget("IO", "")
		ctx.RequireCapWithBudget("IO", "")

		err := ctx.CheckMinimums("")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("fails when minimums not met", func(t *testing.T) {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("IO"))
		ctx.SetBudget(NewBudgetContext(map[string]*int{"IO": intPtr(10)}))
		ctx.SetMinBudgets(map[string]int{"IO": 3})

		// Make only 1 call when 3 required
		ctx.RequireCapWithBudget("IO", "")

		err := ctx.CheckMinimums("test.ail:5:1")
		if err == nil {
			t.Fatal("expected BudgetUnderrunError")
		}

		underrun, ok := err.(*BudgetUnderrunError)
		if !ok {
			t.Fatalf("expected *BudgetUnderrunError, got %T", err)
		}
		if underrun.Actual != 1 || underrun.Min != 3 {
			t.Errorf("expected actual=1, min=3, got actual=%d, min=%d", underrun.Actual, underrun.Min)
		}
	})
}

func TestBudgetUnderrunError(t *testing.T) {
	t.Run("error message without position", func(t *testing.T) {
		err := NewBudgetUnderrunError("IO", 3, 1, "")
		expected := "effect 'IO' budget underrun: min=3, actual=1\nHint: Ensure the function actually performs at least 3 IO operation(s)"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("error message with position", func(t *testing.T) {
		err := NewBudgetUnderrunError("IO", 3, 1, "test.ail:5:1")
		expected := "effect 'IO' budget underrun: min=3, actual=1 at test.ail:5:1\nHint: Ensure the function actually performs at least 3 IO operation(s)"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})
}

func TestBudgetContext_Clone_PreservesMinLimits(t *testing.T) {
	mins := map[string]*int{"IO": intPtr(2), "FS": intPtr(3)}
	limits := map[string]*int{"IO": intPtr(10)}
	bc := NewBudgetContextWithMin(limits, mins)

	clone := bc.Clone()

	// Check min limits are cloned
	if min, ok := clone.Minimum("IO"); !ok || min != 2 {
		t.Errorf("expected cloned IO min 2, got %d", min)
	}
	if min, ok := clone.Minimum("FS"); !ok || min != 3 {
		t.Errorf("expected cloned FS min 3, got %d", min)
	}

	// Verify independence
	bc.minLimits["IO"] = 99
	if min, _ := clone.Minimum("IO"); min != 2 {
		t.Error("clone should be independent of original")
	}
}
