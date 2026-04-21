package testing

import (
	"math/rand"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// Generator produces random values of a specific type for property-based testing.
type Generator interface {
	// Generate produces a random value using the provided RNG.
	Generate(rng *rand.Rand) eval.Value
}

// GenConfig configures generation parameters.
type GenConfig struct {
	Seed     int64 // Seed for deterministic generation
	MaxSize  int   // Maximum size for collections (lists, strings)
	MinInt   int   // Minimum int value
	MaxInt   int   // Maximum int value
	MinFloat float64
	MaxFloat float64
}

// DefaultConfig returns default generation configuration.
func DefaultConfig() GenConfig {
	return GenConfig{
		Seed:     0, // Will use time-based if 0
		MaxSize:  100,
		MinInt:   -1000,
		MaxInt:   1000,
		MinFloat: -1000.0,
		MaxFloat: 1000.0,
	}
}

// IntGenerator generates random integers.
type IntGenerator struct {
	min int
	max int
}

// NewIntGenerator creates a new int generator with specified range.
func NewIntGenerator(min, max int) *IntGenerator {
	return &IntGenerator{min: min, max: max}
}

// Generate produces a random int value.
func (g *IntGenerator) Generate(rng *rand.Rand) eval.Value {
	val := g.min + rng.Intn(g.max-g.min+1)
	return &eval.IntValue{Value: val}
}

// FloatGenerator generates random floats.
type FloatGenerator struct {
	min float64
	max float64
}

// NewFloatGenerator creates a new float generator with specified range.
func NewFloatGenerator(min, max float64) *FloatGenerator {
	return &FloatGenerator{min: min, max: max}
}

// Generate produces a random float value.
func (g *FloatGenerator) Generate(rng *rand.Rand) eval.Value {
	val := g.min + rng.Float64()*(g.max-g.min)
	return &eval.FloatValue{Value: val}
}

// BoolGenerator generates random booleans.
type BoolGenerator struct{}

// NewBoolGenerator creates a new bool generator.
func NewBoolGenerator() *BoolGenerator {
	return &BoolGenerator{}
}

// Generate produces a random bool value.
func (g *BoolGenerator) Generate(rng *rand.Rand) eval.Value {
	return &eval.BoolValue{Value: rng.Intn(2) == 1}
}

// StringGenerator generates random strings.
type StringGenerator struct {
	minLen  int
	maxLen  int
	charset string
}

// NewStringGenerator creates a new string generator with specified length range.
// If charset is empty, uses alphanumeric characters.
func NewStringGenerator(minLen, maxLen int, charset string) *StringGenerator {
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	return &StringGenerator{
		minLen:  minLen,
		maxLen:  maxLen,
		charset: charset,
	}
}

// Generate produces a random string value.
func (g *StringGenerator) Generate(rng *rand.Rand) eval.Value {
	length := g.minLen
	if g.maxLen > g.minLen {
		length = g.minLen + rng.Intn(g.maxLen-g.minLen+1)
	}

	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		idx := rng.Intn(len(g.charset))
		builder.WriteByte(g.charset[idx])
	}

	return &eval.StringValue{Value: builder.String()}
}

// ListGenerator generates random lists using an element generator.
type ListGenerator struct {
	elemGen Generator
	minLen  int
	maxLen  int
}

// NewListGenerator creates a new list generator.
func NewListGenerator(elemGen Generator, minLen, maxLen int) *ListGenerator {
	return &ListGenerator{
		elemGen: elemGen,
		minLen:  minLen,
		maxLen:  maxLen,
	}
}

// Generate produces a random list value.
func (g *ListGenerator) Generate(rng *rand.Rand) eval.Value {
	length := g.minLen
	if g.maxLen > g.minLen {
		length = g.minLen + rng.Intn(g.maxLen-g.minLen+1)
	}

	elements := make([]eval.Value, length)
	for i := 0; i < length; i++ {
		elements[i] = g.elemGen.Generate(rng)
	}

	return &eval.ListValue{Elements: elements}
}

// PropertyRunner runs property-based tests with generators.
type PropertyRunner struct {
	config   GenConfig
	rng      *rand.Rand
	testRuns int // Number of test cases to generate per property
}

// NewPropertyRunner creates a new property runner with config.
func NewPropertyRunner(config GenConfig, testRuns int) *PropertyRunner {
	seed := config.Seed
	if seed == 0 {
		seed = rand.Int63() // Use random seed if not specified
	}

	return &PropertyRunner{
		config:   config,
		rng:      rand.New(rand.NewSource(seed)),
		testRuns: testRuns,
	}
}

// GetRNG returns the random number generator for this runner.
// Useful for custom generators that need direct RNG access.
func (pr *PropertyRunner) GetRNG() *rand.Rand {
	return pr.rng
}

// GetConfig returns the generation configuration.
func (pr *PropertyRunner) GetConfig() GenConfig {
	return pr.config
}

// GenerateIntInRange generates a random int in the configured range.
func (pr *PropertyRunner) GenerateIntInRange() int {
	return pr.config.MinInt + pr.rng.Intn(pr.config.MaxInt-pr.config.MinInt+1)
}

// GenerateFloatInRange generates a random float in the configured range.
func (pr *PropertyRunner) GenerateFloatInRange() float64 {
	return pr.config.MinFloat + pr.rng.Float64()*(pr.config.MaxFloat-pr.config.MinFloat)
}

// GenerateBool generates a random boolean.
func (pr *PropertyRunner) GenerateBool() bool {
	return pr.rng.Intn(2) == 1
}

// GenerateString generates a random string with length up to MaxSize.
func (pr *PropertyRunner) GenerateString() string {
	length := pr.rng.Intn(pr.config.MaxSize + 1)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		idx := pr.rng.Intn(len(charset))
		builder.WriteByte(charset[idx])
	}

	return builder.String()
}

// GenerateList generates a random list using the provided element generator.
func (pr *PropertyRunner) GenerateList(elemGen Generator, maxLen int) []eval.Value {
	length := pr.rng.Intn(maxLen + 1)
	elements := make([]eval.Value, length)

	for i := 0; i < length; i++ {
		elements[i] = elemGen.Generate(pr.rng)
	}

	return elements
}

// ShrinkValue attempts to shrink a failing value to find a minimal counterexample.
// It uses the provided shrinker and predicate function.
// Returns the minimal shrunk value that still fails, or the original if no smaller failure found.
func (pr *PropertyRunner) ShrinkValue(
	original eval.Value,
	shrinker Shrinker,
	predicate func(eval.Value) bool,
) eval.Value {
	if shrinker == nil {
		return original // Can't shrink without a shrinker
	}

	// Start with original failing value
	current := original
	improved := true

	// Keep shrinking while we find smaller failures
	// Limit iterations to prevent infinite loops (max 100 shrink attempts)
	for iteration := 0; iteration < 100 && improved; iteration++ {
		improved = false
		shrinks := shrinker.Shrink(current)

		// Try each shrunk value in order (shrinkers return them from simplest to complex)
		for _, candidate := range shrinks {
			// Check if this shrunk value still fails the property
			if !predicate(candidate) {
				// Found a smaller failure!
				current = candidate
				improved = true
				break // Take first (simplest) failing shrink
			}
		}
	}

	return current
}
