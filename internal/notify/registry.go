package notify

import (
	"fmt"
	"sort"
	"sync"
)

// Registry maps channel name -> Channel. It is the single output router the
// notification dispatcher uses (Get(name).Send(...)), mirroring Aitana's
// ChannelRegistry. Unlike Aitana's process-global singleton, this is an
// instance so tests and multiple daemons stay isolated; the daemon holds one.
type Registry struct {
	mu       sync.RWMutex
	channels map[string]Channel
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{channels: make(map[string]Channel)}
}

// Register adds a channel under its Name. It is idempotent when re-registering
// the *same* instance, but errors when a *different* instance is registered for
// an already-taken name — that catches duplicate-registration bugs early
// (Aitana's hard-won rule). An empty Name is rejected.
func (r *Registry) Register(ch Channel) error {
	name := ch.Name()
	if name == "" {
		return fmt.Errorf("notify: channel has empty Name()")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.channels[name]; ok && existing != ch {
		return fmt.Errorf("notify: channel %q already registered with a different instance", name)
	}
	r.channels[name] = ch
	return nil
}

// Get returns the channel registered under name, or an error if none is.
func (r *Registry) Get(name string) (Channel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ch, ok := r.channels[name]
	if !ok {
		return nil, fmt.Errorf("notify: no channel registered with name %q", name)
	}
	return ch, nil
}

// Names returns the registered channel names in stable (sorted) order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.channels))
	for name := range r.channels {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
