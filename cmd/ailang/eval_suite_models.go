package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// resolveEvalModelList determines the model list for an eval-suite run from the
// model-selection flags (--models / --full / default dev models) and enforces the
// agent-only safety guard. It exits the process on misconfiguration (an agent-only
// model selected without -agent would otherwise degrade silently to empty/garbage
// results — the 2026-05-23 incident: 102 trials, total_tokens=0, provider=ollama).
func resolveEvalModelList(models string, fullSuite, agent bool) []string {
	// Determine model list
	var modelList []string
	if models != "" {
		// User specified models explicitly. Recognize named suites as a
		// single token (e.g. --models agent_suite) and expand them to the
		// composite from models.yml. Otherwise fall back to comma-split.
		modelList = expandModelSuite(models, modelreg.GlobalModelsConfig)
	} else if fullSuite {
		// Full suite: use extended suite (5 models) from models.yml
		if modelreg.GlobalModelsConfig != nil && len(modelreg.GlobalModelsConfig.ExtendedSuite) > 0 {
			modelList = modelreg.GlobalModelsConfig.ExtendedSuite
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-2-codex", "claude-opus-4-6", "claude-sonnet-4-6", "gemini-3-pro", "gemini-2-5-pro"}
		}
	} else {
		// Default: use dev models from models.yml
		if modelreg.GlobalModelsConfig != nil && len(modelreg.GlobalModelsConfig.DevModels) > 0 {
			modelList = modelreg.GlobalModelsConfig.DevModels
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"}
		}
	}
	// SAFETY: every selected model must EXIST in the registry. Nothing downstream
	// checks this, so an unknown name is not an error — it is 1 benchmark x 2
	// languages of instant failures per entry, banked and reported as a result.
	//
	// Measured 2026-09-17 23:55: a run reached the plane as
	// `models: ["--help"]` and published "Eval Suite partial: 0/180 passed
	// (0.0%)" after 53 MILLISECONDS. That notification is indistinguishable from
	// a real measurement of a catastrophically bad model, which is the actual
	// harm — a 0% that nobody can tell from a finding. Go's flag package is how
	// it got there: `--models` takes a value, so `--models --help` binds the
	// FLAG as the VALUE and help never prints.
	//
	// Fail loudly instead (CLAUDE.md #2): a model list is required config, and a
	// wrong one changes what the numbers mean.
	validateEvalModelsExist(modelList)

	// SAFETY: block standard (direct-API) mode only for models that are GENUINELY
	// agent-only — i.e. have an agent_cli AND no cloud standard path (provider is
	// local/CLI-bound, e.g. ollama). Without this guard those degrade silently to
	// junk (the 2026-05-23 incident: 102 trials, total_tokens=0, provider=ollama).
	// Cloud models (anthropic/openai/google/openrouter) that ALSO have an agent_cli
	// — Claude, GPT, Gemini — are dual-mode and run standard natively; they must
	// NOT be blocked (they have hundreds of standard-mode runs across baselines).
	if !agent && modelreg.GlobalModelsConfig != nil {
		var agentOnlyModels []string
		for _, m := range modelList {
			if modelreg.GlobalModelsConfig.SupportsAgentEval(m) &&
				!modelreg.GlobalModelsConfig.SupportsStandardEval(m) {
				cli, _ := modelreg.GlobalModelsConfig.GetAgentCLI(m)
				agentOnlyModels = append(agentOnlyModels, fmt.Sprintf("%s (agent_cli: %q)", m, cli))
			}
		}
		if len(agentOnlyModels) > 0 {
			fmt.Fprintf(os.Stderr, "Error: model(s) require agent mode but -agent was not passed:\n")
			for _, m := range agentOnlyModels {
				fmt.Fprintf(os.Stderr, "  - %s\n", m)
			}
			fmt.Fprintf(os.Stderr, "\nThese models are only usable via their agent CLI; standard-mode runs would silently produce empty/garbage results.\n")
			fmt.Fprintf(os.Stderr, "Re-run with -agent and -benchmarks <list>, e.g.:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite -agent -benchmarks fizzbuzz,adt_option -models %s\n", modelList[0])
			os.Exit(1)
		}
	}
	return modelList
}

// unknownEvalModels returns the entries of modelList that cfg does not carry,
// in the order given. The decision half of validateEvalModelsExist, split out
// because the other half calls os.Exit and cannot be tested in-process.
//
// A nil cfg yields nothing: models.yml failed to load, resolveEvalModelList has
// already fallen back to hardcoded lists, and every name would look unknown.
func unknownEvalModels(modelList []string, cfg *eval_harness.ModelsConfig) []string {
	if cfg == nil {
		return nil
	}
	var unknown []string
	for _, m := range modelList {
		if _, err := cfg.GetModel(m); err != nil {
			unknown = append(unknown, m)
		}
	}
	return unknown
}

