package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-TRACE-LABEL-AWARE M3: the canary test whose absence let std/secret's
// promise drift from the truth. A value labelled <secret> must not reach an
// emitted trace at ANY tier, by any path the IFC pass propagates a label
// through: a labelling function, a labelled parameter, JSON encode/decode,
// getString, a tail-recursive loop, a closure capture, a higher-order map,
// string interpolation, an effect argument, and a labelled TAIL call made
// from an unlabelled frame (fetchAndUse declassifies, so its own call site is
// clean and only the tail call's scope covers useToken).
//
// The value below is an obvious fake; it is never a real credential.
const traceLabelCanary = "CANARY_FAKE_NOT_A_REAL_SECRET_42"

const traceLabelSource = `module trace_label

import std/json (encode, decode, getString, jo, kv, js)
import std/string (substring)
import std/list (map, length)
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)
import std/fs (writeFile)

func classify(raw: string) -> string<secret> ! {} = raw

func postFormBody(token: string<secret>) -> string ! {} =
  encode(jo([kv("access_token", js(token)), kv("token_type", js("Bearer"))]))

func tokenFrom(body: string) -> string ! {} = match decode(body) {
  Ok(j) => match getString(j, "access_token") {
    Some(t) => t,
    None => ""
  },
  Err(e) => e
}

func spin(n: int, acc: string) -> string ! {} = if n == 0 then acc else spin(n - 1, acc)

func plainWork(n: int) -> int ! {} = n + 1

func useToken(s: string<secret>, n: int) -> int ! {Declassify} = n + 1

func fetchAndUse(n: int) -> int ! {Declassify} = useToken(classify("CANARY_FAKE_NOT_A_REAL_SECRET_42"), n)

export func main() -> () ! {IO, FS, Declassify} {
  let warm = plainWork(fetchAndUse(40))
  let token = classify("CANARY_FAKE_NOT_A_REAL_SECRET_42")
  let body = postFormBody(token)
  let back = tokenFrom(body)
  let looped = spin(3, back)
  let firsts = map(\s. substring(s, 0, 6), [looped])
  let wrap = \u. looped
  writeFile("OUT_PATH", wrap(0))
  println("prefix: ${substring(looped, 0, 6)} of ${show(length(firsts))} after ${show(warm)}")
}
`

// runTraceLabelFixture runs the fixture with a collector at tier and returns
// what the program wrote to its output file, and the recorded events.
// engine is "eval", "bytecode" or "strict".
func runTraceLabelFixture(t *testing.T, tier trace.Tier, engine string) (string, []trace.TraceEvent, error) {
	t.Helper()
	return runTraceFixture(t, "trace_label", traceLabelSource, tier, engine, nil)
}

// runTraceFixture compiles source as module name (OUT_PATH replaced by a temp
// file), grants FS and Declassify, applies setup, and runs main.
func runTraceFixture(t *testing.T, name, source string, tier trace.Tier, engine string, setup func(*effects.EffContext)) (string, []trace.TraceEvent, error) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	dir := t.TempDir()
	file := filepath.Join(dir, name+".ail")
	// ToSlash: a raw Windows path (C:\Users\...) inside an AILANG string literal is an illegal \U escape.
	src := strings.Replace(source, "OUT_PATH", filepath.ToSlash(filepath.Join(dir, "out.txt")), 1)

	var out bytes.Buffer
	rt, res := terminalFixtureRuntime(t, file, src, strings.NewReader(""), &out)
	ctx := rt.GetEvaluator().GetEffContext().(*effects.EffContext)
	ctx.Grant(effects.Capability{Name: "FS"})
	ctx.Grant(effects.Capability{Name: "Declassify"})
	if setup != nil {
		setup(ctx)
	}
	collector := trace.NewCollectorWithTier(tier)
	ctx.Trace = collector

	err = ExecuteModuleEntrypoint(rt, ModuleExecParams{
		Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
		NoPrint: true, Quiet: true, PipelineResult: &res,
		BytecodeMode: engine != "eval", StrictBytecode: engine == "strict",
	})
	written, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
	return string(written), collector.Events(), err
}

