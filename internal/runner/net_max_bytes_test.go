package runner

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The Net body cap was a fixed 5 MB with no flag or variable, so a hosted
// service fetching user files by signed URL failed every file over 5 MiB
// whatever it advertised (docparse, 2026-11). --net-max-bytes and
// AILANG_NET_MAX_BYTES set it; the flag beats the variable.

const defaultNetMaxBytes = 5 * 1024 * 1024

func netLimitCtx(t *testing.T, caps ...string) *effects.EffContext {
	t.Helper()
	effCtx := effects.NewEffContext(nil)
	for _, c := range caps {
		effCtx.Grant(effects.NewCapability(c))
	}
	return effCtx
}

func TestSetupNetLimit_Resolution(t *testing.T) {
	cases := []struct {
		name, env, flag string
		want            int64
	}{
		{"unset keeps the 5MB default", "", "", defaultNetMaxBytes},
		{"env sets it", "50MB", "", 50 * 1024 * 1024},
		{"env plain bytes", "52428800", "", 52428800},
		{"flag sets it", "", "8M", 8 * 1024 * 1024},
		{"flag beats env", "50MB", "1KB", 1024},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(config.EnvNetMaxBytes, tc.env)
			effCtx := netLimitCtx(t, "Net")
			if err := SetupNetLimit(effCtx, tc.flag); err != nil {
				t.Fatal(err)
			}
			if effCtx.Net.MaxBytes != tc.want {
				t.Errorf("MaxBytes = %d, want %d", effCtx.Net.MaxBytes, tc.want)
			}
		})
	}
}

// A safety cap never falls back: zero, negative and malformed are errors
// naming the source, and the flag without --caps Net is refused rather than
// silently doing nothing.
func TestSetupNetLimit_RejectsBadValues(t *testing.T) {
	for _, bad := range []string{"0", "-1", "lots", "5XB", "0MB"} {
		t.Run("flag "+bad, func(t *testing.T) {
			t.Setenv(config.EnvNetMaxBytes, "")
			err := SetupNetLimit(netLimitCtx(t, "Net"), bad)
			if err == nil || !strings.Contains(err.Error(), "--net-max-bytes") {
				t.Errorf("--net-max-bytes %q: err = %v", bad, err)
			}
		})
		t.Run("env "+bad, func(t *testing.T) {
			t.Setenv(config.EnvNetMaxBytes, bad)
			err := SetupNetLimit(netLimitCtx(t, "Net"), "")
			if err == nil || !strings.Contains(err.Error(), config.EnvNetMaxBytes) {
				t.Errorf("%s=%q: err = %v", config.EnvNetMaxBytes, bad, err)
			}
		})
	}
	t.Setenv(config.EnvNetMaxBytes, "")
	if err := SetupNetLimit(netLimitCtx(t, "IO"), "50MB"); err == nil || !strings.Contains(err.Error(), "needs --caps Net") {
		t.Errorf("flag without Net cap: err = %v", err)
	}
}

// The configured cap reaches the wire in both directions: a response over it
// is Err(BodyTooLarge("<configured cap>")), and a 6 MiB response — over the
// old fixed 5 MB — is Ok once the cap is raised.
func TestSetupNetLimit_ReachesTheWire(t *testing.T) {
	sizes := map[string]int{"/small": 2048, "/big": 6 * 1024 * 1024}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", sizes[r.URL.Path])))
	}))
	defer server.Close()

	fetch := func(env, flag, path string) *eval.TaggedValue {
		t.Helper()
		t.Setenv(config.EnvNetMaxBytes, env)
		effCtx := netLimitCtx(t, "Net")
		if err := SetupNetHandler(effCtx, true, "", true, false, ""); err != nil {
			t.Fatal(err)
		}
		if err := SetupNetLimit(effCtx, flag); err != nil {
			t.Fatal(err)
		}
		v, err := effects.Call(effCtx, "Net", "httpRequest", []eval.Value{
			&eval.StringValue{Value: "GET"},
			&eval.StringValue{Value: server.URL + path},
			&eval.ListValue{},
			&eval.StringValue{Value: ""},
		})
		if err != nil {
			t.Fatalf("httpRequest %s: %v", path, err)
		}
		tv, ok := v.(*eval.TaggedValue)
		if !ok {
			t.Fatalf("httpRequest %s returned %T", path, v)
		}
		return tv
	}
	errCtor := func(tv *eval.TaggedValue) (string, string) {
		t.Helper()
		if tv.CtorName != "Err" {
			t.Fatalf("got %s, want Err", tv.CtorName)
		}
		inner := tv.Fields[0].(*eval.TaggedValue)
		return inner.CtorName, inner.Fields[0].(*eval.StringValue).Value
	}

	// Over the configured cap: BodyTooLarge carries the configured number.
	ctor, arg := errCtor(fetch("1KB", "", "/small"))
	if ctor != "BodyTooLarge" || arg != "1024" {
		t.Errorf("env 1KB, 2KB body: Err(%s(%q)), want Err(BodyTooLarge(\"1024\"))", ctor, arg)
	}
	ctor, arg = errCtor(fetch("50MB", "1500", "/small"))
	if ctor != "BodyTooLarge" || arg != "1500" {
		t.Errorf("flag 1500 over env 50MB, 2KB body: Err(%s(%q)), want Err(BodyTooLarge(\"1500\"))", ctor, arg)
	}

	// The old fixed default still refuses 6 MiB...
	ctor, arg = errCtor(fetch("", "", "/big"))
	if ctor != "BodyTooLarge" || arg != "5242880" {
		t.Errorf("default cap, 6MiB body: Err(%s(%q)), want Err(BodyTooLarge(\"5242880\"))", ctor, arg)
	}
	// ...and a raised cap admits it, byte for byte.
	for _, src := range [][2]string{{"50MB", ""}, {"", "50MB"}} {
		tv := fetch(src[0], src[1], "/big")
		if tv.CtorName != "Ok" {
			c, a := errCtor(tv)
			t.Fatalf("cap 50MB (env %q, flag %q), 6MiB body: Err(%s(%q))", src[0], src[1], c, a)
		}
		rec := tv.Fields[0].(*eval.RecordValue)
		if n := len(rec.Fields["bodyBytes"].(*eval.BytesValue).Value); n != sizes["/big"] {
			t.Errorf("bodyBytes = %d bytes, want %d", n, sizes["/big"])
		}
	}
}
