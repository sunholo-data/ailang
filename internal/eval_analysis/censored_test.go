package eval_analysis

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func TestFmtDriverScheduleSatisfiesOrderIntegrity(t *testing.T) {
	// DECLARED NARROWING. The artifact under test is produced by the launchd driver, which is a
	// POSIX shell script executed by /bin/bash 3.2 on the rig and by the macOS leg in CI; there is
	// no /bin/bash on windows, so this arm asserts nothing there. Same convention as the repo's
	// other bash-mock skips (e.g. internal/executor/motoko/motoko_test.go:189). Caught by Gate 3b
	// on the windows leg -- `exec: "/bin/bash": executable file not found in %PATH%` -- which no
	// darwin-only command could have surfaced.
	if runtime.GOOS == "windows" {
		t.Skip("driver schedule fixture requires POSIX shell (/bin/bash)")
	}
	repoRoot := filepath.Join("..", "..")
	logPath := filepath.Join(t.TempDir(), "invocations.log")

	// DECLARE THIS TEST'S REAL INPUTS TO THE GO TEST CACHE.
	// The observable here is produced by a SHELL subprocess reading two files outside this package,
	// and `go test` keys its cache on files the TEST PROCESS opens -- it cannot see a subprocess's
	// opens. So without these reads, editing nightly-eval.sh leaves the cached verdict in place and
	// `go test` re-prints the PRE-EDIT result while reporting success: a mutation drill on the
	// driver then reads green for a mutant that was never executed. Measured first-party during the
	// M3 drill: the ON-always-leads mutant returned rc=0 from a cache hit and rc=1
	// (order_integrity_lead_not_alternating) under -count=1. Reading the files here makes the cache
	// key track them, so the stale green is unreachable rather than merely documented.
	for _, input := range []string{
		filepath.Join(repoRoot, "tools", "launchd", "nightly-eval.sh"),
		filepath.Join(repoRoot, "tools", "launchd", "test_fmt_ab_schedule.sh"),
	} {
		if _, err := os.ReadFile(input); err != nil {
			t.Fatalf("instrument failure: cannot read declared cache input %s: %v", input, err)
		}
	}
	// EMIT-ONLY on purpose. The shell harness also asserts the schedule itself, and if this test
	// ran the full suite then a wrong schedule would red those assertions first -- the process would
	// exit non-zero and CheckFmtOrderIntegrity below would never execute. The analyzer call would
	// then be decorative for exactly the defects AC-M3-4 cites it to catch. Measured: under the
	// ON-always-leads mutant the full-suite form died at "shell schedule fixture failed", not at the
	// order-integrity assertion. In emit-only mode the shell writes the driver's raw invocation log
	// and exits 0 regardless of order, so the analyzer is the discriminator.
	cmd := exec.Command("/bin/bash", filepath.Join(repoRoot, "tools", "launchd", "test_fmt_ab_schedule.sh"))
	cmd.Env = append(os.Environ(), "FMT_AB_EMIT_LOG="+logPath, "FMT_AB_EMIT_ONLY=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shell schedule fixture failed to EMIT (this is an instrument failure, not a schedule verdict): %v\n%s", err, output)
	}

	file, err := os.Open(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var on, off []*BenchmarkResult
	scanner := bufio.NewScanner(file)
	stamp := time.Unix(1, 0)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		var benchmark, model string
		for i := 0; i+1 < len(fields); i++ {
			switch fields[i] {
			case "--benchmarks":
				benchmark = fields[i+1]
			case "--models":
				model = fields[i+1]
			}
		}
		row := &BenchmarkResult{ID: benchmark, Timestamp: stamp}
		stamp = stamp.Add(time.Second)
		if model == "motoko-local-qwen3-6-fmt" {
			on = append(on, row)
		} else {
			off = append(off, row)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(on)+len(off) == 0 {
		t.Fatal("instrument failure: shell artifact contained zero invocations")
	}
	// The emit-only artifact is the pure D1b measurement schedule: 6 benchmarks x 2 arms.
	// Asserting the count here is what stops a truncated or contaminated artifact (e.g. one that
	// also carried the M5 smoke row) from being read as a clean schedule.
	if got := len(on) + len(off); got != 12 {
		t.Fatalf("instrument failure: shell artifact has %d invocations, want 12", got)
	}
	if reason := CheckFmtOrderIntegrity(on, off); reason != "" {
		t.Fatalf("CheckFmtOrderIntegrity rejected shell artifact: %s", reason)
	}
}

type censoredFixture struct {
	Name         string `json:"name"`
	Pairs        int    `json:"pairs"`
	OnTokenWins  int    `json:"on_token_wins"`
	OffTokenWins int    `json:"off_token_wins"`
	OnOnlyPasses int    `json:"on_only_passes"`
	WantVerdict  string `json:"want_verdict"`
	WantReason   string `json:"want_reason"`
}

func TestCensoredVerdictMatrix(t *testing.T) {
	data, err := os.ReadFile("testdata/censored_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []censoredFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			on, off := fixtureArms(fixture)
			got := AnalyzeCensoredPairs(on, off)
			if got.Verdict != fixture.WantVerdict || got.Reason != fixture.WantReason {
				t.Fatalf("(verdict, reason) = (%s, %s), want (%s, %s); result=%+v", got.Verdict, got.Reason, fixture.WantVerdict, fixture.WantReason, got)
			}
			if got.NEff != fixture.OnTokenWins+fixture.OffTokenWins+fixture.OnOnlyPasses {
				t.Fatalf("n_eff = %d, want %d", got.NEff, fixture.OnTokenWins+fixture.OffTokenWins+fixture.OnOnlyPasses)
			}
		})
	}
}

