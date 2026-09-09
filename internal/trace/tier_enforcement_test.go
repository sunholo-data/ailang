package trace

import "testing"

// M-TRACE-TIER-NOT-ENFORCED.
//
// The tier was resolved in cmd/ailang, used to decide whether a collector
// existed and which banner to print, and then never reached the collector —
// which had no tier field at all. So `standard` and `deep` recorded identically,
// while options.go documented the opposite ("NOT per-call function spans").
//
// The cost of that gap was measured on a 400-iteration accumulator loop: 2059 MB
// peak RSS at the default tier versus 106 MB with tracing off, because every call
// serialised its arguments and results — for `concat(acc, [x])`, the whole
// accumulator, twice per iteration, giving O(n^2) trace bytes. The same recording
// is the widest value-retention surface in the runtime (fb_b023726953f2ee5a).
//
// This matrix is the test whose absence let prose drift from behavior. It asserts
// what each tier records, so the next divergence is a build failure.
func TestTierGovernsWhatIsRecorded(t *testing.T) {
	// exercise drives one of everything a collector can be asked to record.
	exercise := func(c *Collector) {
		c.RecordModuleStart("m", nil)
		c.RecordFunctionEnter("f", []string{"secret-ish argument"})
		c.RecordFunctionExit("f", "secret-ish result")
		c.RecordModuleEnd("m", 0)
	}

	cases := []struct {
		name        string
		collector   func() *Collector
		wantFuncs   bool
		wantModules bool
	}{
		{
			name:        "standard omits per-call function events",
			collector:   func() *Collector { return NewCollectorWithTier(TierStandard) },
			wantFuncs:   false,
			wantModules: true,
		},
		{
			name:        "deep records them — that is its purpose",
			collector:   func() *Collector { return NewCollectorWithTier(TierDeep) },
			wantFuncs:   true,
			wantModules: true,
		},
		{
			name:        "NewCollector defaults to deep, so existing callers are unchanged",
			collector:   NewCollector,
			wantFuncs:   true,
			wantModules: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.collector()
			exercise(c)

			var gotFuncs, gotModules bool
			for _, e := range c.Events() {
				switch e.Event {
				case EventFunctionEnter, EventFunctionExit:
					gotFuncs = true
				case EventModuleStart, EventModuleEnd:
					gotModules = true
				}
			}
			if gotFuncs != tc.wantFuncs {
				t.Errorf("function events recorded = %v, want %v", gotFuncs, tc.wantFuncs)
			}
			// Structural events must survive at every tier: the span tree is the
			// audit value, and it is not the part that leaks or grows.
			if gotModules != tc.wantModules {
				t.Errorf("module events recorded = %v, want %v", gotModules, tc.wantModules)
			}
		})
	}
}

// TestStandardTierRetainsNoArgumentValues is the security-facing half: it is not
// enough that fewer events are emitted, the VALUES must not be retained anywhere
// in the collector.
func TestStandardTierRetainsNoArgumentValues(t *testing.T) {
	const canary = "CANARY_ya29_SECRET"
	c := NewCollectorWithTier(TierStandard)
	c.RecordFunctionEnter("exchangeRefresh", []string{canary})
	c.RecordFunctionExit("exchangeRefresh", canary)

	for _, e := range c.Events() {
		if e.Function == nil {
			continue
		}
		for _, a := range e.Function.Args {
			if a == canary {
				t.Errorf("standard tier retained an argument value verbatim: %q", a)
			}
		}
		if e.Function.Result == canary {
			t.Errorf("standard tier retained a result value verbatim: %q", e.Function.Result)
		}
	}
}

// TestRecordsFunctionCalls lets the evaluator skip RENDERING arguments it would
// only throw away — the rendering is where the O(n^2) cost is actually paid.
func TestRecordsFunctionCalls(t *testing.T) {
	if NewCollectorWithTier(TierStandard).RecordsFunctionCalls() {
		t.Error("standard reports that it records function calls; the evaluator would render arguments for nothing")
	}
	if !NewCollectorWithTier(TierDeep).RecordsFunctionCalls() {
		t.Error("deep reports that it does not record function calls; per-call spans would be lost")
	}
}
