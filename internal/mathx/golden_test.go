package mathx

import (
	"math"
	"os"
	"runtime"
	"testing"
)

// Golden bits for #1465. These values are the same on every GOARCH — that is
// the whole point — so a failure on any machine means the portable
// implementation drifted (or an FMA got fused back in).
//
// Provenance: generated with go1.26.6 from mathx on darwin/arm64 and checked
// bit-identical against both mathx and the host math package on js/wasm
// (the Go pure-Go algorithms with no assembly and no fusion).
// Regenerate with MATHX_PRINT_GOLDEN=1 go test -run TestGoldenSweepDigests -v.

// goldenSweepDigests is SHA-256 over the output bits of the full sweep corpus
// (corpus_test.go: specials + 20,000 seeded inputs per function).
var goldenSweepDigests = map[string]string{
	"exp":   "fe89300bd21c9818291e5beef7ac50586cff2eae4a30d6f18a30aaa167a79867",
	"log":   "45a0e2430c5d6352e5e18ad8c71b61f1a94817dad9cff5544c4bd4380ab02c39",
	"log10": "d4aa2561766d1bde9f4fe0fcd197fc0d5a70b873c6e17af3dc2db0aa332e1296",
	"sin":   "ce65dd5b965f8fa155822e6e6005a2e8e9a78c16f00fe17a4ee04c41d4c88984",
	"cos":   "46688f1534a6f7230ab74805e8f9b6eb98232ad77123e168ca22566074c0b2e0",
	"tan":   "ff49b06bf803d5b51a2ef231aea3e62eba0829d27b2ee9ff189226e781fd50a9",
	"asin":  "80610a3648d0ba1cd9e117aa7c3ba1005f1b7f7aa8e1fa5d205f10d5470c5d3e",
	"acos":  "f4980fd75145baccdcf9455557c29b2de4d4e996f4332c63c1b5825be5627c63",
	"atan":  "30bbfff640f43681066efafbb92622bd258624ac2e8967acc82f59ad058f7284",
	"atan2": "20640c4fa01b9696541ad876076311c7a7e465bcbf7bde60cffcf371657a8020",
	"pow":   "2842390f09049f9f78159e877fe1954f16c9b5cbb3a81ab683dee0abab55b77d",
}

func sweepOutputs() map[string][]float64 {
	unary := map[string]func(float64) float64{
		"exp": Exp, "log": Log, "log10": Log10, "sin": Sin, "cos": Cos, "tan": Tan,
		"asin": Asin, "acos": Acos, "atan": Atan,
	}
	out := map[string][]float64{}
	for name, xs := range sweepInputs() {
		f := unary[name]
		vals := make([]float64, len(xs))
		for i, x := range xs {
			vals[i] = f(x)
		}
		out[name] = vals
	}
	for name, f := range map[string]func(float64, float64) float64{"atan2": Atan2, "pow": Pow} {
		args := sweepPairs(name)
		vals := make([]float64, len(args))
		for i, a := range args {
			vals[i] = f(a[0], a[1])
		}
		out[name] = vals
	}
	return out
}

func TestGoldenSweepDigests(t *testing.T) {
	outs := sweepOutputs()
	if os.Getenv("MATHX_PRINT_GOLDEN") != "" {
		for _, name := range []string{"exp", "log", "log10", "sin", "cos", "tan", "asin", "acos", "atan", "atan2", "pow"} {
			t.Logf("%q: %q,", name, digest(outs[name]))
		}
		return
	}
	if len(outs) != 11 {
		t.Fatalf("expected 11 functions in the sweep, got %d", len(outs))
	}
	for name, vals := range outs {
		want, ok := goldenSweepDigests[name]
		if !ok {
			t.Errorf("%s: no golden digest", name)
			continue
		}
		if got := digest(vals); got != want {
			t.Errorf("%s on %s/%s: sweep digest %s, want %s — portable math drifted", name, runtime.GOOS, runtime.GOARCH, got, want)
		}
	}
}

