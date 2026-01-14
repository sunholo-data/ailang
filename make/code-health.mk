# =============================================================================
# CODE HEALTH & ORGANIZATION TARGETS
# =============================================================================

.PHONY: check-file-sizes report-file-sizes codebase-health largest-files
.PHONY: fmt fmt-check vet lint install-lint

# Code formatting
fmt: ## Format all Go code
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

fmt-check: ## Check code formatting (CI gate)
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "$(RED)$(CROSS) Go code is not formatted. Run 'make fmt'$(RESET)"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "$(GREEN)$(CHECKMARK) Code formatting check passed$(RESET)"

# Go vet
vet: prepare-embed ## Run go vet
	@echo "Running go vet..."
	$(GOVET) $(shell go list ./... | grep -v examples/agents)
	@echo "Vet complete"

# Linting
install-lint: ## Install golangci-lint v2.x
	@echo "Installing golangci-lint v2.x..."
	@CURRENT_VERSION=$$(golangci-lint --version 2>/dev/null | grep -o 'v[0-9]*' | head -1 || echo "none"); \
	if [ "$$CURRENT_VERSION" != "v2" ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.1.6; \
	else \
		echo "golangci-lint v2.x already installed"; \
	fi

lint: prepare-embed ## Run linter (bug detectors only)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-lint'" && exit 1)
	@golangci-lint run ./cmd/... ./internal/... ./testutil/... 2>&1 | \
		grep -v "(related information)" | \
		grep -v "QF[0-9]" | \
		grep -v "ST[0-9]" | \
		grep -v "SA1019:" | \
		grep -v "SA9003:" | \
		grep -v "SA5011:" | \
		grep -v "SA5012:" | \
		grep -v "is unused" | \
		grep -v "^\t" | \
		grep -v "^[[:space:]]*\^" | \
		tee /tmp/lint.out || true
	@if grep -qE "^(internal|cmd|testutil)" /tmp/lint.out; then \
		echo "$(RED)$(CROSS) Lint errors found$(RESET)"; \
		exit 1; \
	fi
	@echo "$(GREEN)$(CHECKMARK) Lint complete (no bugs detected)$(RESET)"

# File size checks (AI-friendly codebase)
check-file-sizes: ## Check for files >800 lines (CI gate)
	@echo "Checking for files >800 lines..."
	@FOUND=0; \
	for file in $$(find internal cmd -name "*.go"); do \
		SIZE=$$(wc -l < "$$file"); \
		if [ $$SIZE -gt 800 ]; then \
			echo "$(RED)$(CROSS) $$file: $$SIZE lines (exceeds 800)$(RESET)"; \
			FOUND=1; \
		fi; \
	done; \
	if [ $$FOUND -eq 1 ]; then \
		echo "$(YELLOW)$(WARNING) Files exceed 800 line limit. Split for AI maintainability.$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)$(CHECKMARK) All files within 800 line limit$(RESET)"; \
	fi

report-file-sizes: ## Report all files >500 lines
	@echo "$(BOLD)=== File Size Report ===$(RESET)"
	@echo ""
	@echo "$(BOLD)CRITICAL (>800 lines):$(RESET)"
	@CRITICAL=0; \
	find internal cmd -name "*.go" -exec wc -l {} \; | sort -rn | while read SIZE FILE; do \
		if [ $$SIZE -gt 800 ]; then \
			echo "$(RED)$(WARNING) $$FILE: $$SIZE lines$(RESET)"; \
		fi; \
	done
	@echo ""
	@echo "$(BOLD)WARNING (500-800 lines):$(RESET)"
	@find internal cmd -name "*.go" -exec wc -l {} \; | sort -rn | while read SIZE FILE; do \
		if [ $$SIZE -gt 500 ] && [ $$SIZE -le 800 ]; then \
			echo "$(YELLOW)$(WARNING) $$FILE: $$SIZE lines$(RESET)"; \
		fi; \
	done
	@echo ""
	@CRITICAL=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 > 800 {count++} END {print count+0}'); \
	WARNING=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 > 500 && $$1 <= 800 {count++} END {print count+0}'); \
	echo "Summary: $$CRITICAL files >800 lines, $$WARNING files 500-800 lines"

codebase-health: ## Full codebase health metrics
	@echo "$(BOLD)=== Codebase Health Report ===$(RESET)"
	@echo ""
	@echo "$(BOLD)File Size Metrics:$(RESET)"
	@TOTAL=$$(find internal cmd -name "*.go" | wc -l | tr -d ' '); \
	SUM=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '{sum += $$1} END {print sum}'); \
	AVG=$$(echo "$$SUM / $$TOTAL" | bc); \
	echo "  Total files: $$TOTAL"; \
	echo "  Total lines: $$SUM"; \
	echo "  Average size: $$AVG lines/file"
	@echo ""
	@echo "$(BOLD)Distribution:$(RESET)"
	@SMALL=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 <= 500 {count++} END {print count+0}'); \
	MEDIUM=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 > 500 && $$1 <= 800 {count++} END {print count+0}'); \
	LARGE=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 > 800 {count++} END {print count+0}'); \
	echo "  $(GREEN)<=500 lines (good):$(RESET) $$SMALL files"; \
	echo "  $(YELLOW)500-800 lines:$(RESET) $$MEDIUM files"; \
	echo "  $(RED)>800 lines (split):$(RESET) $$LARGE files"
	@echo ""
	@LARGE=$$(find internal cmd -name "*.go" -exec wc -l {} \; | awk '$$1 > 800 {count++} END {print count+0}'); \
	if [ $$LARGE -eq 0 ]; then \
		echo "$(GREEN)$(CHECKMARK) Codebase is AI-friendly$(RESET)"; \
	else \
		echo "$(YELLOW)$(WARNING) $$LARGE files need splitting$(RESET)"; \
	fi

largest-files: ## Show 20 largest files
	@echo "$(BOLD)=== 20 Largest Files ===$(RESET)"
	@find internal cmd -name "*.go" -exec wc -l {} \; | sort -rn | head -20 | \
		awk '{printf "%4d lines: %s\n", $$1, $$2}'
