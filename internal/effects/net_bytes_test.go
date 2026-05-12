package effects

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// netBytesTestCtx returns an EffContext configured to talk to httptest.Server
// (HTTP, localhost). All NetHTTPRequestBytes tests use this — we want the
// security path to fire (we test that explicitly) but not block localhost http.
func netBytesTestCtx(t *testing.T) *EffContext {
	t.Helper()
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Net"))
	ctx.Net = NewNetContext()
	ctx.Net.AllowHTTP = true
	ctx.Net.AllowLocalhost = true
	return ctx
}

func bytesArg(b []byte) *eval.BytesValue {
	return &eval.BytesValue{Value: b}
}

func unwrapOkRecord(t *testing.T, v eval.Value) *eval.RecordValue {
	t.Helper()
	tagged, ok := v.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", v)
	}
	if tagged.CtorName != "Ok" {
		// Surface error for easier debugging
		if errTagged, ok := tagged.Fields[0].(*eval.TaggedValue); ok {
			msg := ""
			if len(errTagged.Fields) > 0 {
				if s, ok := errTagged.Fields[0].(*eval.StringValue); ok {
					msg = s.Value
				}
			}
			t.Fatalf("expected Ok, got Err(%s(%q))", errTagged.CtorName, msg)
		}
		t.Fatalf("expected Ok, got %s", tagged.CtorName)
	}
	rec, ok := tagged.Fields[0].(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue inside Ok, got %T", tagged.Fields[0])
	}
	return rec
}

// TestNetHTTPRequestBytes_RoundTripSHA verifies the request body bytes arrive
// at the server byte-for-byte (acceptance criterion: PUT 1KB random bytes,
// server returns SHA256, assert client SHA matches).
func TestNetHTTPRequestBytes_RoundTripSHA(t *testing.T) {
	payload := make([]byte, 1024)
	if _, err := rand.Read(payload); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	expectedSHA := sha256.Sum256(payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		sum := sha256.Sum256(body)
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:])))
	}))
	defer server.Close()

	ctx := netBytesTestCtx(t)
	args := []eval.Value{
		&eval.StringValue{Value: "PUT"},
		&eval.StringValue{Value: server.URL + "/upload"},
		&eval.ListValue{Elements: []eval.Value{}},
		bytesArg(payload),
	}

	result, err := NetHTTPRequestBytes(ctx, args)
	if err != nil {
		t.Fatalf("NetHTTPRequestBytes returned Go error: %v", err)
	}
	resp := unwrapOkRecord(t, result)

	body := resp.Fields["body"].(*eval.StringValue).Value
	if body != hex.EncodeToString(expectedSHA[:]) {
		t.Errorf("server SHA mismatch.\n  want: %s\n  got:  %s", hex.EncodeToString(expectedSHA[:]), body)
	}
}

// TestNetHTTPRequestBytes_DefaultContentType verifies octet-stream is set when
// caller omits Content-Type, and caller-supplied Content-Type wins.
func TestNetHTTPRequestBytes_DefaultContentType(t *testing.T) {
	var receivedCT string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCT = r.Header.Get("Content-Type")
		w.WriteHeader(200)
	}))
	defer server.Close()

	t.Run("default octet-stream when caller omits", func(t *testing.T) {
		ctx := netBytesTestCtx(t)
		_, err := NetHTTPRequestBytes(ctx, []eval.Value{
			&eval.StringValue{Value: "POST"},
			&eval.StringValue{Value: server.URL},
			&eval.ListValue{Elements: []eval.Value{}},
			bytesArg([]byte{1, 2, 3}),
		})
		if err != nil {
			t.Fatalf("Go error: %v", err)
		}
		if receivedCT != "application/octet-stream" {
			t.Errorf("expected Content-Type=application/octet-stream, got %q", receivedCT)
		}
	})

	t.Run("caller-supplied Content-Type wins", func(t *testing.T) {
		ctx := netBytesTestCtx(t)
		headers := &eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{Fields: map[string]eval.Value{
				"name":  &eval.StringValue{Value: "Content-Type"},
				"value": &eval.StringValue{Value: "image/png"},
			}},
		}}
		_, err := NetHTTPRequestBytes(ctx, []eval.Value{
			&eval.StringValue{Value: "POST"},
			&eval.StringValue{Value: server.URL},
			headers,
			bytesArg([]byte{1, 2, 3}),
		})
		if err != nil {
			t.Fatalf("Go error: %v", err)
		}
		if receivedCT != "image/png" {
			t.Errorf("expected Content-Type=image/png (caller override), got %q", receivedCT)
		}
	})
}