// goldenPoints pins individual inputs to exact output bits so a failure names
// the input. Inputs and outputs are bit patterns, never decimal text.
var goldenPoints = []struct {
	fn      string
	x, y    uint64 // y only for atan2/pow
	wantHex uint64
}{
	{"exp", 0x3fca6d4014d4f0dd, 0, 0x3ff3ab48b88c5dbe},                    // exp(0.2064590551107192) = 1.229317398921793
	{"exp", 0x3ff0000000000000, 0, 0x4005bf0a8b145769},                    // exp(1) = 2.718281828459045
	{"exp", 0xbff0000000000000, 0, 0x3fd78b56362cef38},                    // exp(-1) = 0.36787944117144233
	{"exp", 0x3fe0000000000000, 0, 0x3ffa61298e1e069c},                    // exp(0.5) = 1.6487212707001282
	{"exp", 0x4024000000000000, 0, 0x40d5829dcf950560},                    // exp(10) = 22026.465794806718
	{"exp", 0xc024000000000000, 0, 0x3f07cd79b5647c9a},                    // exp(-10) = 4.539992976248485e-05
	{"exp", 0x4059000000000000, 0, 0x48f3494a9b171bf5},                    // exp(100) = 2.6881171418161356e+43
	{"exp", 0x40862e3d70a3d70a, 0, 0x7fefe9ce5c4c52b4},                    // exp(709.78) = 1.7928227943945155e+308
	{"exp", 0xc087490a3d70a3d7, 0, 0x0000000000000001},                    // exp(-745.13) = 5e-324
	{"exp", 0x3e112e0be826d695, 0, 0x3ff000000044b830},                    // exp(1e-09) = 1.000000001
	{"exp", 0x400d99999999999a, 0, 0x4044394144eeec81},                    // exp(3.7) = 40.4473043600674
	{"exp", 0xbfd3333333333333, 0, 0x3fe7b4c869c37c05},                    // exp(-0.3) = 0.7408182206817179
	{"log", 0x3ff3ab48b88c5dbe, 0, 0x3fca6d4014d4f0da},                    // log(1.229317398921793) = 0.2064590551107191
	{"log", 0x3ff3ab48b88c5dbf, 0, 0x3fca6d4014d4f0e0},                    // log(1.2293173989217931) = 0.20645905511071927
	{"log", 0x4000000000000000, 0, 0x3fe62e42fefa39ef},                    // log(2) = 0.6931471805599453
	{"log", 0x3fe0000000000000, 0, 0xbfe62e42fefa39ef},                    // log(0.5) = -0.6931471805599453
	{"log", 0x4024000000000000, 0, 0x40026bb1bbb55516},                    // log(10) = 2.302585092994046
	{"log", 0x01a56e1fc2f8f359, 0, 0xc085963447f87fb5},                    // log(1e-300) = -690.7755278982137
	{"log", 0x0000000000000001, 0, 0xc0874385446d71c3},                    // log(5e-324) = -744.4400719213812
	{"log", 0x7e37e43c8800759c, 0, 0x4085963447f87fb5},                    // log(1e+300) = 690.7755278982137
	{"log", 0x3feff7ced916872b, 0, 0xbf5064670d979b73},                    // log(0.999) = -0.0010005003335835344
	{"log", 0x401e000000000000, 0, 0x40001e85798eb9a3},                    // log(7.5) = 2.0149030205422647
	{"log", 0x3ff0000000000000, 0, 0x0000000000000000},                    // log(1) = 0
	{"log", 0x0000000000000000, 0, 0xfff0000000000000},                    // log(0) = -Inf
	{"log10", 0x401c000000000000, 0, 0x3feb0b0b0b78cc3f},                  // log10(7) = 0.8450980400142568
	{"log10", 0x4024000000000000, 0, 0x3ff0000000000000},                  // log10(10) = 1
	{"log10", 0x408f400000000000, 0, 0x4008000000000000},                  // log10(1000) = 3
	{"log10", 0x3f50624dd2f1a9fc, 0, 0xc008000000000000},                  // log10(0.001) = -3
	{"log10", 0x4000000000000000, 0, 0x3fd34413509f79ff},                  // log10(2) = 0.3010299956639812
	{"log10", 0x405edd2f1a9fbe77, 0, 0x4000bb6abfc968ef},                  // log10(123.456) = 2.0915122016277716
	{"log10", 0x01a56e1fc2f8f359, 0, 0xc072c00000000000},                  // log10(1e-300) = -300
	{"log10", 0x0000000000000001, 0, 0xc07434e6420f4373},                  // log10(5e-324) = -323.30621534311575
	{"sin", 0x3ff3c083126e978d, 0, 0x3fee351c8409f41d},                    // sin(1.2345) = 0.9439833239445111
	{"sin", 0x3fe0000000000000, 0, 0x3fdeaee8744b05f0},                    // sin(0.5) = 0.479425538604203
	{"sin", 0xbfe0000000000000, 0, 0xbfdeaee8744b05f0},                    // sin(-0.5) = -0.479425538604203
	{"sin", 0x4008000000000000, 0, 0x3fc210386db6d55b},                    // sin(3) = 0.1411200080598672
	{"sin", 0x4059000000000000, 0, 0xbfe03425b78c4db8},                    // sin(100) = -0.5063656411097588
	{"sin", 0x412e848000000000, 0, 0xbfd6664b2568d867},                    // sin(1e+06) = -0.34999350217129294
	{"sin", 0x41c0000000000000, 0, 0x3fd4e67c0e2622df},                    // sin(5.36870912e+08) = 0.3265676630185634
	{"sin", 0x4480f0cf064dd592, 0, 0xbfeb453ab76bf398},                    // sin(1e+22) = -0.8522008497671889
	{"sin", 0x7e37e43c8800759c, 0, 0xbfea2c16b010e386},                    // sin(1e+300) = -0.8178819121159087
	{"sin", 0xc004000000000000, 0, 0xbfe326af0dcfcab0},                    // sin(-2.5) = -0.5984721441039564
	{"sin", 0x401921fb54442d18, 0, 0xbcb1a62633145c00},                    // sin(6.283185307179586) = -2.449293598294703e-16
	{"cos", 0x3ff3c083126e978d, 0, 0x3fd51e9b9f0886ae},                    // cos(1.2345) = 0.32999315767856785
	{"cos", 0x3fe0000000000000, 0, 0x3fec1528065b7d50},                    // cos(0.5) = 0.8775825618903728
	{"cos", 0xbfe0000000000000, 0, 0x3fec1528065b7d50},                    // cos(-0.5) = 0.8775825618903728
	{"cos", 0x4008000000000000, 0, 0xbfefae04be85e5d2},                    // cos(3) = -0.9899924966004454
	{"cos", 0x4059000000000000, 0, 0x3feb981dbf665fdf},                    // cos(100) = 0.8623188722876839
	{"cos", 0x412e848000000000, 0, 0x3fedf9df9906d32c},                    // cos(1e+06) = 0.9367521275331447
	{"cos", 0x41c0000000000000, 0, 0xbfee3edd2dfeef90},                    // cos(5.36870912e+08) = -0.9451738260608966
	{"cos", 0x4480f0cf064dd592, 0, 0x3fe0be2cef01c8f3},                    // cos(1e+22) = 0.5232147853951389
	{"cos", 0x7e37e43c8800759c, 0, 0xbfe2699022adc4c1},                    // cos(1e+300) = -0.5753861119575491
	{"cos", 0xc004000000000000, 0, 0xbfe9a2f7ef858b7d},                    // cos(-2.5) = -0.8011436155469337
	{"cos", 0x3ff921fb54442d18, 0, 0x3c91a62633145c00},                    // cos(1.5707963267948966) = 6.123233995736757e-17
	{"tan", 0x3ff3c083126e978d, 0, 0x4006e28a08810dd4},                    // tan(1.2345) = 2.860614839971939
	{"tan", 0x3fe0000000000000, 0, 0x3fe17b4f5bf3474a},                    // tan(0.5) = 0.5463024898437905
	{"tan", 0xbfe0000000000000, 0, 0xbfe17b4f5bf3474a},                    // tan(-0.5) = -0.5463024898437905
	{"tan", 0x4008000000000000, 0, 0xbfc23ef71254b86f},                    // tan(3) = -0.1425465430742778
	{"tan", 0x4059000000000000, 0, 0xbfe2ca74d62b5d37},                    // tan(100) = -0.587213915156929
	{"tan", 0x412e848000000000, 0, 0xbfd7e9768ab734c0},                    // tan(1e+06) = -0.373624453987599
	{"tan", 0x41c0000000000000, 0, 0xbfd61cd8e0ffa035},                    // tan(5.36870912e+08) = -0.3455106923343039
	{"tan", 0x4480f0cf064dd592, 0, 0xbffa0f79c1b6b259},                    // tan(1e+22) = -1.6287782256068992
	{"tan", 0x7e37e43c8800759c, 0, 0x3ff6be411f37ac76},                    // tan(1e+300) = 1.4214488238747243
	{"tan", 0x3ff921fb54442d18, 0, 0x434d02967c31cdc0},                    // tan(1.5707963267948966) = 1.6331239353195392e+16
	{"tan", 0x3fe921fb54442d18, 0, 0x3ff0000000000000},                    // tan(0.7853981633974483) = 1
	{"asin", 0x3fe0000000000000, 0, 0x3fe0c152382d7366},                   // asin(0.5) = 0.5235987755982989
	{"asin", 0xbfe0000000000000, 0, 0xbfe0c152382d7366},                   // asin(-0.5) = -0.5235987755982989
	{"asin", 0x3fe6666666666666, 0, 0x3fe8d00e692afd95},                   // asin(0.7) = 0.775397496610753
	{"asin", 0x3fe6b851eb851eb8, 0, 0x3fe94391bfac8e4b},                   // asin(0.71) = 0.7894982093461719
	{"asin", 0x3fefae147ae147ae, 0, 0x3ff6de3c6f33d51d},                   // asin(0.99) = 1.4292568534704693
	{"asin", 0xbfefae147ae147ae, 0, 0xbff6de3c6f33d51d},                   // asin(-0.99) = -1.4292568534704693
	{"asin", 0x3fb999999999999a, 0, 0x3fb9a49276037884},                   // asin(0.1) = 0.1001674211615598
	{"asin", 0x3ff0000000000000, 0, 0x3ff921fb54442d18},                   // asin(1) = 1.5707963267948966
	{"asin", 0x3fd5555555555555, 0, 0x3fd5bfe34f051112},                   // asin(0.3333333333333333) = 0.3398369094541219
	{"acos", 0x3fe0000000000000, 0, 0x3ff0c152382d7365},                   // acos(0.5) = 1.0471975511965976
	{"acos", 0xbfe0000000000000, 0, 0x4000c152382d7366},                   // acos(-0.5) = 2.0943951023931957
	{"acos", 0x3fe6666666666666, 0, 0x3fe973e83f5d5c9b},                   // acos(0.7) = 0.7953988301841436
	{"acos", 0x3fe6b851eb851eb8, 0, 0x3fe90064e8dbcbe5},                   // acos(0.71) = 0.7812981174487247
	{"acos", 0x3fefae147ae147ae, 0, 0x3fc21df72882bfd8},                   // acos(0.99) = 0.1415394733244273
	{"acos", 0xbfefae147ae147ae, 0, 0x4008001be1bc011a},                   // acos(-0.99) = 3.0000531802653656
	{"acos", 0x3fb999999999999a, 0, 0x3ff787b22ce3f590},                   // acos(0.1) = 1.4706289056333368
	{"acos", 0x3ff0000000000000, 0, 0x0000000000000000},                   // acos(1) = 0
	{"acos", 0xbff0000000000000, 0, 0x400921fb54442d18},                   // acos(-1) = 3.141592653589793
	{"acos", 0x0000000000000000, 0, 0x3ff921fb54442d18},                   // acos(0) = 1.5707963267948966
	{"atan", 0x3fe0000000000000, 0, 0x3fddac670561bb4f},                   // atan(0.5) = 0.4636476090008061
	{"atan", 0xbfe0000000000000, 0, 0xbfddac670561bb4f},                   // atan(-0.5) = -0.4636476090008061
	{"atan", 0x3fe51eb851eb851f, 0, 0x3fe2aafdde4d0c9f},                   // atan(0.66) = 0.583373006993856
	{"atan", 0x3fe570a3d70a3d71, 0, 0x3fe2e3caf996421e},                   // atan(0.67) = 0.590306746935372
	{"atan", 0x4003333333333333, 0, 0x3ff2d0ead6066395},                   // atan(2.4) = 1.176005207095135
	{"atan", 0x4004000000000000, 0, 0x3ff30b6d796a4da8},                   // atan(2.5) = 1.1902899496825317
	{"atan", 0x4024000000000000, 0, 0x3ff789bd2c160053},                   // atan(10) = 1.4711276743037345
	{"atan", 0x4202a05f20000000, 0, 0x3ff921fb543d4de0},                   // atan(1e+10) = 1.5707963266948965
	{"atan", 0xc008000000000000, 0, 0xbff3fc176b7a8560},                   // atan(-3) = -1.2490457723982544
	{"atan", 0x3fb999999999999a, 0, 0x3fb983e282e2cc4d},                   // atan(0.1) = 0.09966865249116204
	{"atan2", 0x3ff0000000000000, 0x4000000000000000, 0x3fddac670561bb4f}, // atan2(1, 2) = 0.4636476090008061
	{"atan2", 0xbff0000000000000, 0x4000000000000000, 0xbfddac670561bb4f}, // atan2(-1, 2) = -0.4636476090008061
	{"atan2", 0x3ff0000000000000, 0xc000000000000000, 0x40056c6e7397f5ae}, // atan2(1, -2) = 2.677945044588987
	{"atan2", 0xbff0000000000000, 0xc000000000000000, 0xc0056c6e7397f5ae}, // atan2(-1, -2) = -2.677945044588987
	{"atan2", 0x4008000000000000, 0x3fe0000000000000, 0x3ff67d8863bc99bc}, // atan2(3, 0.5) = 1.4056476493802696
	{"atan2", 0x01a56e1fc2f8f359, 0x3ff0000000000000, 0x01a56e1fc2f8f359}, // atan2(1e-300, 1) = 1e-300
	{"atan2", 0x7e37e43c8800759c, 0x01a56e1fc2f8f359, 0x3ff921fb54442d18}, // atan2(1e+300, 1e-300) = 1.5707963267948966
	{"atan2", 0x3fc999999999999a, 0x3fe6666666666666, 0x3fd1cfa95f7a8dcf}, // atan2(0.2, 0.7) = 0.2782996590051114
	{"pow", 0x3ffb333333333333, 0x400a666666666666, 0x40170b0a6f6eccf9},   // pow(1.7, 3.3) = 5.760781994950997
	{"pow", 0x4000000000000000, 0x3fe0000000000000, 0x3ff6a09e667f3bcd},   // pow(2, 0.5) = 1.4142135623730951
	{"pow", 0x4000000000000000, 0x4024000000000000, 0x4090000000000000},   // pow(2, 10) = 1024
	{"pow", 0x4024000000000000, 0xc008000000000000, 0x3f50624dd2f1a9fc},   // pow(10, -3) = 0.001
	{"pow", 0x3ff00068db8bac71, 0x40c3880000000000, 0x4005bec34aabbad1},   // pow(1.0001, 10000) = 2.718145926824356
	{"pow", 0xc000000000000000, 0x4008000000000000, 0xc020000000000000},   // pow(-2, 3) = -8
	{"pow", 0x3fe0000000000000, 0x3ff8000000000000, 0x3fd6a09e667f3bcc},   // pow(0.5, 1.5) = 0.35355339059327373
	{"pow", 0x4008000000000000, 0x3fd5555555555555, 0x3ff7137449123ef6},   // pow(3, 0.3333333333333333) = 1.4422495703074083
	{"pow", 0x3ff8000000000000, 0xc002000000000000, 0x3fd9b3d438ada293},   // pow(1.5, -2.25) = 0.40160089049326436
	{"pow", 0x401c000000000000, 0x3fb999999999999a, 0x3ff36fe0d9dde8a3},   // pow(7, 0.1) = 1.214814044039067
}

func apply(fn string, x, y float64) float64 {
	switch fn {
	case "exp":
		return Exp(x)
	case "log":
		return Log(x)
	case "log10":
		return Log10(x)
	case "sin":
		return Sin(x)
	case "cos":
		return Cos(x)
	case "tan":
		return Tan(x)
	case "asin":
		return Asin(x)
	case "acos":
		return Acos(x)
	case "atan":
		return Atan(x)
	case "atan2":
		return Atan2(x, y)
	case "pow":
		return Pow(x, y)
	}
	panic("unknown fn " + fn)
}

func TestGoldenPoints(t *testing.T) {
	if len(goldenPoints) == 0 {
		t.Fatal("no golden points")
	}
	for _, p := range goldenPoints {
		x, y := math.Float64frombits(p.x), math.Float64frombits(p.y)
		got := math.Float64bits(apply(p.fn, x, y))
		if got != p.wantHex {
			t.Errorf("%s(%v, %v) on %s/%s = %#016x (%v), want %#016x (%v)", p.fn, x, y,
				runtime.GOOS, runtime.GOARCH, got, math.Float64frombits(got), p.wantHex, math.Float64frombits(p.wantHex))
		}
	}
}
