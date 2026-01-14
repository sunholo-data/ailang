# =============================================================================
# CLAUDE CLI TARGETS
# =============================================================================

.PHONY: setup-claude update-claude check-claude test-claude-headless

setup-claude: ## Install Claude CLI globally for headless mode
	@echo "Installing Claude CLI globally..."
	@npm install -g @anthropic-ai/claude-code
	@echo "$(GREEN)$(CHECKMARK) Claude CLI installed$(RESET)"
	@claude --version

update-claude: ## Update Claude CLI to latest version
	@echo "Checking for Claude CLI updates..."
	@CURRENT=$$(claude --version 2>/dev/null | grep -o '[0-9.]*' | head -1); \
	LATEST=$$(npm view @anthropic-ai/claude-code version); \
	if [ -z "$$CURRENT" ]; then \
		echo "$(RED)$(CROSS) Claude CLI not installed. Run: make setup-claude$(RESET)"; \
		exit 1; \
	fi; \
	echo "Current version: $$CURRENT"; \
	echo "Latest version:  $$LATEST"; \
	if [ "$$CURRENT" = "$$LATEST" ]; then \
		echo "$(GREEN)$(CHECKMARK) Already up to date!$(RESET)"; \
	else \
		npm install -g @anthropic-ai/claude-code@latest; \
		echo "$(GREEN)$(CHECKMARK) Updated to $$(claude --version)$(RESET)"; \
	fi

check-claude: ## Verify Claude CLI is installed and working
	@command -v claude >/dev/null 2>&1 || { \
		echo "$(RED)$(CROSS) Claude CLI not found$(RESET)"; \
		echo "Install with: make setup-claude"; \
		exit 1; \
	}
	@echo "$(GREEN)$(CHECKMARK) Claude CLI found: $$(which claude)$(RESET)"
	@echo "$(GREEN)$(CHECKMARK) Version: $$(claude --version)$(RESET)"

test-claude-headless: check-claude ## Test Claude headless mode
	@echo "Testing Claude headless mode..."
	@OUTPUT=$$(claude -p "echo test" --output-format json 2>&1); \
	if echo "$$OUTPUT" | jq -e '.subtype == "success"' >/dev/null 2>&1; then \
		echo "$(GREEN)$(CHECKMARK) Claude headless mode working$(RESET)"; \
	else \
		echo "$(RED)$(CROSS) Claude headless mode failed$(RESET)"; \
		exit 1; \
	fi
