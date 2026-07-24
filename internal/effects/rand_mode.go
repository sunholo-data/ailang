package effects

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"sync"
)

// M-EFFECT-REPLAY-CONTRACTS (v1.0.0, Rand pilot): mode-aware Rand dispatch.
//
// The declared Rand mode of a function reaches the builtin dispatch site via a
// per-EffContext stack, NOT via lowering (the doc-recommended (b) is infeasible
// because _rand_int is only referenced inside std/rand's os-mode wrappers, so
// the outer seeded/crypto mode is erased by the time the builtin runs — see the
// design doc Design-Freeze checkbox). The evaluator pushes a non-os mode at
// moded-lambda entry and pops it on exit; the Rand builtins read the top of the
// stack to choose their source.
//
// Sources:
//   - os (empty stack / default): the global randSource in builtins/rand.go,
//     unchanged, still reseedable via rand_seed. Handled entirely in builtins.
//   - seeded: a dedicated per-context PRNG seeded from AILANG_SEED (Env.Seed),
//     isolated from os draws so its sequence is unperturbable. No seed → typed
//     error at first draw.
//   - crypto: direct crypto/rand draws; entropy failure panics loudly.

// randModeState holds the per-execution Rand-mode dispatch state. It lives
// behind a pointer on EffContext so it can be SHARED across WithBudget scopes
// (the same logical execution) while being RESET per request in Clone. The
// mutex is inside this struct, so copying an EffContext never copies a lock.
type randModeState struct {
	mu     sync.Mutex
	stack  []string          // active NON-os modes; top = mode for the next draw
	seeded *seededRandSource // dedicated seeded PRNG, lazily built from Env.Seed
}

// seededRandSource is the dedicated deterministic PRNG for Rand[mode=seeded].
// It is separate from the os/global source so seeded sequences are never
// perturbed by interleaved os-mode draws (the core determinism guarantee).
type seededRandSource struct {
	r *rand.Rand
}

// randModeStateFor returns ctx's randModeState, lazily allocating it. Callers
// that only READ (CurrentRandMode) tolerate a nil state as "os"; mutating
// callers (Push/Pop/seeded draw) go through here.
func (ctx *EffContext) randModeStateFor() *randModeState {
	if ctx.randMode == nil {
		ctx.randMode = &randModeState{}
	}
	return ctx.randMode
}

// SeededModeError is the typed error returned when a Rand[mode=seeded] draw is
// attempted with no seed provided. Silent fallback to a random seed would make
// "deterministic" a lie (no-silent-fallbacks), so this fails loudly with a fix
// hint pointing to the dedicated seed path (AILANG_SEED) — NOT rand_seed, which
// seeds the os source.
type SeededModeError struct {
	Op string // e.g. "_rand_int"
}

func (e *SeededModeError) Error() string {
	return fmt.Sprintf(
		"RAND_SEEDED_NO_SEED: %s ran under Rand[mode=seeded] but no seed was provided. "+
			"mode=seeded requires an explicit seed for determinism and does NOT read rand_seed "+
			"(which seeds the os-mode source).\n"+
			"  Fix: set the AILANG_SEED environment variable (e.g. AILANG_SEED=42 ailang run ...) "+
			"to seed the deterministic source.", e.Op)
}

// PushRandMode pushes a non-os Rand mode onto the context's mode stack. os is
// never pushed (an empty stack already means os) so that a stdlib os-mode
// wrapper called from a seeded/crypto function does not shadow the outer mode.
// Paired with PopRandMode via a defer at the same lambda-entry site.
func (ctx *EffContext) PushRandMode(mode string) {
	if mode == "" || mode == "os" {
		return
	}
	st := ctx.randModeStateFor()
	st.mu.Lock()
	st.stack = append(st.stack, mode)
	st.mu.Unlock()
}

// PopRandMode pops the innermost mode pushed by PushRandMode. Passing os is a
// no-op (nothing was pushed), keeping push/pop balanced without the caller
// tracking whether a push happened.
func (ctx *EffContext) PopRandMode(mode string) {
	if mode == "" || mode == "os" {
		return
	}
	st := ctx.randModeStateFor()
	st.mu.Lock()
	if n := len(st.stack); n > 0 {
		st.stack = st.stack[:n-1]
	}
	st.mu.Unlock()
}

