package config

import (
	"strconv"
	"time"
)

// The Ollama client (internal/ai/ollama). These reach a launchd job by two
// paths — the plist AND a `launchctl setenv` domain global — and nothing here
// changes that; the getters read what the process was given. Semantics are
// the ones docs/docs/guides/debugging.md ("Ollama Streaming Timeouts")
// documents, including the meaning flip of AILANG_OLLAMA_HTTP_TIMEOUT_SEC
// under AILANG_OLLAMA_V1_STREAM=1.
const (
	EnvOllamaHost           = "OLLAMA_HOST"
	EnvOllamaV1Stream       = "AILANG_OLLAMA_V1_STREAM"
	EnvOllamaHTTPTimeoutSec = "AILANG_OLLAMA_HTTP_TIMEOUT_SEC"
	EnvOllamaIdleTimeoutSec = "AILANG_OLLAMA_IDLE_TIMEOUT_SEC"
	EnvOllamaTTFTTimeoutSec = "AILANG_OLLAMA_TTFT_TIMEOUT_SEC"
	EnvOllamaTemperature    = "AILANG_OLLAMA_TEMPERATURE"
	EnvOllamaNumCtx         = "AILANG_OLLAMA_NUM_CTX"
	EnvOllamaMaxTokens      = "AILANG_OLLAMA_MAX_TOKENS"
	EnvOllamaLogRequests    = "AILANG_OLLAMA_LOG_REQUESTS"
	EnvOllamaNativeTools    = "AILANG_OLLAMA_NATIVE_TOOLS"
)

// The soft streaming windows' defaults, in seconds. The hard deadline's two
// defaults (300 buffered / 3600 streaming) belong to the two resolvers in
// internal/ai/ollama, because which one applies is decided there.
const (
	DefaultOllamaIdleTimeoutSec = 120
	DefaultOllamaTTFTTimeoutSec = 600
)

var ollamaVars = []Var{
	{EnvOllamaHost, "", AreaOllama, "Ollama server every client in the process talks to; set, it wins even over an explicit endpoint option, and unset means http://127.0.0.1:11434 (IPv4-pinned so the harness cannot reach an uncapped listener over ::1)."},
	{EnvOllamaV1Stream, "0", AreaOllama, "Exactly 1 opts the tool-calling /v1 path into streaming with the three watchdog windows; anything else keeps the buffered path with byte-identical requests."},
	{EnvOllamaHTTPTimeoutSec, "", AreaOllama, "Total budget for one /v1 call in seconds, with two meanings: buffered (flag off) it is the HTTP client timeout, default 300, where 0 or negative means no timeout; streaming (flag on) it is the mandatory hard deadline, default 3600, where 0, negative or unparseable is rejected at client construction."},
	{EnvOllamaIdleTimeoutSec, "120", AreaOllama, "Streaming only: max silence between bytes in seconds before the typed idle-timeout trips; non-positive or malformed keeps the default."},
	{EnvOllamaTTFTTimeoutSec, "600", AreaOllama, "Streaming only: max silence before the first byte in seconds (long on purpose: a cold 35B load takes minutes); non-positive or malformed keeps the default."},
	{EnvOllamaTemperature, "", AreaOllama, "Sampling temperature for the agentic path when the request sets none; a value that is not a float > 0 sends no temperature, leaving the model default."},
	{EnvOllamaNumCtx, "", AreaOllama, "Pins ollama's num_ctx on the non-tool paths; unset sends none so ollama sizes the context from the model. Raise or lower only for VRAM."},
	{EnvOllamaMaxTokens, "", AreaOllama, "Output budget for the agentic path; a positive integer overrides both the request's value and the built-in floor."},
	{EnvOllamaLogRequests, "", AreaOllama, "JSONL path each logical request (and every streaming stream_metrics record) is appended to; unset falls back to the ollama-log-requests sentinel file under the state dir, whose contents are the path."},
	{EnvOllamaNativeTools, "0", AreaOllama, "1 forces tool-calling turns down the legacy native /api/chat path instead of /v1."},
}

// OllamaHost returns OLLAMA_HOST verbatim, "" when unset.
func OllamaHost() string { return get(EnvOllamaHost) }

// OllamaV1Stream reports AILANG_OLLAMA_V1_STREAM=1 exactly: default-off is a
// contract (with the flag unset the wire bytes must be identical), so this is
// an opt-in test, never `!= "0"`.
func OllamaV1Stream() bool { return getOr(EnvOllamaV1Stream) == "1" }

// OllamaHTTPTimeoutSec returns AILANG_OLLAMA_HTTP_TIMEOUT_SEC verbatim, ""
// when unset. It is deliberately not parsed here: the buffered path maps
// "0" to no timeout and the streaming path rejects it, and each resolver in
// internal/ai/ollama owns its rule (and its test).
func OllamaHTTPTimeoutSec() string { return get(EnvOllamaHTTPTimeoutSec) }

// OllamaIdleTimeout returns AILANG_OLLAMA_IDLE_TIMEOUT_SEC as a duration when
// it is a positive integer, else the registered default.
func OllamaIdleTimeout() time.Duration {
	return positiveSeconds(EnvOllamaIdleTimeoutSec, DefaultOllamaIdleTimeoutSec)
}

// OllamaTTFTTimeout returns AILANG_OLLAMA_TTFT_TIMEOUT_SEC as a duration when
// it is a positive integer, else the registered default.
func OllamaTTFTTimeout() time.Duration {
	return positiveSeconds(EnvOllamaTTFTTimeoutSec, DefaultOllamaTTFTTimeoutSec)
}

// positiveSeconds reads a positive-integer seconds value, falling back to
// defSec when unset or unusable. Only for the soft windows, where a bad value
// costs nothing but a wrong watchdog period.
func positiveSeconds(name string, defSec int) time.Duration {
	if v := get(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return time.Duration(defSec) * time.Second
}

// OllamaTemperature returns AILANG_OLLAMA_TEMPERATURE and true when it parses
// to a float > 0; otherwise (0, false) and the caller sends no temperature.
func OllamaTemperature() (float64, bool) {
	if v := get(EnvOllamaTemperature); v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil && t > 0 {
			return t, true
		}
	}
	return 0, false
}

// OllamaNumCtx returns AILANG_OLLAMA_NUM_CTX and true when it is a positive
// integer; otherwise (0, false) and no num_ctx is sent.
func OllamaNumCtx() (int, bool) { return positiveInt(EnvOllamaNumCtx) }

// OllamaMaxTokens returns AILANG_OLLAMA_MAX_TOKENS and true when it is a
// positive integer; otherwise (0, false).
func OllamaMaxTokens() (int, bool) { return positiveInt(EnvOllamaMaxTokens) }

func positiveInt(name string) (int, bool) {
	if v := get(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

// OllamaLogRequests returns AILANG_OLLAMA_LOG_REQUESTS verbatim, "" when unset.
func OllamaLogRequests() string { return get(EnvOllamaLogRequests) }

// OllamaNativeTools reports AILANG_OLLAMA_NATIVE_TOOLS=1.
func OllamaNativeTools() bool { return getOr(EnvOllamaNativeTools) == "1" }
