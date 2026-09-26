package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

// Journals have random names, so scan a bounded directory and accept only progress
// bound to the frozen child request. Never expose journal text or provider fields.
func readIterationReceiptProgress(dir, stage, digest string) (*dispatch.Progress, string) {
	if !filepath.IsAbs(dir) {
		return nil, "unavailable: receipt location is not absolute"
	}
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil || canonical != filepath.Clean(dir) {
		return nil, "unavailable: receipt directory missing or redirected"
	}
	f, err := os.Open(dir)
	if err != nil {
		return nil, "unavailable: receipt directory unreadable"
	}
	entries, err := f.ReadDir(129)
	closeErr := f.Close()
	if (err != nil && err != io.EOF) || closeErr != nil {
		return nil, "unavailable: receipt directory unreadable"
	}
	if len(entries) > 128 {
		return nil, "unavailable: receipt directory exceeds inspection bound"
	}
	var found *dispatch.Progress
	status := "unavailable: no matching complete progress event"
	total := int64(0)
	for _, entry := range entries {
		name := entry.Name()
		prefix := stage + "-"
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		nonce := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".jsonl")
		if decoded, e := hex.DecodeString(nonce); e != nil || len(decoded) != 16 {
			continue
		}
		if !entry.Type().IsRegular() {
			return nil, "unavailable: receipt is not a regular file"
		}
		path := filepath.Join(dir, name)
		info, e := os.Lstat(path)
		if e != nil || !info.Mode().IsRegular() {
			return nil, "unavailable: receipt cannot be inspected"
		}
		file, e := os.Open(path)
		if e != nil {
			return nil, "unavailable: receipt cannot be read"
		}
		opened, e := file.Stat()
		if e != nil || !os.SameFile(info, opened) {
			file.Close()
			return nil, "unavailable: receipt changed during inspection"
		}
		data, e := io.ReadAll(io.LimitReader(file, 4*1024*1024+1))
		ce := file.Close()
		total += int64(len(data))
		if e != nil || ce != nil {
			return nil, "unavailable: receipt read failed"
		}
		if len(data) > 4*1024*1024 || total > 16*1024*1024 {
			return nil, "unavailable: receipt data exceeds inspection bound"
		}
		var progress *dispatch.Progress
		incomplete := false
		lines := bytes.Split(data, []byte{'\n'})
		for i, line := range lines {
			if len(line) == 0 {
				continue
			}
			if i == len(lines)-1 {
				incomplete = true
				break
			}
			// Minimal projection avoids retaining prompts, transcripts or stderr.
			var event struct {
				Version int    `json:"version"`
				Kind    string `json:"kind"`
				Digest  string `json:"request_digest"`
				Report  *struct {
					Digest   string             `json:"request_digest"`
					Progress *dispatch.Progress `json:"progress"`
				} `json:"report"`
			}
			if json.Unmarshal(line, &event) != nil {
				incomplete = true
				break
			}
			if event.Kind != "progress" || event.Digest != digest {
				continue
			}
			if event.Version != 1 || event.Report == nil || event.Report.Digest != digest || event.Report.Progress == nil {
				incomplete = true
				break
			}
			p := event.Report.Progress
			if p.ToolCalls < 0 || p.CompletedTools < 0 || p.RepeatedCalls < 0 || p.RepeatedCalls > p.ToolCalls || !safeIterationProgressTool(p.LastCompletedTool) {
				incomplete = true
				break
			}
			progress = p
		}
		if progress != nil {
			if found != nil {
				return nil, "unavailable: multiple journals contain progress for this request"
			}
			found = progress
			status = "receipt: last complete observed progress event"
			if incomplete {
				status = "receipt incomplete: showing last complete observed progress event"
			}
		} else if incomplete && found == nil {
			status = "unavailable: receipt incomplete without matching progress"
		}
	}
	return found, status
}
func safeIterationProgressTool(name string) bool {
	if len(name) > 80 {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}
