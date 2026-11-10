# Downstream compatibility gate (design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md).
# Runs `ailang check` over every corpus in tools/downstream/corpora.yaml with a
# baseline and a candidate binary and reports only the difference. Exit 0 = no
# unacknowledged break; 2 = unacknowledged/over-max break in a blocking corpus;
# 3 = the instrument could not measure (never a silent skip).

.PHONY: downstream-check test-downstream

downstream-check: ## Baseline vs CANDIDATE=<binary> over every downstream corpus; reports only the difference (BASELINE=latest|vX.Y.Z|<bin> TIER=release|ci CORPORA=a,b REF=default|pinned|worktree OUT= DOWNSTREAM_STATE= OFFLINE=1)
	@test -n "$(CANDIDATE)" || { echo "usage: make downstream-check CANDIDATE=<ailang binary> [BASELINE=latest|vX.Y.Z|<binary>] [TIER=release|ci] [CORPORA=a,b] [REF=default|pinned|worktree] [OUT=<dir>] [DOWNSTREAM_STATE=<dir>] [OFFLINE=1]" >&2; exit 64; }
	@go run ./tools/downstream --candidate "$(CANDIDATE)" \
		--baseline "$(or $(BASELINE),latest)" --tier "$(or $(TIER),release)" --ref "$(or $(REF),default)" \
		$(if $(CORPORA),--corpora "$(CORPORA)") $(if $(OUT),--out "$(OUT)") \
		$(if $(DOWNSTREAM_STATE),--state "$(DOWNSTREAM_STATE)") $(if $(OFFLINE),--offline) $(if $(JOBS),--jobs $(JOBS))

test-downstream: ## The downstream gate's own controls (null control, planted break, expected-breaks, loud failures)
	@go test -count=1 ./tools/downstream/
