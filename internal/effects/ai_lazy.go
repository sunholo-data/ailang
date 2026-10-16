package effects

import "sync"

// lazyAIContext publishes a fully constructed context or one cached error.
// Synchronization covers initialization/publication only; AIHandler operations
// retain their existing single-threaded contract.
type lazyAIContext struct {
	once       sync.Once
	mu         sync.RWMutex
	initialize func() (*AIContext, error)
	resolved   *AIContext
	err        error
}

// NewLazyAIContext defers provider construction until an AI operation. Hosts
// must prepare immutable configuration first. Capability/budget authorization
// remains at the effect dispatch boundary, before invoking this context.
func NewLazyAIContext(initialize func() (*AIContext, error)) *AIContext {
	return &AIContext{lazy: &lazyAIContext{initialize: initialize}}
}

func (c *AIContext) ready() (*AIContext, error) {
	if c == nil {
		return nil, ErrNoAIHandler
	}
	if c.lazy == nil {
		if c.handler == nil {
			return nil, ErrNoAIHandler
		}
		return c, nil
	}
	l := c.lazy
	l.once.Do(func() {
		var resolved *AIContext
		var err error
		if l.initialize == nil {
			err = ErrNoAIHandler
		} else {
			resolved, err = l.initialize()
		}
		if err == nil && (resolved == nil || resolved == c || resolved.lazy != nil || (resolved.handler == nil && resolved.embedder == nil)) {
			err = ErrNoAIHandler
		}
		if err == nil && c.modelResolver != nil {
			resolved.WithModelResolver(c.modelResolver)
		}
		l.mu.Lock()
		l.resolved, l.err, l.initialize = resolved, err, nil
		l.mu.Unlock()
	})
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.resolved, l.err
}

// Metadata inspection must not create a provider or read credentials.
func (l *lazyAIContext) metadataContext() *AIContext {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.err != nil {
		return nil
	}
	return l.resolved
}
