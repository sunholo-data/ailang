package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// format keeps the verb out of a string literal on purpose. These assertions
// exist to prove that each fmt verb redacts, so staticcheck must not be allowed
// to rewrite them into direct String() calls — that would delete the test.
func format(verb string, value any) string { return fmt.Sprintf(verb, value) }

func testProtector(t *testing.T) *StaticKeyProtector {
	t.Helper()
	protector, err := NewRandomStaticKeyProtector("test-kek")
	if err != nil {
		t.Fatal(err)
	}
	return protector
}

func TestSealOpenRoundTrip(t *testing.T) {
	protector := testProtector(t)
	plaintext := []byte(`{"cookies":[{"name":"sid","value":"super-secret-session"}]}`)

	sealed, err := Seal(context.Background(), protector, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if sealed.Empty() {
		t.Fatal("sealed envelope reports empty")
	}
	opened, err := Open(context.Background(), protector, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("round trip mismatch: got %q", opened)
	}
}

// Criterion 1 (first half): every seal draws a fresh nonce. A repeated nonce
// under the same key is a total confidentiality break for GCM, so this asserts
// uniqueness across many seals of identical plaintext rather than just two.
func TestSealUsesFreshNoncePerBlob(t *testing.T) {
	protector := testProtector(t)
	plaintext := []byte("identical plaintext every time")

	const seals = 64
	nonces := make(map[string]bool, seals)
	ciphertexts := make(map[string]bool, seals)
	for i := 0; i < seals; i++ {
		sealed, err := Seal(context.Background(), protector, plaintext)
		if err != nil {
			t.Fatal(err)
		}
		nonce := string(sealed.Nonce())
		if nonce == "" {
			t.Fatal("sealed envelope has no nonce")
		}
		if nonces[nonce] {
			t.Fatalf("nonce reused after %d seals", i)
		}
		nonces[nonce] = true

		raw := string(sealed.Bytes())
		if ciphertexts[raw] {
			t.Fatalf("identical sealed bytes after %d seals; nonce is not varying", i)
		}
		ciphertexts[raw] = true
	}
}

// Criterion 1 (second half): a tampered ciphertext must fail closed with
// FailureMaterializeFailed and return NO plaintext at all.
func TestOpenTamperedCiphertextFailsClosed(t *testing.T) {
	protector := testProtector(t)
	plaintext := []byte("canonical storage state with a recognizable marker")

	sealed, err := Seal(context.Background(), protector, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	raw := sealed.Bytes()

	// Flip a bit in every byte position in turn; the last region is ciphertext
	// and every single-bit corruption anywhere must fail closed.
	for offset := 0; offset < len(raw); offset++ {
		corrupted := append([]byte(nil), raw...)
		corrupted[offset] ^= 0x01

		parsed, parseErr := ParseSealedEnvelope(corrupted)
		if parseErr != nil {
			// A corrupted header may fail at parse; that is also failing closed,
			// but it must still be the materialize category.
			if !IsFailure(parseErr, FailureMaterializeFailed) {
				t.Fatalf("offset %d: parse error = %v, want materialize failure", offset, parseErr)
			}
			continue
		}
		opened, openErr := Open(context.Background(), protector, parsed)
		if openErr == nil {
			if bytes.Equal(opened, plaintext) {
				// Byte flipped in a region that is not authenticated at all.
				t.Fatalf("offset %d: corruption did not change the plaintext; envelope is not fully authenticated", offset)
			}
			t.Fatalf("offset %d: tampered envelope opened successfully", offset)
		}
		if !IsFailure(openErr, FailureMaterializeFailed) {
			t.Fatalf("offset %d: error = %v, want %s", offset, openErr, FailureMaterializeFailed)
		}
		if opened != nil {
			t.Fatalf("offset %d: failed open returned %d bytes; must be a total failure, never a partial read", offset, len(opened))
		}
	}
}

func TestOpenWithWrongProtectorFailsClosed(t *testing.T) {
	sealed, err := Seal(context.Background(), testProtector(t), []byte("state"))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), testProtector(t), sealed)
	if !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
	if opened != nil {
		t.Fatalf("wrong-key open returned %d bytes", len(opened))
	}
}

