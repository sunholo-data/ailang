// ollama-tap is a transparent logging reverse-proxy for ollama. It forwards
// every request to an upstream ollama and appends the full request body (plus
// method, path, and a harness tag) to a JSONL log — so you can capture EXACTLY
// what each coding harness (motoko, pi, opencode, …) sends to the same model and
// diff the system prompts / tool schemas / sampling params apples-to-apples.
//
// Why a proxy: motoko routes through AILANG's internal/ai/ollama; pi/opencode
// drive ollama's /v1 directly. None of them log the outbound request body in a
// comparable form. They ALL POST to ollama's HTTP API, so one proxy captures all
// three identically with no per-harness code changes — just repoint the endpoint.
//
// Usage:
//
//	OLLAMA_TAP_LOG=/tmp/tap.jsonl go run ./tools/ollama-tap            # listen :11435 → :11434
//	# then point ONE harness at the tap for one benchmark, e.g.:
//	#   motoko:   OLLAMA_HOST=http://localhost:11435  (AILANG reads OLLAMA_HOST)
//	#   pi:       set its ollama base URL to http://localhost:11435/v1
//	#   opencode: likewise
//	# tag the source so the log is easy to split:
//	#   OLLAMA_HOST=http://localhost:11435?harness=motoko
//	#   (the tap reads ?harness=… from the URL, or the X-Harness header)
//
// Env:
//
//	OLLAMA_TAP_LISTEN    (default ":11435")
//	OLLAMA_TAP_UPSTREAM  (default "http://localhost:11434")
//	OLLAMA_TAP_LOG       (default "ollama-tap.jsonl") — JSONL of captured requests
//
// Each log line: {"ts","harness","method","path","body"} where body is the raw
// request payload (the /v1/chat/completions or /api/chat JSON). Responses are
// streamed through untouched (we only tap the request).
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type capture struct {
	TS      string `json:"ts"`
	Harness string `json:"harness"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Body    string `json:"body"`
}

func main() {
	listen := envOr("OLLAMA_TAP_LISTEN", ":11435")
	upstreamRaw := envOr("OLLAMA_TAP_UPSTREAM", "http://localhost:11434")
	logPath := envOr("OLLAMA_TAP_LOG", "ollama-tap.jsonl")

	upstream, err := url.Parse(upstreamRaw)
	if err != nil {
		log.Fatalf("bad OLLAMA_TAP_UPSTREAM %q: %v", upstreamRaw, err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Fatalf("cannot open log %q: %v", logPath, err)
	}
	defer func() { _ = logFile.Close() }()

	var mu sync.Mutex // serialize log writes
	enc := json.NewEncoder(logFile)

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	// Preserve the upstream host header (ollama is lenient, but be correct).
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		origDirector(r)
		r.Host = upstream.Host
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Tee the body: read it for logging, then restore it for forwarding.
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			r.ContentLength = int64(len(bodyBytes))
		}

		// Harness tag: ?harness=… query param wins, else X-Harness header, else "".
		harness := r.URL.Query().Get("harness")
		if harness == "" {
			harness = r.Header.Get("X-Harness")
		}

		// Only log payload-bearing model calls; skip health/list/ps chatter.
		if len(bodyBytes) > 0 {
			mu.Lock()
			_ = enc.Encode(capture{
				TS:      time.Now().UTC().Format(time.RFC3339Nano),
				Harness: harness,
				Method:  r.Method,
				Path:    r.URL.Path,
				Body:    string(bodyBytes),
			})
			mu.Unlock()
		}

		proxy.ServeHTTP(w, r)
	}

	log.Printf("ollama-tap: %s → %s, logging requests to %s", listen, upstream, logPath)
	if err := http.ListenAndServe(listen, http.HandlerFunc(handler)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
