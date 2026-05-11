package main

import (
	"context"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	eval_harness "github.com/sunholo-data/ailang/internal/eval_harness"
)

// TestSetupAIHandlerFromConfig_OpenAIKeyRelaxation verifies the 4-case
// matrix for M-AI-OPENAI-LOCAL-ENDPOINT-RELAX in setupAIHandlerFromConfig.
func TestSetupAIHandlerFromConfig_OpenAIKeyRelaxation(t *testing.T) {
	model := &eval_harness.ModelConfig{
		APIName:  "gpt-5",
		Provider: "openai",
		EnvVar:   "OPENAI_API_KEY",
	}

	cases := []struct {
		name        string
		apiKey      string
		baseURL     string
		wantErr     bool
		errContains string
	}{
		{
			name:    "key set, no custom URL — creates handler",
			apiKey:  "sk-test",
			baseURL: "",
			wantErr: false,
		},
		{
			name:        "no key, no custom URL — returns error with hint",
			apiKey:      "",
			baseURL:     "",
			wantErr:     true,
			errContains: "OPENAI_BASE_URL",
		},
		{
			name:    "no key, custom URL set — creates handler (local endpoint)",
			apiKey:  "",
			baseURL: "http://localhost:8000/v1",
			wantErr: false,
		},
		{
			name:    "key set, custom URL set — creates handler",
			apiKey:  "sk-test",
			baseURL: "http://localhost:8000/v1",
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_KEY", tc.apiKey)
			t.Setenv("OPENAI_BASE_URL", tc.baseURL)

			effCtx := effects.NewEffContext([]string{})
			err := setupAIHandlerFromConfig(effCtx, model, "gpt-5", nil, nil)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q missing expected hint %q", err.Error(), tc.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				if effCtx.AI == nil {
					t.Error("expected AI handler to be installed on effCtx")
				}
			}
		})
	}
}

// TestExecuteAPI_OpenAIKeyRelaxation verifies the 4-case matrix for
// M-AI-OPENAI-LOCAL-ENDPOINT-RELAX in executeAPI.
func TestExecuteAPI_OpenAIKeyRelaxation(t *testing.T) {
	cases := []struct {
		name        string
		apiKey      string
		baseURL     string
		wantKeyErr  bool
		errContains string
	}{
		{
			name:       "key set, no custom URL — passes key guard",
			apiKey:     "sk-test",
			baseURL:    "",
			wantKeyErr: false,
		},
		{
			name:        "no key, no custom URL — key guard fires with hint",
			apiKey:      "",
			baseURL:     "",
			wantKeyErr:  true,
			errContains: "OPENAI_BASE_URL",
		},
		{
			name:       "no key, custom URL set — passes key guard (local endpoint)",
			apiKey:     "",
			baseURL:    "http://localhost:8000/v1",
			wantKeyErr: false,
		},
		{
			name:       "key set, custom URL set — passes key guard",
			apiKey:     "sk-test",
			baseURL:    "http://localhost:8000/v1",
			wantKeyErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_KEY", tc.apiKey)
			t.Setenv("OPENAI_BASE_URL", tc.baseURL)

			// executeAPI will attempt a real HTTP request; cancel immediately so
			// the only errors we see are (a) the key-guard return or (b) a
			// network/context error (not a key error).
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // already cancelled

			_, err := executeAPI(ctx, "openai", "test directive", "gpt-5", "", 0, false, nil)

			keyErrPhrases := []string{"OPENAI_API_KEY environment variable required"}

			if tc.wantKeyErr {
				if err == nil {
					t.Fatal("expected key-guard error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q missing expected hint %q", err.Error(), tc.errContains)
				}
				// Must be the key-guard error, not something else.
				foundKeyErr := false
				for _, phrase := range keyErrPhrases {
					if strings.Contains(err.Error(), phrase) {
						foundKeyErr = true
					}
				}
				if !foundKeyErr {
					t.Errorf("expected key-guard error, got different error: %v", err)
				}
			} else {
				// Key guard passed — any error must NOT be the key-guard error.
				if err != nil {
					for _, phrase := range keyErrPhrases {
						if strings.Contains(err.Error(), phrase) {
							t.Errorf("key guard fired when it should have passed: %v", err)
						}
					}
				}
			}
		})
	}
}

// Compile-time check: ai.ProviderOpenAI is used in both functions.
var _ = ai.ProviderOpenAI
