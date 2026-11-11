package golang

import (
	"bufio"
	"encoding/hex"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/strutil"
)

// quoteAgreePieces are the second reviewer's string classes: NFD and NFC
// text, combining marks in and out of canonical order, Hangul jamo,
// singletons, compatibility characters, ZWJ, variation selectors, U+2028,
// BOM, characters that trigger escapes, and invalid bytes.
var quoteAgreePieces = []string{
	"a", "e", "n", "q", "\"", "\\", "\n", "\t", "\r", "$", "{", "${", " ", "\x00", "\x7f",
	"\U000000E9", "e\U00000301", "\U00000301", "\U00000303", "\U00000308", "\U00000323", "\U00000302", "\U00000344",
	"\U0000212B", "\U00002126", "\U00000958", "\U0000FB01", "\U000000BD", "\U0000FF21",
	"\U00001100", "\U00001161", "\U000011A8", "\U0000AC00", "\U0000AC01",
	"\U00000928", "\U0000093C", "\U00000BC6", "\U00000BBE", "\U000009BE", "\U000005B8", "\U0000064E", "\U00000654",
	"\U0000200D", "\U0000FE0F", "\U000E0100", "\U0001F468", "\U00002028", "\U0000FEFF", "\U00003099", "\U0000304B",
	"\xff", "\xc3",
}

// The quoting rule compiled into a program agrees with strutil.QuoteTo byte
// for byte on 100 000 random strings (one definition of the rule).
func TestEmittedQuoteAgreesWithStrutil(t *testing.T) {
	rnd := rand.New(rand.NewSource(20261111))
	inputs := make([]string, 100000)
	var in strings.Builder
	for i := range inputs {
		var b strings.Builder
		for n := 1 + rnd.Intn(6); n > 0; n-- {
			b.WriteString(quoteAgreePieces[rnd.Intn(len(quoteAgreePieces))])
		}
		// Long runs of marks (the stream-safe limit, review 3 of #2229).
		if rnd.Intn(20) == 0 {
			b.WriteString(strings.Repeat("\U00000301", 25+rnd.Intn(15)))
			if rnd.Intn(50) == 0 {
				b.WriteString(strings.Repeat("\U00000316\U00000301", 500+rnd.Intn(1500)))
			}
		}
		inputs[i] = b.String()
		in.WriteString(hex.EncodeToString([]byte(inputs[i])))
		in.WriteByte('\n')
	}
	runtimeCode, err := New("quoteagree").GenerateRuntime()
	if err != nil {
		t.Fatal(err)
	}
	const driver = `package quoteagree

import (
	"bufio"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestQuoteAll(t *testing.T) {
	in, err := os.Open(os.Getenv("QUOTE_IN"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(os.Getenv("QUOTE_OUT"))
	if err != nil {
		t.Fatal(err)
	}
	w := bufio.NewWriter(out)
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		raw, err := hex.DecodeString(sc.Text())
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		showQuote(&b, string(raw))
		w.WriteString(hex.EncodeToString([]byte(b.String())))
		w.WriteByte('\n')
	}
	w.Flush()
	out.Close()
}
`
	tmp := t.TempDir()
	inPath, outPath := filepath.Join(tmp, "in.hex"), filepath.Join(tmp, "out.hex")
	modDir := filepath.Join(tmp, "mod")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(modDir, "go.mod"):         "module quoteagree\n\ngo 1.21\n",
		filepath.Join(modDir, "runtime.go"):     string(runtimeCode),
		filepath.Join(modDir, "driver_test.go"): driver,
		inPath:                                  in.String(),
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Now()
	cmd := exec.Command("go", "test", "-count=1", "-run", "TestQuoteAll", "./...")
	cmd.Dir = filepath.Join(tmp, "mod")
	cmd.Env = append(os.Environ(), "QUOTE_IN="+inPath, "QUOTE_OUT="+outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("emitted quote driver failed: %v\n%s", err, output)
	}
	t.Logf("compiled driver (build + 100k quotes): %v", time.Since(started))
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	i, escapedNFC := 0, 0
	for ; sc.Scan(); i++ {
		got, err := hex.DecodeString(sc.Text())
		if err != nil {
			t.Fatal(err)
		}
		want := strutil.Quote(inputs[i])
		if string(got) != want {
			t.Fatalf("string %d %+q: compiled program quotes %s, strutil %s", i, inputs[i], got, want)
		}
		if strings.Contains(want, `\u{`) {
			escapedNFC++
		}
	}
	if i != len(inputs) {
		t.Fatalf("compiled program quoted %d of %d strings", i, len(inputs))
	}
	if escapedNFC < 10000 {
		t.Fatalf("only %d quotes used \\u{…}; the inputs do not exercise the NFC rule", escapedNFC)
	}
}
