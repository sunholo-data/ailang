# Evaluation: ticket quota:admission-has-no-headroom-margin-and-flaps (iteration 65 ghost claim)

Checkout: /Users/voightkampff/.ailang-driver-pin/fleet @ 58a41bad4 (read-only).

## 1. Per-ask findings

(1) Admission margin: BUILT-DIFFERENTLY.
- internal/mission/quota_margin.go:12-45 defines fixed per-bucket start margins: codex 2, anthropic 1, ollama 3, openrouter 0.60. It has an AILANG_QUOTA_MARGIN_<BUCKET> override. Invalid values fall back to the default. 0 restores the bare test.
- The margin is applied at every bucket evaluator:
  - codex_quota.go:130 and 169-210 (`headroom < margin` sets short, then State="over" with HeadroomShort)
  - anthropic_quota.go:415 (delegates to the Codex evaluator)
  - ollama_quota.go:241 and ollama_rate_ration.go:165
  - openrouter_quota.go:165
- Tests: quota_margin_test.go has per-bucket tests (Codex half-a-point blocked, enough headroom admitted, Anthropic own margin, Ollama 8pp of 10pp blocked, OpenRouter daily and monthly).
- Commit 71eec62b6 message: "A headroom-short bucket blocks like 'over' ... so `mission quota --over` and every driver consumer need no change."

(2) Hysteresis: BUILT-BY-CONSTRUCTION, no dedicated test.
- There is no blocked-state memory. Admission is stateless: `headroom >= margin` is recomputed on every fire (codex_quota.go ~176-186).
- A bucket that was blocked is therefore re-admitted only when pace - used >= margin. That is the ask's semantics. It is not a separate hysteresis band: the same threshold applies on the way in and the way out.
- The ticket's flap came from the pace line creeping over `used`. Under a margin, the line must creep a further margin above `used` before admission, so flapping is reduced to one transition per margin of pace growth.
- It still is a threshold, not a latch. Because a run costs about 1pp and the margin is 2pp, the sequence "admit at headroom 2.0, run, fall to ~1.0, blocked, pace creeps +1pp, admit" cycles with the pace line. That cycle is the intended throttle, not flapping.
- I found no test named for re-admission. The closest are TestCodexMargin_HalfAPointUnderThePaceIsBlocked and ..._EnoughHeadroomIsAdmitted, which pin both sides of the threshold.

