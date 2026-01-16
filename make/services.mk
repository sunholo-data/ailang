# =============================================================================
# SERVICE MANAGEMENT TARGETS
# =============================================================================

.PHONY: repl run serve serve-bg
.PHONY: coordinator-start coordinator-stop coordinator-status
.PHONY: services-start services-stop services-restart services-status
.PHONY: ui-deploy

# REPL & Run
repl: build ## Start the AILANG REPL
	@$(BUILD_DIR)/$(BINARY) repl

run: build ## Run an AILANG file (FILE=path/to/file.ail)
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make run FILE=path/to/file.ail"; \
		exit 1; \
	fi
	@$(BUILD_DIR)/$(BINARY) run $(FILE)

# Server
# AILANG_DASHBOARD=1 enables AILANG transforms (event_formatter.ail, heatmap.ail, budget_checker.ail)
# AILANG_PROJECT_ROOT tells the embed.Engine where to find the .ail files
AILANG_ENV := AILANG_DASHBOARD=1 AILANG_PROJECT_ROOT="$(CURDIR)"

serve: quick-install ## Start Collaboration Hub server (foreground)
	@echo "Starting AILANG Collaboration Hub..."
	@echo "  AILANG transforms: enabled"
	@$(AILANG_ENV) ailang serve

serve-bg: quick-install ## Start server in background
	@if curl -s http://127.0.0.1:1957/health >/dev/null 2>&1; then \
		echo "$(GREEN)$(CHECKMARK) Server already running on port 1957$(RESET)"; \
	else \
		echo "Starting AILANG server in background..."; \
		echo "  AILANG transforms: enabled"; \
		$(AILANG_ENV) nohup ailang serve > ~/.ailang/logs/server.log 2>&1 & \
		sleep 2; \
		if curl -s http://127.0.0.1:1957/health >/dev/null 2>&1; then \
			echo "$(GREEN)$(CHECKMARK) Server started$(RESET)"; \
		else \
			echo "$(RED)$(CROSS) Server failed to start. Check ~/.ailang/logs/server.log$(RESET)"; \
		fi \
	fi

# Coordinator
coordinator-start: quick-install ## Start coordinator daemon
	@echo "Starting coordinator daemon..."
	@ailang coordinator start

coordinator-stop: ## Stop coordinator daemon
	@echo "Stopping coordinator daemon..."
	@ailang coordinator stop || echo "Coordinator not running"

coordinator-status: ## Check coordinator status
	@ailang coordinator status

# Combined services
services-start: serve-bg ## Start both server + coordinator
	@sleep 1
	@if ailang coordinator status 2>/dev/null | grep -q "running"; then \
		echo "$(GREEN)$(CHECKMARK) Coordinator already running$(RESET)"; \
	else \
		echo "Starting coordinator daemon..."; \
		nohup ailang coordinator start > /dev/null 2>&1 & \
		sleep 3; \
		ailang coordinator status; \
	fi
	@echo ""
	@echo "$(GREEN)$(CHECKMARK) Services started:$(RESET)"
	@echo "  - Server: http://127.0.0.1:1957"
	@echo "  - Coordinator: running (check with 'make coordinator-status')"

services-stop: coordinator-stop ## Stop all services
	@echo "Stopping server..."
	@pkill -f "ailang serve" 2>/dev/null || echo "Server not running"
	@echo "$(GREEN)$(CHECKMARK) Services stopped$(RESET)"

services-restart: services-stop ## Rebuild and restart all services
	@echo "Rebuilding..."
	@$(MAKE) quick-install
	@echo ""
	@$(MAKE) services-start

services-status: ## Show status of all services
	@echo "$(BOLD)AILANG Services Status$(RESET)"
	@echo "$(LINE)"
	@echo ""
	@echo "$(BOLD)Server:$(RESET)"
	@if curl -s http://127.0.0.1:1957/health >/dev/null 2>&1; then \
		curl -s http://127.0.0.1:1957/health | jq -r '"  Status: healthy\n  Connections: \(.connections)\n  Version: \(.version)"'; \
	else \
		echo "  Status: not running"; \
	fi
	@echo ""
	@echo "$(BOLD)Coordinator:$(RESET)"
	@ailang coordinator status 2>/dev/null || echo "  Status: not running"

# UI deployment
ui-deploy: ## Build and deploy UI to server
	@echo "Building UI..."
	@cd ui && npm run build
	@echo "Cleaning old assets..."
	@rm -rf internal/server/dist/assets/*
	@echo "Deploying to server..."
	@cp -r ui/dist/* internal/server/dist/
	@echo "$(GREEN)$(CHECKMARK) UI deployed ($$(ls internal/server/dist/assets | wc -l | tr -d ' ') assets)$(RESET)"
