package runner

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// TestSetupNetHandler_Parity ensures Net effect gets the same security
// configuration options as Stream. Regression test: --net-allow-http flag
// was documented in error messages but never wired up, causing httpRequest
// to silently reject all http:// URLs (including GCP metadata server).
func TestSetupNetHandler(t *testing.T) {
	tests := []struct {
		name           string
		caps           string
		allowHTTP      bool
		allowLocalhost bool
		allowMetadata  bool
		allowDomains   string
		wantHTTP       bool
		wantLocalhost  bool
		wantMetadata   bool
		wantDomains    int
	}{
		{
			name:      "no Net cap — defaults unchanged",
			caps:      "IO",
			allowHTTP: true,
			wantHTTP:  false, // Net not granted, so AllowHTTP stays default
		},
		{
			name:      "Net cap with AllowHTTP",
			caps:      "Net",
			allowHTTP: true,
			wantHTTP:  true,
		},
		{
			name:           "Net cap with AllowLocalhost",
			caps:           "Net",
			allowLocalhost: true,
			wantLocalhost:  true,
		},
		{
			name:          "Net cap with AllowMetadata",
			caps:          "Net",
			allowMetadata: true,
			wantMetadata:  true,
		},
		{
			name:         "Net cap with domain allowlist",
			caps:         "Net",
			allowDomains: "metadata.google.internal,example.com",
			wantDomains:  2,
		},
		{
			name:     "Net cap defaults — http blocked",
			caps:     "Net",
			wantHTTP: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effCtx := effects.NewEffContext([]string{})
			if err := GrantCapabilities(effCtx, tt.caps); err != nil {
				t.Fatal(err)
			}
			if err := SetupNetHandler(effCtx, tt.allowHTTP, tt.allowDomains, tt.allowLocalhost, tt.allowMetadata, ""); err != nil {
				t.Fatal(err)
			}

			if effCtx.Net.AllowHTTP != tt.wantHTTP {
				t.Errorf("AllowHTTP = %v, want %v", effCtx.Net.AllowHTTP, tt.wantHTTP)
			}
			if effCtx.Net.AllowLocalhost != tt.wantLocalhost {
				t.Errorf("AllowLocalhost = %v, want %v", effCtx.Net.AllowLocalhost, tt.wantLocalhost)
			}
			if effCtx.Net.AllowMetadata != tt.wantMetadata {
				t.Errorf("AllowMetadata = %v, want %v", effCtx.Net.AllowMetadata, tt.wantMetadata)
			}
			if tt.wantDomains > 0 && len(effCtx.Net.AllowedDomains) != tt.wantDomains {
				t.Errorf("AllowedDomains len = %d, want %d", len(effCtx.Net.AllowedDomains), tt.wantDomains)
			}
		})
	}
}

func streamCtx(caps ...string) *effects.EffContext {
	ctx := effects.NewEffContext(nil)
	for _, c := range caps {
		ctx.Grant(effects.NewCapability(c))
	}
	return ctx
}

func TestSetupStreamHandler_MaxMessage(t *testing.T) {
	cases := []struct {
		flag    string
		want    int64
		wantErr string
	}{
		{"", 1 << 20, ""}, // default
		{"256KB", 256 << 10, ""},
		{"8MB", 8 << 20, ""},
		{"65536", 65536, ""},
		{"0", 0, "must be positive"},
		{"lots", 0, "--stream-max-message"},
		{"-1", 0, "--stream-max-message"},
	}
	for _, c := range cases {
		ctx := streamCtx("Stream")
		err := SetupStreamHandler(ctx, StreamOptions{MaxMessage: c.flag})
		if c.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%q: err = %v, want containing %q", c.flag, err, c.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", c.flag, err)
		}
		if ctx.Stream.MaxMessageSize != c.want {
			t.Errorf("%q: MaxMessageSize = %d, want %d", c.flag, ctx.Stream.MaxMessageSize, c.want)
		}
	}
}

// Setting the cap without the capability is an error, not an ignored flag.
func TestSetupStreamHandler_MaxMessageNeedsStreamCap(t *testing.T) {
	err := SetupStreamHandler(streamCtx("IO"), StreamOptions{MaxMessage: "4MB"})
	if err == nil || !strings.Contains(err.Error(), "needs --caps Stream") {
		t.Errorf("err = %v, want 'needs --caps Stream'", err)
	}
	if err := SetupStreamHandler(streamCtx("IO"), StreamOptions{}); err != nil {
		t.Errorf("no Stream cap, no flag: %v", err)
	}
}
