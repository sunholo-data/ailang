package coordinator

import (
	"fmt"
	"sync"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// ScheduleCascadeUpdate determines the execution order for packages affected by
// an update to triggerPkg. Returns package names in topological order (leaf deps first).
// Returns an error if a cycle is detected.
func ScheduleCascadeUpdate(index *pkg.RegistryIndex, triggerPkg string) ([]string, error) {
	// Build reverse dependency graph: for each package, which packages depend on it
	dependents := make(map[string][]string)
	allPkgs := make(map[string]bool)
	for _, entry := range index.Packages {
		allPkgs[entry.Name] = true
		for _, dep := range entry.Dependencies {
			dependents[dep] = append(dependents[dep], entry.Name)
		}
	}

	if !allPkgs[triggerPkg] {
		return nil, nil // Package not in index
	}

	// BFS from triggerPkg through dependents to find all affected packages
	affected := make(map[string]bool)
	queue := []string{triggerPkg}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dep := range dependents[current] {
			if !affected[dep] {
				affected[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	if len(affected) == 0 {
		return nil, nil
	}

	// Topological sort of affected packages using Kahn's algorithm.
	// Build sub-graph of only affected packages.
	inDegree := make(map[string]int)
	subDeps := make(map[string][]string) // forward edges within affected set
	for _, entry := range index.Packages {
		if !affected[entry.Name] {
			continue
		}
		inDegree[entry.Name] = 0
	}
	for _, entry := range index.Packages {
		if !affected[entry.Name] {
			continue
		}
		for _, dep := range entry.Dependencies {
			if affected[dep] || dep == triggerPkg {
				inDegree[entry.Name]++
				subDeps[dep] = append(subDeps[dep], entry.Name)
			}
		}
	}

	// Start with packages whose only affected dep is the trigger itself
	var ready []string
	for name, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, name)
		}
	}
	// Also add direct dependents of triggerPkg that have degree > 0
	// but whose non-trigger deps are all outside the affected set
	for _, dep := range dependents[triggerPkg] {
		if affected[dep] && inDegree[dep] == 1 {
			// Only dependency is the trigger
			found := false
			for _, r := range ready {
				if r == dep {
					found = true
					break
				}
			}
			if !found {
				ready = append(ready, dep)
				inDegree[dep] = 0
			}
		}
	}

	var order []string
	for len(ready) > 0 {
		current := ready[0]
		ready = ready[1:]
		order = append(order, current)
		for _, next := range subDeps[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				ready = append(ready, next)
			}
		}
	}

	if len(order) != len(affected) {
		return nil, fmt.Errorf("cycle detected in dependency graph among affected packages")
	}

	return order, nil
}

// CascadeCircuitBreaker tracks failures during a cascade update.
// If MaxFailures consecutive failures occur, IsBroken returns true.
type CascadeCircuitBreaker struct {
	MaxFailures   int
	CorrelationID string

	mu           sync.Mutex
	failureCount int
}

// RecordSuccess resets the consecutive failure counter.
func (cb *CascadeCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount = 0
}

// RecordFailure increments the consecutive failure counter.
func (cb *CascadeCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failureCount++
}

// IsBroken returns true if consecutive failures have reached the threshold.
func (cb *CascadeCircuitBreaker) IsBroken() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount >= cb.MaxFailures
}

// FailureCount returns the current consecutive failure count.
func (cb *CascadeCircuitBreaker) FailureCount() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failureCount
}
