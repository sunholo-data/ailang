package notify

import (
	"context"
	"fmt"
	"log"
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

// SendAll delivers n to every registered channel, then decides ack vs retry by
// the "local best-effort, remote authoritative" policy:
//
//   - Local channels (macOS desktop) are fired but never gate the ack and never
//     trigger a retry — they can report success by merely queueing a banner the
//     absent user never sees.
//   - Remote channels (Discord, etc.) are authoritative: the ack requires EVERY
//     remote channel to succeed. If any remote channel fails, SendAll returns an
//     error so the daemon nacks and Pub/Sub redelivers (re-firing remotes; the
//     local channel's per-task Group coalesces its repeat banner).
//   - If there are no remote channels at all, SendAll falls back to local
//     best-effort: nil if at least one local channel delivered (today's macOS
//     behaviour), else the last local error.
//
// Per-channel failures are logged regardless.
func (r *Registry) SendAll(ctx context.Context, n Notification, logger *log.Logger) error {
	names := r.Names()
	if len(names) == 0 {
		return nil
	}

	var remoteTotal, remoteOK int
	localDelivered := false
	var lastRemoteErr, lastLocalErr error

	for _, name := range names {
		ch, err := r.Get(name)
		if err != nil {
			continue
		}
		sendErr := ch.Send(ctx, n)
		if isLocal(ch) {
			if sendErr != nil {
				lastLocalErr = sendErr
				logf(logger, "notify: local channel %q send failed: %v", name, sendErr)
			} else {
				localDelivered = true
			}
			continue
		}
		// Remote/authoritative channel.
		remoteTotal++
		if sendErr != nil {
			lastRemoteErr = sendErr
			logf(logger, "notify: remote channel %q send failed: %v", name, sendErr)
		} else {
			remoteOK++
		}
	}

	if remoteTotal > 0 {
		if remoteOK == remoteTotal {
			return nil
		}
		return lastRemoteErr
	}
	// No remote channels: fall back to local best-effort.
	if localDelivered {
		return nil
	}
	return lastLocalErr
}

// FanOut adapts the registry to the daemon's notifier shape
// (func(Notification) error), delivering to every channel via SendAll.
func (r *Registry) FanOut(logger *log.Logger) func(Notification) error {
	return func(n Notification) error {
		return r.SendAll(context.Background(), n, logger)
	}
}
