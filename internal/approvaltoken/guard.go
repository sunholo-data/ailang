package approvaltoken

import "sync"

// SingleUseGuard enforces that a token nonce is accepted at most once. It is an
// in-memory set; for multi-instance deployments back it with shared state.
type SingleUseGuard struct {
	mu   sync.Mutex
	used map[string]struct{}
}

// NewSingleUseGuard returns an empty guard.
func NewSingleUseGuard() *SingleUseGuard {
	return &SingleUseGuard{used: make(map[string]struct{})}
}

// Use marks nonce as consumed. It returns ErrReused if the nonce was already
// seen, otherwise nil (and records it).
func (g *SingleUseGuard) Use(nonce string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.used[nonce]; ok {
		return ErrReused
	}
	g.used[nonce] = struct{}{}
	return nil
}