(3) In-run cap: NOT BUILT.
- grep of internal/mission/*.go and tools/launchd/mission-control.sh for in-run / per-run cap / 10pp finds nothing relevant. The only "10pp" hits are the ollama daily ration (ollama_rate_ration*.go).
- Quota is read once per fire (`_mc_load_ration`, lane-probe.sh ~192, one `ailang mission quota --over` call). MISSION_OVER_RATION is exported once at mission-control.sh:2302 as a fire-start snapshot. No mid-run check or exit exists.
- So a long run, such as the ticket's iter-217 24pp case, is not stopped.

(4) Fallback rungs and role spawns use the same admission: BUILT, with caveats.
- Call chain, all through one verdict, `ailang mission quota --over` (lane-probe.sh:192, rc via `_mc_bounded`):
  - `_mc_load_ration` fills MC_OVER_RATION from the bare bucket names (lane-probe.sh:194).
  - `_mc_is_over_ration` (lane-probe.sh:278-286) maps a rung to its bucket via `_mc_rung_bucket` (262-276).
  - `_mc_ration_refuses` (298-302) wraps that for controller selection.
- Preferred controller candidates: mission-control.sh:1043 (`_mc_ration_refuses "$m"`).
- Fallback chain rungs: mission-control.sh:1081 (`_mc_ration_refuses "$fb"`).
- Pinned model and override: mission-control.sh:1002 and 1017.
- Lane probes call `_mc_is_over_ration`: lane-probe.sh:67, 130, 148.
- Role spawns: mission-control.sh:2302 exports MISSION_OVER_RATION from `_mc_spawn_over_ration` (lane-probe.sh:310-318). resolve-role-spawn.sh:40 and spawn-pin-hook.sh:69 refuse recipes on those buckets.
- A margin-short bucket is emitted as State="over", so it flows through all of these with no shell change. The commit message says the same.
- Caveats, none of them a margin bypass:
  - `_mc_rung_bucket` returns empty for local ollama and unknown pi:* rungs, which are never rationed. That is by design (lane-probe.sh:268-273).
  - An UNREADABLE Anthropic quota is deliberately admitted to the probe (lane-probe.sh:298-302, 310-318). That is the unreadable-is-not-exhausted rule, not the margin.
  - If the quota command fails, the driver blocks only "codex ollama" (lane-probe.sh ~205-207), not anthropic or openrouter. That is a separate fail-open edge.
  - Admission is per-fire, so the margin is evaluated once, before the controller and spawns.

Attended sessions ungated: TRUE, but by absence rather than by a check.
- quota_margin.go:29 says "Attended sessions are never gated by this." There is no attended or unattended branch in the Go code.
- The gate is consumed only by the launchd driver, through lane-probe.sh and the `--over` seam. Attended sessions do not call it, so the claim holds in effect.
- The comment is mildly misleading in that the code itself is mode-agnostic.

## 2. Per-bucket versus per-mission rolling p75

- It is a real deviation. The ticket asked for rolling per-mission+bucket p75, seeded at 5pp for codex and anthropic.
- Implemented: static per-bucket margins that were measured once as p75 (2026-09-24..10-02) and rounded up. They are about 2.5x smaller for codex and 5x smaller for anthropic than the ticket's 5pp seed.
- Downsides: the margins do not adapt as runs grow or shrink, and a mission that costs more than the margin still overshoots. That is exactly what ask (3) was meant to bound.
- D-FLEET-12 (design_docs/fleet-mission.md:156, RESOLVED, Mark attended 2026-10-02) names the exact margins "codex 2pp, anthropic 1pp, ollama 3pp, openrouter $0.60". So the numbers are covered by a ruling. It does not mention the per-mission or rolling mechanism, and it says nothing on asks (3) or (4). Mark's quoted words in the commit ("only start if we have some headroom") support a simple margin.
- The deviation is covered for the numbers and tacitly accepted for the static design. It is not explicitly ruled on as a replacement for rolling p75.

## 3. Disposition

RESOLVE as a ghost, with caveats stated in the resolution text. Recommended text:
- Cite 71eec62b6 (#1524, "Resolves harness ticket ...") and 134e02999 (#1616, idle-window deadlock fix).
- State that asks (1), (2) and (4) are delivered: static per-bucket margins per D-FLEET-12, stateless same-threshold re-admission, and a single `mission quota --over` verdict that gates every rung and spawn.
- State plainly that ask (3), the in-run pp cap, was NOT built and that no commit or decision row addresses it.
- State that the rolling per-mission p75 was replaced by fixed margins (D-FLEET-12 numbers).

Why not KEEP OPEN: the ticket's primary harm, flapping admission, is fixed. The attended session declared it resolved, and D-FLEET-12 forbids the fleet from re-implementing it.

Why ask (3) still needs a home: it is real, unmitigated spend risk. Do not let it vanish into the resolution text. Raise it as a separate OPEN decision row for Mark, or file a new narrow ticket such as `quota:no-in-run-spend-cap`. The row should offer A (build a cap) and B (drop it, since the margin now bounds only the start). Resolving the old ticket without that follow-up would lose the ask silently. Resolution text alone is not sufficient.

## 4. Score and objections

Score: 86.

Strongest objection: "ghost, resolved" overstates completeness. The ticket had four asks. One is entirely missing (3), one was redesigned (1), and the hysteresis in (2) is a plain threshold with no test that names re-admission. The attended commit's "Resolves" line is not Mark's ruling that (3) is waived. D-FLEET-12 says nothing on (3).

Not reproduced or not checked:
- I did not run the tests.
- I did not read the production inbox message or the full ticket body. I used the quoted text only.
- I did not verify the iteration-65 claim "iter 217's 24pp".
- I did not independently check the Dashboard or ticket resolve CLI behaviour.
- I confirmed that the Go gate has no attended switch. I did not verify the caller graph beyond the launchd scripts.

EVALUATION_VERDICT: PASS 86