// TestNetHTTPRequestBytes_ContentLengthExplicit verifies Content-Length is set
// (no chunked encoding), including for empty body.
func TestNetHTTPRequestBytes_ContentLengthExplicit(t *testing.T) {
	var receivedLen int64
	var receivedTE string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLen = r.ContentLength
		if len(r.TransferEncoding) > 0 {
			receivedTE = r.TransferEncoding[0]
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	cases := []struct {
		name string
		body []byte
	}{
		{"non-empty body", []byte("hello binary")},
		{"empty body", []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := netBytesTestCtx(t)
			_, err := NetHTTPRequestBytes(ctx, []eval.Value{
				&eval.StringValue{Value: "PUT"},
				&eval.StringValue{Value: server.URL},
				&eval.ListValue{Elements: []eval.Value{}},
				bytesArg(tc.body),
			})
			if err != nil {
				t.Fatalf("Go error: %v", err)
			}
			if receivedLen != int64(len(tc.body)) {
				t.Errorf("expected Content-Length=%d, got %d", len(tc.body), receivedLen)
			}
			if receivedTE == "chunked" {
				t.Errorf("expected no chunked encoding, got Transfer-Encoding=%q", receivedTE)
			}
		})
	}
}

// TestNetHTTPRequestBytes_ResponseBodyBytes verifies binary response data is
// captured byte-for-byte in resp.bodyBytes (resolves the transparent-gzip
// open question — server uses Content-Encoding: identity).
func TestNetHTTPRequestBytes_ResponseBodyBytes(t *testing.T) {
	respPayload := make([]byte, 256)
	if _, err := rand.Read(respPayload); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Encoding", "identity")
		_, _ = w.Write(respPayload)
	}))
	defer server.Close()

	ctx := netBytesTestCtx(t)
	result, err := NetHTTPRequestBytes(ctx, []eval.Value{
		&eval.StringValue{Value: "GET"},
		&eval.StringValue{Value: server.URL},
		&eval.ListValue{Elements: []eval.Value{}},
		bytesArg(nil),
	})
	if err != nil {
		t.Fatalf("Go error: %v", err)
	}
	resp := unwrapOkRecord(t, result)

	bodyBytesVal, ok := resp.Fields["bodyBytes"].(*eval.BytesValue)
	if !ok {
		t.Fatalf("expected bodyBytes to be BytesValue, got %T", resp.Fields["bodyBytes"])
	}
	if len(bodyBytesVal.Value) != len(respPayload) {
		t.Fatalf("expected bodyBytes len=%d, got %d", len(respPayload), len(bodyBytesVal.Value))
	}
	for i := range respPayload {
		if bodyBytesVal.Value[i] != respPayload[i] {
			t.Fatalf("bodyBytes mismatch at index %d", i)
		}
	}
}

// TestNetHTTPRequestBytes_CapabilityRequired verifies Net capability is enforced.
func TestNetHTTPRequestBytes_CapabilityRequired(t *testing.T) {
	ctx := NewEffContext([]string{}) // no caps

	_, err := NetHTTPRequestBytes(ctx, []eval.Value{
		&eval.StringValue{Value: "GET"},
		&eval.StringValue{Value: "https://example.com"},
		&eval.ListValue{Elements: []eval.Value{}},
		bytesArg([]byte{}),
	})
	if err == nil {
		t.Fatal("expected capability error, got nil")
	}
	if capErr, ok := err.(*CapabilityError); !ok || capErr.Effect != "Net" {
		t.Errorf("expected CapabilityError(Net), got %v", err)
	}
}

// TestNetHTTPRequestBytes_BodyMustBeBytes verifies type error if body is wrong type.
func TestNetHTTPRequestBytes_BodyMustBeBytes(t *testing.T) {
	ctx := netBytesTestCtx(t)
	_, err := NetHTTPRequestBytes(ctx, []eval.Value{
		&eval.StringValue{Value: "PUT"},
		&eval.StringValue{Value: "https://example.com"},
		&eval.ListValue{Elements: []eval.Value{}},
		&eval.StringValue{Value: "not bytes"}, // wrong: should be BytesValue
	})
	if err == nil {
		t.Fatal("expected type error for string body, got nil")
	}
}

// TestNetHTTPRequestBytes_SecurityChecksFire verifies the M1-extracted
// buildSecureRequest helper is invoked: blocked header should produce
// InvalidHeader error, just as it would for httpRequest.
func TestNetHTTPRequestBytes_SecurityChecksFire(t *testing.T) {
	ctx := netBytesTestCtx(t)
	headers := &eval.ListValue{Elements: []eval.Value{
		&eval.RecordValue{Fields: map[string]eval.Value{
			"name":  &eval.StringValue{Value: "Connection"}, // hop-by-hop, blocked
			"value": &eval.StringValue{Value: "close"},
		}},
	}}
	result, err := NetHTTPRequestBytes(ctx, []eval.Value{
		&eval.StringValue{Value: "PUT"},
		&eval.StringValue{Value: "http://127.0.0.1:1/anything"}, // never dialed
		headers,
		bytesArg([]byte{1}),
	})
	if err != nil {
		t.Fatalf("Go error: %v", err)
	}
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Err" {
		t.Fatalf("expected Err result for blocked header, got %v", result)
	}
	errVal := tagged.Fields[0].(*eval.TaggedValue)
	if errVal.CtorName != "InvalidHeader" {
		t.Errorf("expected InvalidHeader, got %s", errVal.CtorName)
	}
}