func TestD2VoidTreatmentOnRate(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 6, OnTokenWins: 6})
	for _, index := range []int{0, 2} {
		on[index].Validity = &eval_harness.Validity{Valid: false, Reason: eval_harness.ReasonTreatmentUnproven}
	}
	got := AnalyzeCensoredPairs(on, off)
	assertVoidReason(t, got, "treatment_unproven_rate")
}

func TestD2VoidTreatmentOffContamination(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 2, OnTokenWins: 2})
	off[1].FmtHookEvents = []eval_harness.FmtHookEvent{{Status: "formatted", File: "x.ail"}}
	got := AnalyzeCensoredPairs(on, off)
	assertVoidReason(t, got, "control_contaminated")
}

func TestD2CensorOnePassWins(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 10, OnOnlyPasses: 10})
	got := AnalyzeCensoredPairs(on, off)
	if got.NEff != 10 || got.OnWins != 10 || got.Verdict != CensoredVerdictInconclusive || got.Reason != "insufficient-neff" {
		t.Fatalf("(n_eff, W, verdict, reason) = (%d, %d, %s, %s)", got.NEff, got.OnWins, got.Verdict, got.Reason)
	}
}

func TestD2MarginMakesNearRatiosTies(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 12})
	for i := range on {
		on[i].TotalTokens = 950
		off[i].TotalTokens = 1000
	}
	got := AnalyzeCensoredPairs(on, off)
	if got.NEff != 0 || got.Ties != 12 {
		t.Fatalf("(n_eff, ties) = (%d, %d), want (0, 12)", got.NEff, got.Ties)
	}
}

func TestD2KeepRequiresSignTest(t *testing.T) {
	got := AnalyzeCensoredPairs(fixtureArms(censoredFixture{Pairs: 40, OnTokenWins: 24, OffTokenWins: 16}))
	if got.Verdict == CensoredVerdictKeep || got.SignPValue <= 0.05 || got.MedianTokenRatio > 0.90 {
		t.Fatalf("sign-test control did not discriminate: %+v", got)
	}
}

func TestD2KeepRequiresMedianRatio(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 40, OnOnlyPasses: 40})
	got := AnalyzeCensoredPairs(on, off)
	if got.Verdict == CensoredVerdictKeep || got.SignPValue > 0.05 || got.MedianTokenRatio != 0 {
		t.Fatalf("median-ratio control did not discriminate: %+v", got)
	}
}

func TestD2KeepRequiresMcNemarGuardrail(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 40, OnTokenWins: 30, OffTokenWins: 10})
	for i := 30; i < 40; i++ {
		on[i].CompileOk, on[i].RuntimeOk, on[i].StdoutOk = false, false, false
	}
	got := AnalyzeCensoredPairs(on, off)
	if got.Verdict == CensoredVerdictKeep || got.SignPValue > 0.05 || got.MedianTokenRatio > 0.90 || !got.PassRateLoss {
		t.Fatalf("McNemar control did not discriminate: %+v", got)
	}
}

