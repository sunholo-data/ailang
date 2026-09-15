package gemini

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// M-V1-SIMPLIFY-S4 M1: the Vertex AI location decides the regional endpoint,
// data residency and (per-region pricing) the bill. Unset, "global" is the
// deprecated default — served with a warning, refused under
// AILANG_STRICT_CONFIG=1; WithLocation and GOOGLE_CLOUD_LOCATION are never
// deprecated. The project is passed explicitly so no metadata or gcloud
// lookup runs.
func TestNewVertexAIClient_LocationIsTheDeprecatedDefaultThenStrict(t *testing.T) {
	t.Setenv(config.EnvStrict, "")
	t.Setenv(EnvVertexLocation, "")

	c, err := NewVertexAIClient("proj")
	if err != nil || c.location != deprecatedVertexLocation {
		t.Fatalf("unset: (%v, %v), want the deprecated %q served", c, err, deprecatedVertexLocation)
	}

	t.Setenv(config.EnvStrict, "1")
	if c, err = NewVertexAIClient("proj"); c != nil || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: (%v, %v), want config.ErrDeprecatedDefault (through ai.ProviderError's Unwrap)", c, err)
	}

	if c, err = NewVertexAIClient("proj", WithLocation("europe-west4")); err != nil || c.location != "europe-west4" {
		t.Fatalf("strict with WithLocation: (%v, %v)", c, err)
	}

	t.Setenv(EnvVertexLocation, "us-central1")
	if c, err = NewVertexAIClient("proj"); err != nil || c.location != "us-central1" {
		t.Fatalf("strict with %s set: (%v, %v)", EnvVertexLocation, c, err)
	}
	// An explicit option still outranks the environment.
	if c, err = NewVertexAIClient("proj", WithLocation("europe-west4")); err != nil || c.location != "europe-west4" {
		t.Fatalf("WithLocation must outrank %s: (%v, %v)", EnvVertexLocation, c, err)
	}
}

// The AI Studio client has no location and must not imply one.
func TestNewClient_NoImpliedLocation(t *testing.T) {
	if loc := NewClient("k").location; loc != "" {
		t.Fatalf("NewClient location = %q, want empty: the API-key endpoint is not regional", loc)
	}
}
