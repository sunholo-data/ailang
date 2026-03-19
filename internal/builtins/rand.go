package builtins

import (
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"sync"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// randSource is the global random source for the Rand effect.
// Use SetRandSeed to set the seed for deterministic behavior.
var (
	randSource *rand.Rand
	randMu     sync.Mutex
)

func init() {
	// Initialize with a default seed (can be overridden with SetRandSeed)
	randSource = rand.New(rand.NewSource(0))

	registerRandInt()
	registerRandFloat()
	registerRandBool()
	registerRandSeed()
	registerUuid4()
}

// SetRandSeed sets the random seed for deterministic random generation.
// Call this at the start of a game session with a known seed for reproducible results.
func SetRandSeed(seed int64) {
	randMu.Lock()
	defer randMu.Unlock()
	randSource = rand.New(rand.NewSource(seed))
}

// _rand_int: Generate random integer in range [min, max]
func registerRandInt() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/rand",
		Name:    "_rand_int",
		NumArgs: 2,
		Effect:  "Rand",
		Type:    makeRandIntType,
		Impl:    randIntImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate random integer in range [min, max] inclusive",
			Params: []ParamDoc{
				{Name: "min", Description: "Minimum value (inclusive)"},
				{Name: "max", Description: "Maximum value (inclusive)"},
			},
			Returns: "Random integer in [min, max]",
			Examples: []Example{
				{Code: "_rand_int(1, 6)", Description: "Roll a d6 (returns 1-6)"},
				{Code: "_rand_int(0, 100)", Description: "Random percentage"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"random", "rand", "int", "game"},
			Category:  "rand",
		},
	})
	if err != nil {
		panic("failed to register _rand_int builtin: " + err.Error())
	}
}

func makeRandIntType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Int(), T.Int()).Returns(T.Int()).Effects("Rand")
}

func randIntImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Rand capability
	if err := ctx.RequireCapWithBudget("Rand", ""); err != nil {
		return nil, err
	}

	min := args[0].(*eval.IntValue).Value
	max := args[1].(*eval.IntValue).Value

	// Handle edge cases
	if min > max {
		min, max = max, min // Swap if reversed
	}

	randMu.Lock()
	result := min + randSource.Intn(max-min+1)
	randMu.Unlock()

	return &eval.IntValue{Value: result}, nil
}

// _rand_float: Generate random float in range [min, max)
func registerRandFloat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/rand",
		Name:    "_rand_float",
		NumArgs: 2,
		Effect:  "Rand",
		Type:    makeRandFloatType,
		Impl:    randFloatImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate random float in range [min, max)",
			Params: []ParamDoc{
				{Name: "min", Description: "Minimum value (inclusive)"},
				{Name: "max", Description: "Maximum value (exclusive)"},
			},
			Returns: "Random float in [min, max)",
			Examples: []Example{
				{Code: "_rand_float(0.0, 1.0)", Description: "Random float 0-1"},
				{Code: "_rand_float(-10.0, 10.0)", Description: "Random float in range"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"random", "rand", "float", "game"},
			Category:  "rand",
		},
	})
	if err != nil {
		panic("failed to register _rand_float builtin: " + err.Error())
	}
}

func makeRandFloatType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Float(), T.Float()).Returns(T.Float()).Effects("Rand")
}

func randFloatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Rand capability
	if err := ctx.RequireCapWithBudget("Rand", ""); err != nil {
		return nil, err
	}

	min := args[0].(*eval.FloatValue).Value
	max := args[1].(*eval.FloatValue).Value

	// Handle edge cases
	if min > max {
		min, max = max, min
	}

	randMu.Lock()
	result := min + randSource.Float64()*(max-min)
	randMu.Unlock()

	return &eval.FloatValue{Value: result}, nil
}

// _rand_bool: Generate random boolean
func registerRandBool() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/rand",
		Name:    "_rand_bool",
		NumArgs: 1, // Unit-argument model
		Effect:  "Rand",
		Type:    makeRandBoolType,
		Impl:    randBoolImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate random boolean (50/50 chance)",
			Params:      []ParamDoc{},
			Returns:     "Random boolean (true or false)",
			Examples: []Example{
				{Code: "_rand_bool()", Description: "Coin flip"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"random", "rand", "bool", "game"},
			Category:  "rand",
		},
	})
	if err != nil {
		panic("failed to register _rand_bool builtin: " + err.Error())
	}
}

func makeRandBoolType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.Bool()).Effects("Rand")
}

func randBoolImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Rand capability
	if err := ctx.RequireCapWithBudget("Rand", ""); err != nil {
		return nil, err
	}

	// Validate unit argument
	if len(args) != 1 {
		panic("internal invariant violation: _rand_bool expects exactly 1 argument (unit)")
	}

	randMu.Lock()
	result := randSource.Intn(2) == 1
	randMu.Unlock()

	return &eval.BoolValue{Value: result}, nil
}

// _rand_seed: Set the random seed for deterministic generation
func registerRandSeed() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/rand",
		Name:    "_rand_seed",
		NumArgs: 1,
		Effect:  "Rand",
		Type:    makeRandSeedType,
		Impl:    randSeedImpl,
		Metadata: &BuiltinMetadata{
			Description: "Set random seed for deterministic generation",
			Params: []ParamDoc{
				{Name: "seed", Description: "Seed value for random number generator"},
			},
			Returns: "Unit (no return value)",
			SeeAlso: []string{"_rand_int", "_rand_float", "_rand_bool"},
			Examples: []Example{
				{Code: "_rand_seed(42)", Description: "Set seed to 42 for reproducible results"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"random", "rand", "seed", "game", "determinism"},
			Category:  "rand",
		},
	})
	if err != nil {
		panic("failed to register _rand_seed builtin: " + err.Error())
	}
}

func makeRandSeedType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Int()).Returns(T.Unit()).Effects("Rand")
}

func randSeedImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Rand capability
	if err := ctx.RequireCapWithBudget("Rand", ""); err != nil {
		return nil, err
	}

	seed := int64(args[0].(*eval.IntValue).Value)
	SetRandSeed(seed)

	return &eval.UnitValue{}, nil
}

// _uuid4: Generate a random UUID v4
func registerUuid4() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/rand",
		Name:    "_uuid4",
		NumArgs: 1, // unit arg
		IsPure:  false,
		Effect:  "Rand",
		Type:    makeUuid4Type,
		Impl:    uuid4Impl,
		Metadata: &BuiltinMetadata{
			Description: "Generate a random UUID v4",
			Returns:     "string - RFC 4122 v4 UUID",
			Since:       "v0.9.4",
			Stability:   StabilityStable,
			Tags:        []string{"uuid", "random", "id"},
			Category:    "rand",
		},
	})
	if err != nil {
		panic("failed to register _uuid4: " + err.Error())
	}
}

func makeUuid4Type() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Unit()).Returns(T.String()).Effects("Rand")
}

func uuid4Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("Rand", ""); err != nil {
		return nil, err
	}

	var uuid [16]byte
	if _, err := io.ReadFull(cryptorand.Reader, uuid[:]); err != nil {
		return nil, fmt.Errorf("_uuid4: failed to generate random bytes: %w", err)
	}
	// Set version (4) and variant (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	result := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
	return &eval.StringValue{Value: result}, nil
}