func TestSealedEnvelopeRedactsEverywhere(t *testing.T) {
	const marker = "SUPER-SECRET-COOKIE-VALUE"
	sealed, err := Seal(context.Background(), testProtector(t), []byte(marker))
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(sealed)
	if err != nil {
		t.Fatal(err)
	}
	presentations := []string{
		sealed.String(),
		sealed.GoString(),
		sealed.Error(),
		format("%v", sealed),
		format("%s", sealed),
		format("%#v", sealed),
		format("%+v", sealed),
		string(encoded),
	}
	for i, presentation := range presentations {
		if !strings.Contains(presentation, Redacted) {
			t.Fatalf("presentation %d does not redact: %s", i, presentation)
		}
		// The base64/binary of the ciphertext must not leak either.
		if strings.Contains(presentation, string(sealed.rawCiphertext())) {
			t.Fatalf("presentation %d leaked ciphertext: %s", i, presentation)
		}
	}
}

func TestSealedEnvelopeSurvivesEncodeDecode(t *testing.T) {
	protector := testProtector(t)
	plaintext := []byte("durable canonical material")
	sealed, err := Seal(context.Background(), protector, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	reparsed, err := ParseSealedEnvelope(sealed.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.KeyID() != sealed.KeyID() {
		t.Fatalf("key id = %q, want %q", reparsed.KeyID(), sealed.KeyID())
	}
	opened, err := Open(context.Background(), protector, reparsed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatalf("decoded envelope opened to %q", opened)
	}
}

func TestParseSealedEnvelopeRejectsGarbage(t *testing.T) {
	cases := map[string][]byte{
		"empty":         {},
		"short":         []byte("AIL"),
		"bad magic":     bytes.Repeat([]byte{0x7f}, 64),
		"truncated":     append([]byte(envelopeMagic), 0x01),
		"not an envelo": []byte(`{"looks":"like json"}`),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseSealedEnvelope(raw); !IsFailure(err, FailureMaterializeFailed) {
				t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
			}
		})
	}
}

func TestSealRejectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Seal(ctx, testProtector(t), []byte("x")); !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
}

func TestOpenPropagatesProtectorFailureAsMaterializeFailed(t *testing.T) {
	protector := testProtector(t)
	sealed, err := Seal(context.Background(), protector, []byte("state"))
	if err != nil {
		t.Fatal(err)
	}
	failing := failingProtector{}
	opened, err := Open(context.Background(), failing, sealed)
	if !IsFailure(err, FailureMaterializeFailed) {
		t.Fatalf("error = %v, want %s", err, FailureMaterializeFailed)
	}
	if opened != nil {
		t.Fatalf("failed unwrap returned %d bytes", len(opened))
	}
	// The protector's raw error text must not reach the public error.
	if strings.Contains(err.Error(), protectorSecretText) {
		t.Fatalf("public error leaked protector diagnostic: %v", err)
	}
}

const protectorSecretText = "kms endpoint https://internal.example/keys/abc"

type failingProtector struct{}

func (failingProtector) WrapKey(context.Context, []byte) ([]byte, string, error) {
	return nil, "", fmt.Errorf("%s", protectorSecretText)
}

func (failingProtector) UnwrapKey(context.Context, []byte, string) ([]byte, error) {
	return nil, fmt.Errorf("%s", protectorSecretText)
}

func TestLocalFileKeyProtectorCreatesAndReusesKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys", "kek.bin")

	first, err := NewLocalFileKeyProtector(path)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := Seal(context.Background(), first, []byte("state"))
	if err != nil {
		t.Fatal(err)
	}

	second, err := NewLocalFileKeyProtector(path)
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(context.Background(), second, sealed)
	if err != nil {
		t.Fatalf("reopened protector could not open envelope: %v", err)
	}
	if string(opened) != "state" {
		t.Fatalf("opened = %q", opened)
	}
	assertOwnerOnlyFile(t, path)
}

var _ KeyProtector = failingProtector{}
var _ KeyProtector = (*StaticKeyProtector)(nil)
