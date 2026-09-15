// Package httpjson is the ONE way an AILANG HTTP handler writes a JSON
// response. Seven private writeJSON copies (coordinator, observatory,
// apiserver ×2, server, cmd) each set the header, wrote the status and
// encoded — and disagreed only about what to do when encoding failed
// (ignore it, log it, or call http.Error after the status had already
// gone out, which Go reports as a superfluous WriteHeader). Stdlib leaf.
// M-V1-SIMPLIFY-S3 M5.
package httpjson

import (
	"encoding/json"
	"log"
	"net/http"
)

// Write sets Content-Type: application/json, writes status, and encodes v.
// The status line has already been sent by the time encoding can fail, so
// a failure is logged, not turned into a second response.
func Write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpjson: encoding %T after status %d: %v", v, status, err)
	}
}

// Error writes {"error": message} with the given status — the shape every
// AILANG API already returns for failures.
func Error(w http.ResponseWriter, status int, message string) {
	Write(w, status, map[string]string{"error": message})
}
