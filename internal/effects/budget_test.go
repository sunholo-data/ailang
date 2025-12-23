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
		err := NewBudgetExhaustedError("IO", 5, 5, "")
		expected := "effect 'IO' budget exhausted: limit=5, used=5\nHint: Increase the budget with @limit=N or refactor to use fewer IO operations"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("error message with position", func(t *testing.T) {
		err := NewBudgetExhaustedError("IO", 5, 5, "file.ail:10:5")
		expected := "effect 'IO' budget exhausted: limit=5, used=5 at file.ail:10:5\nHint: Increase the budget with @limit=N or refactor to use fewer IO operations"
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
