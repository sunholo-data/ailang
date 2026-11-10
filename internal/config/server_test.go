package config

import "testing"

// AILANG_INGEST_AUTH is a security gate: unset is the registered default
// (warn), and anything unrecognised — a typo included — enforces.
func TestIngestAuthMode_UnknownEnforces(t *testing.T) {
	for in, want := range map[string]string{
		"": IngestAuthWarn, "warn": IngestAuthWarn, " Warn ": IngestAuthWarn,
		"off": IngestAuthOff, "OFF": IngestAuthOff,
		"enforce": IngestAuthEnforce, "enforced": IngestAuthEnforce, "true": IngestAuthEnforce,
	} {
		t.Setenv(EnvIngestAuth, in)
		if got := IngestAuthMode(); got != want {
			t.Errorf("%s=%q: got %q, want %q", EnvIngestAuth, in, got, want)
		}
	}
}

// The ingest secrets are trimmed, so a Secret Manager value with a trailing
// newline still matches what a sender presents.
func TestIngestTokens_Trimmed(t *testing.T) {
	t.Setenv(EnvHubToken, " hub-secret\n")
	t.Setenv(EnvOTLPIngestToken, "otlp-secret\n")
	if got := HubToken(); got != "hub-secret" {
		t.Errorf("HubToken() = %q, want trimmed", got)
	}
	if got := OTLPIngestToken(); got != "otlp-secret" {
		t.Errorf("OTLPIngestToken() = %q, want trimmed", got)
	}
}