// validateEvalModelsExist refuses a run whose model list names anything the
// registry does not carry. Exits 1 on the first bad list rather than returning
// an error, matching the agent-only guard immediately below it — both are
// misconfiguration caught before any trial is dispatched or any row is banked.
//
// Skipped entirely when models.yml did not load: resolveEvalModelList has
// already fallen back to hardcoded lists in that case, and validating those
// against a nil registry would refuse every run.
func validateEvalModelsExist(modelList []string) {
	unknown := unknownEvalModels(modelList, modelreg.GlobalModelsConfig)
	if len(unknown) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "Error: model(s) not in the registry:\n")
	for _, m := range unknown {
		fmt.Fprintf(os.Stderr, "  - %q\n", m)
	}

	// Name the footgun when we can see it. An operator who typed
	// `--models --help` is not helped by "not in the registry" alone: the
	// value they meant to pass is missing and the flag they meant to read was
	// eaten, and neither is visible from the message without this.
	for _, m := range unknown {
		if strings.HasPrefix(m, "-") {
			fmt.Fprintf(os.Stderr,
				"\n%q starts with a dash: it looks like a FLAG that was consumed as the VALUE of\n"+
					"--models. Go's flag package binds the next argument unconditionally, so\n"+
					"`--models --help` sets models to \"--help\" and never prints help.\n"+
					"Check for a missing value or an empty shell variable before --models.\n", m)
			break
		}
	}

	fmt.Fprintf(os.Stderr, "\nSelected models must be registry keys from internal/modelreg/models.yml\n")
	fmt.Fprintf(os.Stderr, "(friendly names such as gpt5-mini, not provider ids). Which registry is\n")
	fmt.Fprintf(os.Stderr, "in use: ailang models source\n")
	fmt.Fprintf(os.Stderr, "\nRefusing to run: an unknown model produces instant per-trial failures that\n")
	fmt.Fprintf(os.Stderr, "bank and report as a 0%% result, which is indistinguishable from a finding.\n")
	os.Exit(1)
}

// clampConcurrencyForSerializedLanes forces --parallel 1 when any selected agent
// lane must run serially. Extracted verbatim from runEvalSuite so that
// eval_suite.go stays under the 800-line ceiling `make check-file-sizes`
// enforces; the decision, its two independent causes, and the operator-facing
// message are unchanged.
func clampConcurrencyForSerializedLanes(agent bool, maxConcurrent *int, modelList []string) {
	// SAFETY (M-RIG single-GPU): agent runs on agent-only / local models (ollama-backed, e.g. qwen)
	// MUST be serial. The rig is ONE GPU; concurrent trials thrash ollama and crash motoko (each
	// produces a 0-byte JSONL -> "terminated without emitting run_summary"). --parallel defaults to
	// 10, so a raw `eval-suite --agent` silently oversubscribes. Clamp to 1 here. Recurring footgun:
	// the dead -agent-parallel flag, the 2026-05-22/23 rotations, and the 2026-06-22 contract_leap_year
	// run that lost 7/8 trials to GPU contention. (Override with an isolated box only.)
	// SERIALIZATION HAS TWO INDEPENDENT CAUSES — conflating them broke a run.
	//
	// (1) GPU contention: the rig is one GPU, and concurrent on-device trials
	//
	//	thrash ollama. Ollama CLOUD rows load nothing on the GPU (measured:
	//	concurrent cloud requests at idle latency while a 45GB model held the
	//	GPU at 100%, `ollama ps` unchanged), so they are exempt — that is D4.
	//
	// (2) motoko's FIXED BACKEND PORT: every motoko profile pins
	//
	//	backend.port 8080 and the harness never varies it, so two motoko runs
	//	collide REGARDLESS OF ROUTE. This has nothing to do with the GPU.
	//
	// The first version of D4 gated only on (1) and let 8 motoko cloud trials
	// start in the same second: 7 crashed at startup ("terminated without
	// emitting run_summary"), 1 survived — the one that started 5s later, after
	// the others died and freed the port. Re-run at --parallel 1, the identical
	// set scored 8/8. The clamp had been protecting against (2) by accident, and
	// removing it traded a false block for a real collision.
	//
	// Until motoko takes a per-run port, ANY motoko row serializes.
	if agent && *maxConcurrent > 1 && modelreg.GlobalModelsConfig != nil {
		for _, m := range modelList {
			gpuBound := modelreg.GlobalModelsConfig.UsesLocalGPU(m)
			cli, _ := modelreg.GlobalModelsConfig.GetAgentCLI(m)
			motokoBound := cli == "motoko"
			if !gpuBound && !motokoBound {
				continue
			}
			// Name WHICH cause fired: they have different fixes and different
			// resume conditions, and a single vague message is how they got
			// conflated in the first place.
			reason := "motoko's fixed backend port 8080 (collides regardless of route)"
			if gpuBound {
				reason = "single-GPU rig contention"
				if motokoBound {
					reason = "single-GPU rig contention + motoko's fixed port 8080"
				}
			}
			if modelreg.GlobalModelsConfig.SupportsAgentEval(m) && !modelreg.GlobalModelsConfig.SupportsStandardEval(m) {
				fmt.Fprintf(os.Stderr, "\u26a0 Serializing agent run \u2014 forcing --parallel 1 (was %d): %s.\n", *maxConcurrent, reason)
				*maxConcurrent = 1
				break
			}
		}
	}
}
