package eval

import (
	"fmt"
	"sync"
	"testing"
)

// TestEnvironmentConcurrentReadWrite verifies that Environment is safe for
// concurrent Get/Set/Clone operations. This tests the fix for the
// "undefined variable: node" bug under concurrent serve-api evaluation.
//
// Root cause: module-level closures share their captured Env across goroutines.
// Clone() reads the shared map while Set() writes from another goroutine.
// Without RWMutex protection, this is a Go concurrent map read+write crash.
func TestEnvironmentConcurrentReadWrite(t *testing.T) {
	// Create a "module-level" environment with bindings
	moduleEnv := NewEnvironment()
	moduleEnv.Set("processor", &StringValue{Value: "default"})
	moduleEnv.Set("config", &IntValue{Value: 42})

	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Simulate what CallFunction does: Clone the shared env, then Set locals
			localEnv := moduleEnv.Clone()
			localEnv.Set("node", &StringValue{Value: fmt.Sprintf("node-%d", idx)})
			localEnv.Set("index", &IntValue{Value: idx})

			// Read from the cloned env (should find both local and cloned values)
			if v, ok := localEnv.Get("node"); !ok {
				errors <- fmt.Sprintf("goroutine %d: node undefined", idx)
			} else if sv, ok := v.(*StringValue); !ok || sv.Value != fmt.Sprintf("node-%d", idx) {
				errors <- fmt.Sprintf("goroutine %d: node wrong value: %v", idx, v)
			}

			if v, ok := localEnv.Get("processor"); !ok {
				errors <- fmt.Sprintf("goroutine %d: processor undefined (inherited from module env)", idx)
			} else if sv, ok := v.(*StringValue); !ok || sv.Value != "default" {
				errors <- fmt.Sprintf("goroutine %d: processor wrong value: %v", idx, v)
			}

			if v, ok := localEnv.Get("config"); !ok {
				errors <- fmt.Sprintf("goroutine %d: config undefined", idx)
			} else if iv, ok := v.(*IntValue); !ok || iv.Value != 42 {
				errors <- fmt.Sprintf("goroutine %d: config wrong value: %v", idx, v)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	var errs []string
	for e := range errors {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		t.Errorf("%d errors in %d goroutines:\n%s", len(errs), numGoroutines, joinStrings(errs))
	}
}

// TestEnvironmentConcurrentCloneWhileSet verifies that Clone() is safe
// while another goroutine writes to the same Environment.
// This is the exact race condition from the DocParse bug.
func TestEnvironmentConcurrentCloneWhileSet(t *testing.T) {
	env := NewEnvironment()
	env.Set("x", &IntValue{Value: 1})

	const iterations = 1000
	var wg sync.WaitGroup

	// Writer goroutine: continuously sets values
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			env.Set(fmt.Sprintf("key%d", i%10), &IntValue{Value: i})
		}
	}()

	// Reader goroutines: continuously clone and read
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				clone := env.Clone()
				// Reading from clone should not panic or return corrupt data
				if v, ok := clone.Get("x"); ok {
					if iv, ok := v.(*IntValue); ok {
						_ = iv.Value // just access it
					}
				}
			}
		}()
	}

	wg.Wait()
	// If we get here without panic/race detector complaint, the test passes
}

// TestEnvironmentConcurrentParentChain verifies that reading through
// parent chain is safe under concurrent writes to child envs.
func TestEnvironmentConcurrentParentChain(t *testing.T) {
	grandparent := NewEnvironment()
	grandparent.Set("module_var", &StringValue{Value: "from_module"})

	parent := grandparent.NewChildEnvironment()
	parent.Set("func_var", &StringValue{Value: "from_function"})

	const numGoroutines = 50
	var wg sync.WaitGroup
	errors := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Each goroutine creates its own child (like CallFunction cloning fn.Env)
			child := parent.Clone()
			child.Set("local", &IntValue{Value: idx})

			// Should find local binding
			if _, ok := child.Get("local"); !ok {
				errors <- fmt.Sprintf("goroutine %d: local undefined", idx)
			}
			// Should find parent binding via chain
			if _, ok := child.Get("func_var"); !ok {
				errors <- fmt.Sprintf("goroutine %d: func_var undefined (parent)", idx)
			}
			// Should find grandparent binding via chain
			if _, ok := child.Get("module_var"); !ok {
				errors <- fmt.Sprintf("goroutine %d: module_var undefined (grandparent)", idx)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	var errs []string
	for e := range errors {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		t.Errorf("%d errors:\n%s", len(errs), joinStrings(errs))
	}
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += "\n"
		}
		result += s
	}
	return result
}