func TestD2OrderRefusalBranches(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assertOrderReason(t, nil, nil, "order_integrity_empty")
	})
	t.Run("timestamp", func(t *testing.T) {
		on, off := fixtureArms(censoredFixture{Pairs: 2, OnTokenWins: 2})
		on[0].Timestamp = time.Time{}
		assertOrderReason(t, on, off, "order_integrity_timestamp")
	})
	t.Run("noncontiguous", func(t *testing.T) {
		on, off := fixtureArms(censoredFixture{Pairs: 2, OnTokenWins: 2})
		duplicate := *on[0]
		duplicate.Trial = 2
		duplicate.Timestamp = off[0].Timestamp.Add(500 * time.Millisecond)
		on = append(on, &duplicate)
		assertOrderReason(t, on, off, "order_integrity_noncontiguous_block")
	})
	t.Run("unpaired-block", func(t *testing.T) {
		on, off := fixtureArms(censoredFixture{Pairs: 2, OnTokenWins: 2})
		assertOrderReason(t, on, off[:1], "order_integrity_unpaired_block")
	})
	t.Run("nonadjacent", func(t *testing.T) {
		on, off := fixtureArms(censoredFixture{Pairs: 2, OnTokenWins: 2})
		off[0].Timestamp = on[1].Timestamp.Add(500 * time.Millisecond)
		assertOrderReason(t, on, off, "order_integrity_nonadjacent_arms")
	})
	t.Run("lead-not-alternating", func(t *testing.T) {
		on, off := fixtureArms(censoredFixture{Pairs: 3, OnTokenWins: 3})
		on[1].Timestamp, off[1].Timestamp = off[1].Timestamp, on[1].Timestamp
		assertOrderReason(t, on, off, "order_integrity_lead_not_alternating")
	})
}

func assertOrderReason(t *testing.T, on, off []*BenchmarkResult, want string) {
	t.Helper()
	if got := CheckFmtOrderIntegrity(on, off); got != want {
		t.Fatalf("reason = %q, want %q", got, want)
	}
}

func TestD2VoidOrderPerfectArmBlock(t *testing.T) {
	on, off := fixtureArms(censoredFixture{Pairs: 6, OnTokenWins: 6})
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	for i := range on {
		on[i].Timestamp = base.Add(time.Duration(i) * time.Second)
		off[i].Timestamp = base.Add(time.Duration(10+i) * time.Second)
	}
	assertVoidReason(t, AnalyzeCensoredPairs(on, off), "order_integrity_nonadjacent_arms")
}

func assertVoidReason(t *testing.T, got CensoredPairResult, reason string) {
	t.Helper()
	if got.Verdict != CensoredVerdictVoid || got.VoidReason != reason || got.Reason != reason {
		t.Fatalf("(verdict, void_reason, reason) = (%s, %s, %s), want (VOID, %s, %s)", got.Verdict, got.VoidReason, got.Reason, reason, reason)
	}
	if got.NEff != 0 || got.OnWins != 0 || got.OffWins != 0 {
		t.Fatalf("VOID leaked numeric evidence: %+v", got)
	}
}

func fixtureArms(fixture censoredFixture) ([]*BenchmarkResult, []*BenchmarkResult) {
	base := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	on := make([]*BenchmarkResult, fixture.Pairs)
	off := make([]*BenchmarkResult, fixture.Pairs)
	for i := 0; i < fixture.Pairs; i++ {
		onPass, offPass := true, true
		onTokens, offTokens := 1000, 1000
		switch {
		case i < fixture.OnOnlyPasses:
			offPass = false
		case i < fixture.OnOnlyPasses+fixture.OnTokenWins:
			onTokens = 800
		case i < fixture.OnOnlyPasses+fixture.OnTokenWins+fixture.OffTokenWins:
			onTokens = 1200
		}
		on[i] = fixtureRow(i, onPass, onTokens)
		off[i] = fixtureRow(i, offPass, offTokens)
		pairStart := base.Add(time.Duration(i*4) * time.Second)
		if i%2 == 0 {
			on[i].Timestamp, off[i].Timestamp = pairStart, pairStart.Add(time.Second)
		} else {
			off[i].Timestamp, on[i].Timestamp = pairStart, pairStart.Add(time.Second)
		}
	}
	return on, off
}

func fixtureRow(index int, pass bool, tokens int) *BenchmarkResult {
	return &BenchmarkResult{
		ID: "bench-" + string(rune('a'+index)), Lang: "ailang", Model: "fixture", Trial: 1,
		CompileOk: pass, RuntimeOk: pass, StdoutOk: pass, TotalTokens: tokens,
	}
}
