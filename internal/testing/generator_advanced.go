package testing

import (
	"math/rand"

	"github.com/sunholo-data/ailang/internal/eval"
)

// MapGenerator transforms generated values using a mapping function.
type MapGenerator struct {
	source Generator
	mapFn  func(eval.Value) eval.Value
}

// NewMapGenerator creates a generator that transforms values from source.
func NewMapGenerator(source Generator, mapFn func(eval.Value) eval.Value) *MapGenerator {
	return &MapGenerator{
		source: source,
		mapFn:  mapFn,
	}
}

// Generate produces a transformed value.
func (g *MapGenerator) Generate(rng *rand.Rand) eval.Value {
	val := g.source.Generate(rng)
	return g.mapFn(val)
}

// FilterGenerator generates values that satisfy a predicate.
// Retries up to maxRetries times before giving up.
type FilterGenerator struct {
	source     Generator
	predicate  func(eval.Value) bool
	maxRetries int
}

// NewFilterGenerator creates a generator that filters values.
// maxRetries specifies how many attempts before giving up (0 = unlimited, dangerous!)
func NewFilterGenerator(source Generator, predicate func(eval.Value) bool, maxRetries int) *FilterGenerator {
	if maxRetries == 0 {
		maxRetries = 100 // Default safety limit
	}
	return &FilterGenerator{
		source:     source,
		predicate:  predicate,
		maxRetries: maxRetries,
	}
}

// Generate produces a value that satisfies the predicate.
func (g *FilterGenerator) Generate(rng *rand.Rand) eval.Value {
	for i := 0; i < g.maxRetries; i++ {
		val := g.source.Generate(rng)
		if g.predicate(val) {
			return val
		}
	}
	// If we can't find a valid value, return the last attempt
	// This prevents infinite loops but may return invalid data
	return g.source.Generate(rng)
}

// OneOfGenerator randomly chooses from multiple generators.
type OneOfGenerator struct {
	generators []Generator
}

// NewOneOfGenerator creates a generator that randomly picks from options.
func NewOneOfGenerator(generators ...Generator) *OneOfGenerator {
	return &OneOfGenerator{
		generators: generators,
	}
}

// Generate produces a value from a randomly chosen generator.
func (g *OneOfGenerator) Generate(rng *rand.Rand) eval.Value {
	idx := rng.Intn(len(g.generators))
	return g.generators[idx].Generate(rng)
}

// FrequencyGenerator chooses generators with weighted probability.
type FrequencyGenerator struct {
	weights     []int
	generators  []Generator
	totalWeight int
}

// NewFrequencyGenerator creates a generator with weighted choice.
// weights and generators must have the same length.
func NewFrequencyGenerator(weights []int, generators []Generator) *FrequencyGenerator {
	total := 0
	for _, w := range weights {
		total += w
	}
	return &FrequencyGenerator{
		weights:     weights,
		generators:  generators,
		totalWeight: total,
	}
}

// Generate produces a value from a weighted random generator.
func (g *FrequencyGenerator) Generate(rng *rand.Rand) eval.Value {
	r := rng.Intn(g.totalWeight)
	cumulative := 0

	for i, weight := range g.weights {
		cumulative += weight
		if r < cumulative {
			return g.generators[i].Generate(rng)
		}
	}

	// Fallback (should never reach here)
	return g.generators[len(g.generators)-1].Generate(rng)
}

// SizedGenerator adjusts size based on a size parameter.
type SizedGenerator struct {
	createFn func(size int) Generator
	size     int
}

// NewSizedGenerator creates a size-aware generator.
func NewSizedGenerator(size int, createFn func(size int) Generator) *SizedGenerator {
	return &SizedGenerator{
		createFn: createFn,
		size:     size,
	}
}

// Generate produces a value with the configured size.
func (g *SizedGenerator) Generate(rng *rand.Rand) eval.Value {
	gen := g.createFn(g.size)
	return gen.Generate(rng)
}

