# Downstream compatibility gate (design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md,
# usage and exit codes: tools/downstream/README.md). Runs `ailang check` over every corpus in
# tools/downstream/corpora.yaml with a baseline and a candidate binary and reports only the
# difference. Exit 0 = PASS; 2 = unacknowledged/over-max break in a blocking corpus, or an unused
# expected-breaks entry; 3 = the instrument could not measure; 4 = a blocking corpus has
# NOT_MEASURED entries beyond its allow_not_measured; 64 = usage.
#
# The tool is built, then run: `go run` reports every non-zero exit as 1, which
# would erase the difference between a break, an instrument failure and an
# incomplete measurement.

DOWNSTREAM_BIN := bin/downstream

.PHONY: downstream-check test-downstream

downstream-check: ## Baseline vs CANDIDATE=<binary> over every downstream corpus; reports only the difference (BASELINE=latest|vX.Y.Z|<bin> TIER=release|ci CORPORA=a,b REF=default|pinned|worktree OUT= DOWNSTREAM_STATE= OFFLINE=0|1 JOBS= TIMEOUT=)
	@test -n "$(CANDIDATE)" || { echo "usage: make downstream-check CANDIDATE=<ailang binary> [BASELINE=latest|vX.Y.Z|<binary>] [TIER=release|ci] [CORPORA=a,b] [REF=default|pinned|worktree] [OUT=<dir>] [DOWNSTREAM_STATE=<dir>] [OFFLINE=0|1] [JOBS=n] [TIMEOUT=s]" >&2; exit 64; }
	@go build -o $(DOWNSTREAM_BIN) ./tools/downstream
	@$(DOWNSTREAM_BIN) --candidate "$(CANDIDATE)" \
		--baseline "$(or $(BASELINE),latest)" --tier "$(or $(TIER),release)" --ref "$(or $(REF),default)" \
		$(if $(CORPORA),--corpora "$(CORPORA)") $(if $(OUT),--out "$(OUT)") \
		$(if $(DOWNSTREAM_STATE),--state "$(DOWNSTREAM_STATE)") --offline="$(or $(OFFLINE),0)" $(if $(JOBS),--jobs $(JOBS)) $(if $(TIMEOUT),--timeout $(TIMEOUT))

test-downstream: ## The downstream gate's own controls (null control, planted break, expected-breaks, not-measured, timeouts, loud failures)
	@go test -count=1 ./tools/downstream/
