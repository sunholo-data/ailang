package builtins

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

func TestEmbeddingEncode(t *testing.T) {
	ctx := &effects.EffContext{}

	tests := []struct {
		name    string
		input   []float64
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty list",
			input:   []float64{},
			wantLen: 0,
		},
		{
			name:    "single float",
			input:   []float64{1.0},
			wantLen: 4,
		},
		{
			name:    "three floats",
			input:   []float64{1.0, 2.0, 3.0},
			wantLen: 12,
		},
		{
			name:    "standard MiniLM dimension",
			input:   make([]float64, 384),
			wantLen: 384 * 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create list of floats
			elements := make([]eval.Value, len(tt.input))
			for i, f := range tt.input {
				elements[i] = &eval.FloatValue{Value: f}
			}
			args := []eval.Value{&eval.ListValue{Elements: elements}}

			result, err := embeddingEncodeImpl(ctx, args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			bytesVal, ok := result.(*eval.BytesValue)
			if !ok {
				t.Fatalf("expected BytesValue, got %T", result)
			}

			if len(bytesVal.Value) != tt.wantLen {
				t.Errorf("expected %d bytes, got %d", tt.wantLen, len(bytesVal.Value))
			}
		})
	}
}

func TestEmbeddingDecode(t *testing.T) {
	ctx := &effects.EffContext{}

	tests := []struct {
		name       string
		inputBytes []byte
		wantNone   bool
		wantFloats []float64
	}{
		{
			name:       "empty bytes",
			inputBytes: []byte{},
			wantFloats: []float64{},
		},
		{
			name:       "invalid length (3 bytes)",
			inputBytes: []byte{1, 2, 3},
			wantNone:   true,
		},
		{
			name:       "invalid length (5 bytes)",
			inputBytes: []byte{1, 2, 3, 4, 5},
			wantNone:   true,
		},
		{
			name:       "valid 4 bytes (1 float)",
			inputBytes: float32ToBytes(1.0),
			wantFloats: []float64{1.0},
		},
		{
			name:       "valid 8 bytes (2 floats)",
			inputBytes: append(float32ToBytes(1.0), float32ToBytes(2.0)...),
			wantFloats: []float64{1.0, 2.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{&eval.BytesValue{Value: tt.inputBytes}}

			result, err := embeddingDecodeImpl(ctx, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			tagged, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("expected TaggedValue, got %T", result)
			}

			if tt.wantNone {
				if tagged.CtorName != "None" {
					t.Errorf("expected None, got %s", tagged.CtorName)
				}
				return
			}

			if tagged.CtorName != "Some" {
				t.Errorf("expected Some, got %s", tagged.CtorName)
				return
			}

			if len(tagged.Fields) != 1 {
				t.Fatalf("expected 1 field, got %d", len(tagged.Fields))
			}

			listVal, ok := tagged.Fields[0].(*eval.ListValue)
			if !ok {
				t.Fatalf("expected ListValue, got %T", tagged.Fields[0])
			}

			if len(listVal.Elements) != len(tt.wantFloats) {
				t.Errorf("expected %d floats, got %d", len(tt.wantFloats), len(listVal.Elements))
				return
			}

			for i, elem := range listVal.Elements {
				fv, ok := elem.(*eval.FloatValue)
				if !ok {
					t.Errorf("element %d: expected FloatValue, got %T", i, elem)
					continue
				}
				if fv.Value != tt.wantFloats[i] {
					t.Errorf("element %d: expected %f, got %f", i, tt.wantFloats[i], fv.Value)
				}
			}
		})
	}
}

func TestEmbeddingRoundTrip(t *testing.T) {
	ctx := &effects.EffContext{}

	testCases := [][]float64{
		{1.0, 2.0, 3.0},
		{0.0},
		{-1.5, 0.0, 1.5},
		{math.Pi, math.E},
		{1e-10, 1e10},
	}

	for _, original := range testCases {
		// Encode
		elements := make([]eval.Value, len(original))
		for i, f := range original {
			elements[i] = &eval.FloatValue{Value: f}
		}
		encodeArgs := []eval.Value{&eval.ListValue{Elements: elements}}

		encoded, err := embeddingEncodeImpl(ctx, encodeArgs)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		// Decode
		decodeArgs := []eval.Value{encoded}
		decoded, err := embeddingDecodeImpl(ctx, decodeArgs)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		// Check Some
		tagged := decoded.(*eval.TaggedValue)
		if tagged.CtorName != "Some" {
			t.Fatal("expected Some")
		}

		// Compare
		listVal := tagged.Fields[0].(*eval.ListValue)
		if len(listVal.Elements) != len(original) {
			t.Errorf("length mismatch: want %d, got %d", len(original), len(listVal.Elements))
			continue
		}

		for i, elem := range listVal.Elements {
			fv := elem.(*eval.FloatValue)
			// Note: float64 -> float32 -> float64 may lose precision
			// Use float32 comparison
			origF32 := float32(original[i])
			gotF32 := float32(fv.Value)
			if origF32 != gotF32 {
				t.Errorf("element %d: want %f, got %f", i, origF32, gotF32)
			}
		}
	}
}

func TestEmbeddingStandardDimensions(t *testing.T) {
	ctx := &effects.EffContext{}

	dimensions := []int{384, 768, 1536} // MiniLM, BERT, OpenAI

	for _, dim := range dimensions {
		t.Run(string(rune('0'+dim)), func(t *testing.T) {
			// Create random-ish embedding
			elements := make([]eval.Value, dim)
			for i := 0; i < dim; i++ {
				elements[i] = &eval.FloatValue{Value: float64(i) / float64(dim)}
			}
			args := []eval.Value{&eval.ListValue{Elements: elements}}

			// Encode
			result, err := embeddingEncodeImpl(ctx, args)
			if err != nil {
				t.Fatalf("encode failed: %v", err)
			}

			bytesVal := result.(*eval.BytesValue)
			expectedLen := dim * 4
			if len(bytesVal.Value) != expectedLen {
				t.Errorf("expected %d bytes, got %d", expectedLen, len(bytesVal.Value))
			}

			// Decode and verify
			decodeArgs := []eval.Value{result}
			decoded, err := embeddingDecodeImpl(ctx, decodeArgs)
			if err != nil {
				t.Fatalf("decode failed: %v", err)
			}

			tagged := decoded.(*eval.TaggedValue)
			if tagged.CtorName != "Some" {
				t.Fatal("expected Some")
			}

			listVal := tagged.Fields[0].(*eval.ListValue)
			if len(listVal.Elements) != dim {
				t.Errorf("expected %d elements, got %d", dim, len(listVal.Elements))
			}
		})
	}
}

// Helper to create float32 bytes
func float32ToBytes(f float64) []byte {
	buf := make([]byte, 4)
	bits := math.Float32bits(float32(f))
	binary.LittleEndian.PutUint32(buf, bits)
	return buf
}
