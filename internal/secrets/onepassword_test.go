package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		wantErr bool
	}{
		{"valid vault/item/field", "op://Prod/stripe/api-key", false},
		{"valid with section", "op://Prod/db/credentials/password", false},
		{"missing scheme", "Prod/stripe/api-key", true},
		{"wrong scheme", "vault://Prod/stripe/api-key", true},
		{"too few segments", "op://Prod/stripe", true},
		{"empty segment", "op://Prod//api-key", true},
		{"empty ref", "", true},
		{"only scheme", "op://", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateRef(%q) = nil, want error", tt.ref)
				}
				if !errors.Is(err, ErrInvalidRef) {
					t.Fatalf("ValidateRef(%q) error = %v, want ErrInvalidRef", tt.ref, err)
				}
			} else if err != nil {
				t.Fatalf("ValidateRef(%q) = %v, want nil", tt.ref, err)
			}
		})
	}
}

func TestRead_Success(t *testing.T) {
	r := &OnePasswordResolver{runner: func(_ context.Context, args ...string) ([]byte, error) {
		// op read <ref> — confirm we invoke the read subcommand with the ref.
		if len(args) < 2 || args[0] != "read" {
			t.Fatalf("unexpected op args: %v", args)
		}
		if args[len(args)-1] != "op://Prod/stripe/api-key" {
			t.Fatalf("ref not passed as last arg: %v", args)
		}
		return []byte("s3cr3t-value\n"), nil
	}}
	got, err := r.Read(context.Background(), "op://Prod/stripe/api-key")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if got != "s3cr3t-value" {
		t.Fatalf("Read = %q, want %q (trailing newline trimmed)", got, "s3cr3t-value")
	}
}

func TestRead_InvalidRefSkipsRunner(t *testing.T) {
	called := false
	r := &OnePasswordResolver{runner: func(_ context.Context, _ ...string) ([]byte, error) {
		called = true
		return nil, nil
	}}
	_, err := r.Read(context.Background(), "not-a-ref")
	if !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("Read error = %v, want ErrInvalidRef", err)
	}
	if called {
		t.Fatal("runner was called for an invalid ref; validation must gate the CLI call")
	}
}

func TestRead_RunnerFailureIsSecretUnavailable(t *testing.T) {
	r := &OnePasswordResolver{runner: func(_ context.Context, _ ...string) ([]byte, error) {
		return nil, errors.New("op: exit status 1: item not found")
	}}
	_, err := r.Read(context.Background(), "op://Prod/stripe/api-key")
	if !errors.Is(err, ErrSecretUnavailable) {
		t.Fatalf("Read error = %v, want ErrSecretUnavailable", err)
	}
}

// TestRead_ErrorNeverContainsValue: the failure path must not embed a resolved
// value (there is none on failure, but guard against future regressions where a
// caller might pass-through stdout into the error).
func TestRead_ErrorNeverContainsValue(t *testing.T) {
	const sentinel = "THE-ACTUAL-SECRET"
	r := &OnePasswordResolver{runner: func(_ context.Context, _ ...string) ([]byte, error) {
		// Even if a buggy backend returned the value alongside an error, Read
		// must not surface it.
		return []byte(sentinel), errors.New("op: exit status 1")
	}}
	_, err := r.Read(context.Background(), "op://Prod/stripe/api-key")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), sentinel) {
		t.Fatalf("error message leaked the secret value: %q", err.Error())
	}
}

func TestNewOnePasswordResolver_HasRunner(t *testing.T) {
	r := NewOnePasswordResolver()
	if r.runner == nil {
		t.Fatal("NewOnePasswordResolver returned a resolver with nil runner")
	}
}

// Compile-time check that OnePasswordResolver satisfies Resolver.
var _ Resolver = (*OnePasswordResolver)(nil)