// CurrentRandMode returns the Rand mode in effect for the next draw: the top of
// the mode stack, or "os" when the stack is empty (the default).
func (ctx *EffContext) CurrentRandMode() string {
	if ctx.randMode == nil {
		return "os"
	}
	st := ctx.randMode
	st.mu.Lock()
	defer st.mu.Unlock()
	if n := len(st.stack); n > 0 {
		return st.stack[n-1]
	}
	return "os"
}

// SeededIntn returns a deterministic int in [0, n) from the per-context seeded
// source, initialising it from Env.Seed (AILANG_SEED) on first use. op is used
// only for the error message; a *SeededModeError is returned when AILANG_SEED
// was not set (no silent fallback).
func (ctx *EffContext) SeededIntn(op string, n int) (int, error) {
	src, err := ctx.seededSourceOrErr(op)
	if err != nil {
		return 0, err
	}
	return src.r.Intn(n), nil
}

// SeededFloat64 returns a deterministic float in [0.0, 1.0) from the per-context
// seeded source (see SeededIntn for the seed contract).
func (ctx *EffContext) SeededFloat64(op string) (float64, error) {
	src, err := ctx.seededSourceOrErr(op)
	if err != nil {
		return 0, err
	}
	return src.r.Float64(), nil
}

// seededSourceOrErr lazily builds the per-context seeded PRNG from Env.Seed, or
// returns a *SeededModeError if no seed was provided. Guarded by the state mutex
// so concurrent draws on a shared context are safe.
func (ctx *EffContext) seededSourceOrErr(op string) (*seededRandSource, error) {
	if !ctx.seedSet {
		return nil, &SeededModeError{Op: op}
	}
	st := ctx.randModeStateFor()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.seeded == nil {
		st.seeded = &seededRandSource{r: rand.New(rand.NewSource(ctx.Env.Seed))}
	}
	return st.seeded, nil
}

// SeedProvided reports whether AILANG_SEED was set for this context (gates the
// seeded-mode source — Env.Seed defaults to 0 when unset, a valid seed value).
func (ctx *EffContext) SeedProvided() bool {
	return ctx.seedSet
}

// SetSeededModeSeed explicitly provides the seed for the Rand[mode=seeded]
// source, marking it as available and resetting any already-drawn seeded
// sequence. This is the dedicated seeded-source API (distinct from rand_seed,
// which seeds the os source). AILANG_SEED sets the same seed at process start
// via NewEffContext; this setter lets embedders/tests provide it programmatically
// (e.g. a serve-api request that pins a replay seed).
func (ctx *EffContext) SetSeededModeSeed(seed int64) {
	ctx.Env.Seed = seed
	ctx.seedSet = true
	st := ctx.randModeStateFor()
	st.mu.Lock()
	st.seeded = &seededRandSource{r: rand.New(rand.NewSource(seed))}
	st.mu.Unlock()
}

// CryptoIntn returns a uniform, unbiased crypto/rand int in [0, n) for
// Rand[mode=crypto]. n must be > 0. Entropy failure panics loudly (matching the
// existing cryptoSeed stance in builtins/rand.go — a silent fallback to a
// predictable value would be worse for security-sensitive callers).
func CryptoIntn(n int) int {
	if n <= 0 {
		panic("effects/rand_mode: CryptoIntn requires n > 0")
	}
	// Rejection sampling for an unbiased result in [0, n): reject draws in the
	// final partial block so the modulo is uniform (avoids modulo bias).
	limit := ^uint64(0) - (^uint64(0) % uint64(n))
	for {
		v := cryptoUint64()
		if v < limit {
			return int(v % uint64(n))
		}
	}
}

// CryptoFloat64 returns a crypto/rand float in [0.0, 1.0) for Rand[mode=crypto].
// Uses 53 bits of entropy (the float64 mantissa) for a uniform result.
func CryptoFloat64() float64 {
	return float64(cryptoUint64()>>11) / (1 << 53)
}

// cryptoUint64 reads 8 bytes from crypto/rand as a uint64, panicking on entropy
// failure (no silent fallback — see CryptoIntn).
func cryptoUint64() uint64 {
	var b [8]byte
	if _, err := io.ReadFull(cryptorand.Reader, b[:]); err != nil {
		panic("effects/rand_mode: failed to read entropy from crypto/rand: " + err.Error())
	}
	return binary.LittleEndian.Uint64(b[:])
}