func TestTraceNeverCarriesALabelledValue(t *testing.T) {
	for _, tier := range []trace.Tier{trace.TierStandard, trace.TierDeep} {
		t.Run(tier.String(), func(t *testing.T) {
			written, events, err := runTraceLabelFixture(t, tier, "eval")
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			// The program itself is unchanged: redaction applies to the trace only.
			if written != traceLabelCanary {
				t.Fatalf("program wrote %q, want the value itself", written)
			}
			raw, err := json.Marshal(events)
			if err != nil {
				t.Fatal(err)
			}
			if n := strings.Count(string(raw), traceLabelCanary); n > 0 {
				t.Fatalf("%d verbatim copies of a <secret> value in the %s trace:\n%s", n, tier, raw)
			}
			// Firing controls: the trace must still have structure, and a call
			// that carries no label must still record its values — or the
			// absence above proves nothing.
			var sawWrite, sawPlain bool
			for _, ev := range events {
				if ev.Effect != nil && ev.Effect.OpName == "writeFile" {
					sawWrite = true
					if len(ev.Effect.Args) != 2 || !strings.HasPrefix(ev.Effect.Args[1], "<redacted:secret>") {
						t.Errorf("writeFile effect args = %q, want arity kept and the content redacted", ev.Effect.Args)
					}
				}
				if ev.Function != nil && ev.Function.Name == "plainWork" && ev.Event == trace.EventFunctionEnter {
					sawPlain = true
					if len(ev.Function.Args) != 1 || ev.Function.Args[0] != "41" {
						t.Errorf("plainWork(41) recorded args %q; an unlabelled call must keep its values", ev.Function.Args)
					}
				}
			}
			if !sawWrite {
				t.Errorf("no writeFile effect event at %s; the effect surface was not exercised", tier)
			}
			if tier == trace.TierDeep && !sawPlain {
				t.Error("no plainWork function event at deep; the function surface was not exercised")
			}
		})
	}
}

// The bytecode VM does not carry trace-redaction sites, so a traced run of a
// program with labelled values must not run on it (--bytecode falls back to
// the evaluator; --strict-bytecode fails, see vmTraceLabelGuard).
func TestTraceLabelRedactionCoversBytecodeMode(t *testing.T) {
	for _, tier := range []trace.Tier{trace.TierStandard, trace.TierDeep} {
		t.Run(tier.String(), func(t *testing.T) {
			written, events, err := runTraceLabelFixture(t, tier, "bytecode")
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if written != traceLabelCanary {
				t.Fatalf("program wrote %q, want the value itself", written)
			}
			raw, _ := json.Marshal(events)
			if n := strings.Count(string(raw), traceLabelCanary); n > 0 {
				t.Fatalf("%d verbatim copies of a <secret> value in the --bytecode %s trace:\n%s", n, tier, raw)
			}
		})
	}
}

func TestTraceLabelRedactionStrictBytecodeFailsLoudly(t *testing.T) {
	_, events, err := runTraceLabelFixture(t, trace.TierStandard, "strict")
	if err == nil || !strings.Contains(err.Error(), "M-TRACE-LABEL-AWARE") {
		raw, _ := json.Marshal(events)
		t.Fatalf("--strict-bytecode on a traced labelled program: err = %v, want the M-TRACE-LABEL-AWARE refusal; trace:\n%s", err, raw)
	}
}

// fakeSecretResolver stands in for 1Password; it returns the fake canary.
type fakeSecretResolver struct{}

func (fakeSecretResolver) Read(context.Context, string) (string, error) {
	return traceLabelCanary, nil
}

// The guarantee std/secret documents: a value resolved by secret() does not
// reach the trace, through the wrapper's own frame or any call it flows into.
const secretTraceSource = `module secret_trace

import std/secret (secret)
import std/string (substring)
import std/fs (writeFile)

func bearer(token: string) -> string ! {} = "Bearer ${token}"

export func main() -> () ! {IO, FS, Secret} {
  let key = secret("op://Fake/canary/credential", "trace redaction test")
  let header = bearer(key)
  writeFile("OUT_PATH", substring(header, 7, 39))
}
`

func TestStdSecretValueNeverReachesTheTrace(t *testing.T) {
	for _, tier := range []trace.Tier{trace.TierStandard, trace.TierDeep} {
		t.Run(tier.String(), func(t *testing.T) {
			written, events, err := runTraceFixture(t, "secret_trace", secretTraceSource, tier, "eval", func(ctx *effects.EffContext) {
				ctx.Grant(effects.Capability{Name: "Secret"})
				ctx.Secret = &effects.SecretContext{Resolver: fakeSecretResolver{}}
			})
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if written != traceLabelCanary {
				t.Fatalf("program wrote %q, want the resolved value itself", written)
			}
			raw, _ := json.Marshal(events)
			if n := strings.Count(string(raw), traceLabelCanary); n > 0 {
				t.Fatalf("%d verbatim copies of a secret() value in the %s trace:\n%s", n, tier, raw)
			}
			if !strings.Contains(string(raw), `"writeFile"`) {
				t.Fatalf("control: no writeFile effect in the %s trace, so the effect surface was not exercised:\n%s", tier, raw)
			}
		})
	}
}