// WithSize returns a new generator with a different size.
func (g *SizedGenerator) WithSize(size int) *SizedGenerator {
	return &SizedGenerator{
		createFn: g.createFn,
		size:     size,
	}
}

// ADTGenerator generates algebraic data type values (tagged unions).
type ADTGenerator struct {
	tag          string
	fieldGens    []Generator
	constructTag bool // If true, wrap in TaggedValue
}

// NewADTGenerator creates a generator for ADT values.
// If constructTag is true, wraps result in TaggedValue.
func NewADTGenerator(tag string, fieldGens []Generator, constructTag bool) *ADTGenerator {
	return &ADTGenerator{
		tag:          tag,
		fieldGens:    fieldGens,
		constructTag: constructTag,
	}
}

// Generate produces an ADT value.
func (g *ADTGenerator) Generate(rng *rand.Rand) eval.Value {
	// Generate field values
	fields := make([]eval.Value, len(g.fieldGens))
	for i, fieldGen := range g.fieldGens {
		fields[i] = fieldGen.Generate(rng)
	}

	if g.constructTag {
		return &eval.TaggedValue{
			ModulePath: "test", // Test module
			TypeName:   "ADT",  // Generic ADT type for testing
			CtorName:   g.tag,  // Constructor name (e.g., "Some", "None")
			Fields:     fields, // Constructor fields
		}
	}

	// If not constructing tagged value, return fields directly
	// (useful for testing payload generation separately)
	if len(fields) == 0 {
		return &eval.UnitValue{} // Unit for nullary constructors
	} else if len(fields) == 1 {
		return fields[0]
	} else {
		return &eval.TupleValue{Elements: fields}
	}
}

// RecordGenerator generates record values (field: value maps).
type RecordGenerator struct {
	fieldGens map[string]Generator
}

// NewRecordGenerator creates a generator for record values.
func NewRecordGenerator(fieldGens map[string]Generator) *RecordGenerator {
	return &RecordGenerator{
		fieldGens: fieldGens,
	}
}

// Generate produces a record value.
func (g *RecordGenerator) Generate(rng *rand.Rand) eval.Value {
	fields := make(map[string]eval.Value)

	for fieldName, fieldGen := range g.fieldGens {
		fields[fieldName] = fieldGen.Generate(rng)
	}

	return &eval.RecordValue{Fields: fields}
}

// TupleGenerator generates tuple values.
type TupleGenerator struct {
	elemGens []Generator
}

// NewTupleGenerator creates a generator for tuple values.
func NewTupleGenerator(elemGens []Generator) *TupleGenerator {
	return &TupleGenerator{
		elemGens: elemGens,
	}
}

// Generate produces a tuple value.
func (g *TupleGenerator) Generate(rng *rand.Rand) eval.Value {
	elements := make([]eval.Value, len(g.elemGens))

	for i, elemGen := range g.elemGens {
		elements[i] = elemGen.Generate(rng)
	}

	return &eval.TupleValue{Elements: elements}
}

// Common ADT Generators

// OptionGenerator generates Option type values (Some(x) | None).
func OptionGenerator(valueGen Generator) Generator {
	return NewOneOfGenerator(
		NewADTGenerator("None", []Generator{}, true),
		NewADTGenerator("Some", []Generator{valueGen}, true),
	)
}

// ResultGenerator generates Result type values (Ok(x) | Err(e)).
func ResultGenerator(okGen, errGen Generator) Generator {
	return NewOneOfGenerator(
		NewADTGenerator("Ok", []Generator{okGen}, true),
		NewADTGenerator("Err", []Generator{errGen}, true),
	)
}

// ConstantGenerator always returns the same value.
type ConstantGenerator struct {
	value eval.Value
}

// NewConstantGenerator creates a generator that always returns the same value.
func NewConstantGenerator(value eval.Value) *ConstantGenerator {
	return &ConstantGenerator{value: value}
}

// Generate returns the constant value.
func (g *ConstantGenerator) Generate(rng *rand.Rand) eval.Value {
	return g.value
}
