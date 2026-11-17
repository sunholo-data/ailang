package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// std/audio.wavInfo accepts or rejects each WAV in a malformed corpus the same
// way on the evaluator and the strict VM.
func TestAudioWavInfoMalformedCorpusEvaluatorAndVM(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "audio_probe.ail")
	code := `module audio_probe
import std/audio (wavInfo)
import std/bytes (fromInts)
import std/result (Ok, Err)
export pure func main(xs: [int]) -> bool {
 match wavInfo(fromInts(xs)) { Ok(_) => true, Err(_) => false }
}
`
	if err := os.WriteFile(probe, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	fixtures := audioMalformedCorpus()
	for i, c := range fixtures {
		ints := make([]int, len(c.raw))
		for j, b := range c.raw {
			ints[j] = int(b)
		}
		data, err := json.Marshal(ints)
		if err != nil {
			t.Fatal(err)
		}
		argfile := filepath.Join(dir, fmt.Sprintf("args-%d.json", i))
		if err := os.WriteFile(argfile, data, 0600); err != nil {
			t.Fatal(err)
		}
		for _, vm := range []bool{false, true} {
			args := []string{"run", "--stdlib-path", filepath.Join(root, "std"), "--args-file", argfile}
			if vm {
				args = append(args, "--strict-bytecode")
			}
			args = append(args, probe)
			out, stderr, rc := runAilangBin(t, bin, args...)
			if rc != 0 || strings.TrimSpace(out) != strconv.FormatBool(c.accepted) {
				t.Fatalf("%s vm=%v: exit %d stdout=%q stderr=%s", c.name, vm, rc, out, stderr)
			}
		}
	}
}

type audioCorpusCase struct {
	name     string
	raw      []byte
	accepted bool
}

func audioMalformedCorpus() []audioCorpusCase {
	le := func(n uint32) []byte { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, n); return b }
	chunk := func(id string, data []byte, pad bool) []byte {
		out := append(append([]byte(id), le(uint32(len(data)))...), data...)
		if pad && len(data)%2 != 0 {
			out = append(out, 77)
		}
		return out
	}
	riff := func(chunks ...[]byte) []byte {
		body := []byte("WAVE")
		for _, c := range chunks {
			body = append(body, c...)
		}
		return append(append([]byte("RIFF"), le(uint32(len(body)))...), body...)
	}
	fmtData := []byte{1, 0, 1, 0, 64, 31, 0, 0, 128, 62, 0, 0, 2, 0, 16, 0}
	format := chunk("fmt ", fmtData, true)
	data := chunk("data", nil, true)
	change := func(at int, b byte) []byte {
		x := append([]byte(nil), fmtData...)
		x[at] = b
		return chunk("fmt ", x, true)
	}
	missingPad := riff(format, chunk("JUNK", []byte{1}, false))
	missingPad = append(missingPad, 0)
	shortRiff := riff(format, data)
	binary.LittleEndian.PutUint32(shortRiff[4:8], 3)
	giantChunk := append(append([]byte("data"), le(0xffffffff)...), 0, 0)
	return []audioCorpusCase{
		{"canonical", riff(format, data), true}, {"data before fmt", riff(data, format), true},
		{"odd unknown pad", riff(chunk("JUNK", []byte{1}, true), format, data), true},
		{"trailing bytes ignored", append(riff(format, data), 1, 2, 3), true},
		{"unknown after data", riff(format, data, chunk("JUNK", []byte{2}, true)), true},
		{"PCM fmt extension", riff(chunk("fmt ", append(append([]byte(nil), fmtData...), 0, 0), true), data), true},
		{"missing fmt", riff(data), false}, {"missing data", riff(format), false},
		{"duplicate fmt", riff(format, format, data), false}, {"duplicate data", riff(format, data, data), false},
		{"bad alignment", riff(change(12, 1), data), false}, {"bad byte rate", riff(change(8, 1), data), false},
		{"short fmt", riff(chunk("fmt ", []byte{1, 0}, true), data), false},
		{"17-byte fmt", riff(chunk("fmt ", append(append([]byte(nil), fmtData...), 0), true), data), false},
		{"cbSize extent", riff(chunk("fmt ", append(append([]byte(nil), fmtData...), 1, 0), true), data), false},
		{"float format", riff(change(0, 3), data), false}, {"extensible format", riff(change(0, 254), data), false},
		{"partial s16 frame", riff(format, chunk("data", []byte{0}, true)), false},
		{"chunk exceeds RIFF", riff(format, giantChunk), false}, {"pad outside RIFF", missingPad, false},
		{"RIFF missing WAVE extent", shortRiff, false}, {"partial chunk header", riff(format, data, []byte{1, 2, 3}), false},
		{"bad magic", []byte("RIFX\x04\x00\x00\x00WAVE"), false}, {"short prefix", []byte("RIFF"), false},
	}
}
