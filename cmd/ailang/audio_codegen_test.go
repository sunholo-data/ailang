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
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// This exercises the public compiler and actually builds/runs its output.
// Registry metadata tests alone cannot catch an invalid []byte/*Bytes cast or
// a generated WAV parser whose rejection policy drifts from pure std/audio.
func TestAudioGeneratedGoExampleAndMalformedCorpus(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "audio_probe.ail")
	code := `module audio_probe
import std/audio (wavInfo, wavFromPcm, wavHeaderChecked, pcm16ToFloats)
import std/bytes (fromInts)
import std/array as A
import std/result (Ok, Err)
export pure func main(xs: [int]) -> bool {
 match wavInfo(fromInts(xs)) { Ok(_) => true, Err(_) => false }
}
export pure func legacy(pcm: bytes) -> bytes = wavFromPcm(pcm, 8000, 1, 16)
export pure func arrayResult() -> Array[float] {
 match pcm16ToFloats(fromInts([0,0]),0,1,1) {
   Err(_) => A.fromList([0.0]),
   Ok(xs) => xs
 }
}
export pure func arraySet() -> float = A.get(A.set(arrayResult(),0,0.5),0)
export pure func firstClassHeader() -> bool {
 let header = wavHeaderChecked;
 match header(0,8000,1,16) { Ok(_) => true, Err(_) => false }
}
export pure func firstClassDecode() -> bool {
 let decode = pcm16ToFloats;
 match decode(fromInts([0,0]),0,1,1) { Ok(xs) => A.length(xs) == 1, Err(_) => false }
}
export pure func recursiveTotal(n: int) -> int = if n <= 0 then 0 else n + recursiveTotal(n - 1)
export pure func sin(x: float) -> float = x + 1.0
export pure func trim(x: string) -> string = x
export pure func topLevelNames() -> bool = sin(1.0) == 2.0 && trim(" value ") == " value "
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
	outdir := filepath.Join(dir, "generated")
	// runAilangBin uses the project root; a relative source keeps MOD010's
	// canonical module identity on every host (including Windows).
	example := filepath.Join("examples", "runnable", "audio_checked.ail")
	out, stderr, rc := runAilangBin(t, bin, "compile", "--emit-go", "--package-name", "audiocheck", "--out", outdir, example, probe)
	if rc != 0 {
		t.Fatalf("compile real audio example: %d\n%s\n%s", rc, out, stderr)
	}
	generated := filepath.Join(outdir, "audiocheck")
	if err := os.WriteFile(filepath.Join(generated, "go.mod"), []byte("module audiocheck\n\ngo 1.21\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var test strings.Builder
	test.WriteString("package audiocheck\nimport(\"testing\";\"strings\";\"fmt\")\nfunc TestActualExample(t *testing.T) { if !audio_checked__RoundTrip() || !audio_checked__RejectsPartialFrame() || audio_checked__FrameBytes(2)!=4 { t.Fatal(\"actual example failed\") } }\n")
	test.WriteString("func TestMetadataCorpus(t *testing.T) { cases:=[]struct{name string;raw []byte; accepted bool}{\n")
	for _, c := range fixtures {
		fmt.Fprintf(&test, "{%q, %#v, %t},\n", c.name, c.raw, c.accepted)
	}
	test.WriteString(`}
for _,c:=range cases {
 r:=AudioWavInfo(c.raw).(*Result)
 if (r.Kind==ResultKindOk)!=c.accepted {t.Errorf("%s: got %v",c.name,r)}
 if c.accepted {
   info:=r.Ok.Value0.(*WavInfo)
   if info.Rate!=8000 || info.Channels!=1 || info.ContainerBits!=16 || info.ValidBits!=16 || info.BlockAlign!=2 || info.ByteRate!=16000 || info.DataBytes!=0 || info.Frames!=0 { t.Errorf("%s: invalid metadata %+v",c.name,info) }
 }
}
}
func TestCheckedAndLegacyFailure(t *testing.T) {
 if AudioWavFromPcmChecked([]byte{0},int64(8000),int64(1),int64(16)).(*Result).Kind!=ResultKindErr {t.Fatal("partial frame accepted")}
 func(){ defer func(){if r:=recover();r==nil || !strings.Contains(fmt.Sprint(r),"frame"){t.Errorf("legacy partial frame did not fail loudly: %v",r)}}();AudioWavFromPcm([]byte{0},int64(8000),int64(1),int64(16)) }()
 r:=AudioWavFromPcmChecked([]byte{128},int64(8000),int64(1),int64(8)).(*Result)
 if r.Kind!=ResultKindOk {t.Fatal(r)}
 b:=r.Ok.Value0.([]byte)
 if len(b)!=46 || b[4]!=38 || b[40]!=1 || b[45]!=0 {t.Fatalf("odd-byte WAV incorrect: %v",b)}
}
func TestPackedArrayComposition(t *testing.T) {
 xs:=audio_probe__ArrayResult()
 if len(xs)!=1 || xs[0]!=0 {t.Fatalf("decoded return: %v",xs)}
 if got:=audio_probe__ArraySet();got!=0.5 {t.Fatalf("packed set/get: %v",got)}
 if xs[0]!=0 {t.Fatal("packed source mutated")}
}
func TestFirstClassFourArgumentAudio(t *testing.T) {
 if !audio_probe__FirstClassHeader() {t.Fatal("first-class header rejected valid input")}
 if !audio_probe__FirstClassDecode() {t.Fatal("first-class decode rejected valid input")}
}
func TestTopLevelRecursion(t *testing.T) {
 if got:=audio_probe__RecursiveTotal(5);got!=15 {t.Fatalf("recursion: %v",got)}
 if !audio_probe__TopLevelNames() {t.Fatal("user top-level functions were replaced by builtins")}
}
func TestPackedEncoderRejectsNonFloatElements(t *testing.T) {
 defer func(){if r:=recover();r==nil||!strings.Contains(fmt.Sprint(r),"ConvertToFloatArray"){t.Fatalf("malformed float array did not fail loudly: %v",r)}}()
 AudioFloatsToPCM16(ArrayVal{0.0,"bad"},int64(1),false)
}
`)
	if err := os.WriteFile(filepath.Join(generated, "audio_generated_test.go"), []byte(test.String()), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, rc := testutil.RunBounded(t, generated, 90*time.Second, "go", "test", "-count=1", ".")
	if rc != 0 {
		t.Fatalf("build/run emitted Go: exit %d\n%s\n%s", rc, stdout, stderr)
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
