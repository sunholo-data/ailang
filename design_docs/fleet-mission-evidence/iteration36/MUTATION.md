# Mutation proof — IN-SANDBOX

Every mutant is applied to the real file after a scratch copy, then restored from that copy in a finally block. Each test uses -count=1.

## M1 — IN-SANDBOX

Mutation: `"%v at byte %d of %d (window [%d:%d]: %q)", err, syntax.Offset, len(b), a, z, b[a:z]` → `"%v of %d (window [%d:%d]: %q)", err, len(b), a, z, b[a:z]` in `internal/mission/quorum/salvage.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageDiagnostics/offset$`

Red exit: 1; assertion: `salvage_test.go:23: F1: error reviewer returned non-JSON or malformed response: invalid character '|' in string escape code of 108 (window [0:108]: "{\"verdict\":\"reject\",\"strongest_objection\":\"the table row a \\| b is unescaped\",\"catch\":\"x\",\"proposed_fix\":\"\"}") lacks "at byte 61"`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.317s`

Full output: `mut/M1-red.log`, `mut/M1-green.log`.

## M4b — IN-SANDBOX

Mutation: `case '{', '[':
` → `case '{', '[':
				depth = 0
				continue
` in `internal/mission/quorum/salvage.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageVerdict/after_nested_object$`

Red exit: 1; assertion: `salvage_test.go:127: salvage = <nil>, want verdict=reject objection="after nested"`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.318s`

Full output: `mut/M4b-red.log`, `mut/M4b-green.log`.

M5b first attempt produced an unused-variable compile error and is NOT counted as mutation proof. The corrected mutant below preserves compilation while removing key/value discrimination.

## M5b — IN-SANDBOX

Mutation: `if expectKey {` → `if (expectKey || !expectKey) && (token == "verdict" || token == "strongest_objection") {` in `internal/mission/quorum/salvage.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageVerdict/key_value_state$`

Red exit: 1; assertion: `salvage_test.go:127: salvage = &{Verdict:reject StrongestObjection: Offset:62}, want verdict=pass objection=""`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.318s`

Full output: `mut/M5b-red.log`, `mut/M5b-green.log`.

## M6 — IN-SANDBOX

Mutation: `foldAbsent(&s, o, o.Model)` → `s.AbsentReviewers = append(s.AbsentReviewers, AbsentNote{Model: o.Model, Reason: o.AbsentReason})` in `internal/mission/quorum/quorum.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageSynthesis/reject$`

Red exit: 1; assertion: `salvage_test.go:161: salvaged reject not folded: {Verdict:proceed BlockingObjections:[] AbsentReviewers:[{Model:B Reason:invalid SalvagedVerdict:}] TotalCostUSD:0.02 TotalTokensIn:100 TotalTokensOut:20}, want objection "B [salvaged from invalid response]: real objection"`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.332s`

Full output: `mut/M6-red.log`, `mut/M6-green.log`.

## M8 — IN-SANDBOX

Mutation: `recordInvalid(out, raw)` → `out.AbsentReason = ReasonInvalid` in `internal/mission/quorum/run.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageProduction/plain$`

Red exit: 1; assertion: `salvage_test.go:212: invalid outcome lost salvage or became present: &{Model:B CostUSD:0.00014000000000000001 TokensIn:100 TokensOut:20 Present:false AbsentReason:invalid Err:reviewer returned non-JSON or malformed response: invalid character 'G' after object key:value pair at byte 94 of 130 (window [0:130]: "{\"verdict\":\"reject\",\"strongest_objection\":\"asserts \\\"GOOS\\\" absent\",\"catch\":\"the test greps \"GOARCH\" in ci.yml\",\"proposed_fix\":\"\"}") Result:<nil> Salvaged:<nil> Landed:<nil> Tier:}`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.319s`

Full output: `mut/M8-red.log`, `mut/M8-green.log`.

## M8b — IN-SANDBOX

Mutation: `recordInvalid(out, raw)` → `out.AbsentReason = ReasonInvalid
		if resp.FinishReason != "length" { recordInvalid(out, raw) }` in `internal/mission/quorum/run.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageProduction/truncated$`

Red exit: 1; assertion: `salvage_test.go:212: invalid outcome lost salvage or became present: &{Model:B CostUSD:0.00014000000000000001 TokensIn:100 TokensOut:20 Present:false AbsentReason:invalid Err:reviewer output truncated at 20 tokens (finish_reason=length) — reasoning exhausted the model's declared max_output_tokens (0); raise it in models.yml. Parse error: reviewer returned non-JSON or malformed response: unexpected end of JSON input at byte 59 of 59 (window [0:59]: "{\"verdict\":\"reject\",\"strongest_objection\":\"cut off mid-sent") Result:<nil> Salvaged:<nil> Landed:<nil> Tier:}`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.397s`

Full output: `mut/M8b-red.log`, `mut/M8b-green.log`.

## M8c — IN-SANDBOX

Mutation: `recordInvalid(out, raw)` → `out.AbsentReason = ReasonInvalid
		if salvage := salvageVerdict(raw); salvage != nil && salvage.Offset != -1 { recordInvalid(out, raw) }` in `internal/mission/quorum/run.go`.

Command: `go test ./internal/mission/quorum/ -count=1 -v -run ^TestSalvageProduction/gate_violation$`

Red exit: 1; assertion: `salvage_test.go:212: invalid outcome lost salvage or became present: &{Model:B CostUSD:0.00014000000000000001 TokensIn:100 TokensOut:20 Present:false AbsentReason:invalid Err:reviewer verdict "reject" has empty strongest_objection (reject-by-default requires a stated objection; a missing one is not a pass) Result:<nil> Salvaged:<nil> Landed:<nil> Tier:}`

Restored exit: 0; control: `ok  	github.com/sunholo-data/ailang/internal/mission/quorum	0.329s`

Full output: `mut/M8c-red.log`, `mut/M8c-green.log`.

