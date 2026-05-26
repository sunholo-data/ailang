# M-COORD-MULTI-HOST-WORKERS (v0.22.0): coordinator-daemon install/uninstall
# targets that wrap tools/launchd/install_coordinator.sh.
#
# Usage:
#   make coord-install                                         # default config
#   make coord-install TAGS="ollama:gemma4-26b-ailang,gpu:m4-max"
#   make coord-install TAGS="ollama:gemma4-26b-ailang" HOST_ID=studio.eval-rig
#   make coord-install-dry                                     # show planned changes
#   make coord-uninstall                                       # remove daemon
#   make coord-status                                          # quick check
#
# The script is the source of truth — these targets are thin wrappers
# that pass through TAGS/HOST_ID/CLOUD_PROJECT environment-style args.

COORD_INSTALLER := tools/launchd/install_coordinator.sh

# Optional knobs (overridable on the command line):
TAGS         ?=
HOST_ID      ?=
CLOUD_PROJECT ?=

# Compose the optional flags lazily so an empty TAGS doesn't produce `--tags ""`
COORD_FLAGS :=
ifneq ($(strip $(TAGS)),)
COORD_FLAGS += --tags $(TAGS)
endif
ifneq ($(strip $(HOST_ID)),)
COORD_FLAGS += --host-id $(HOST_ID)
endif
ifneq ($(strip $(CLOUD_PROJECT)),)
COORD_FLAGS += --cloud-project $(CLOUD_PROJECT)
endif

.PHONY: coord-install coord-install-dry coord-uninstall coord-status

coord-install: ## Install the coordinator daemon as a launchd LaunchAgent
	@$(COORD_INSTALLER) $(COORD_FLAGS)

coord-install-dry: ## Preview what coord-install would do without applying
	@$(COORD_INSTALLER) --dry-run $(COORD_FLAGS)

coord-uninstall: ## Unload and remove the coordinator launchd LaunchAgent
	@$(COORD_INSTALLER) --uninstall

coord-status: ## Show coordinator daemon status (launchd + ailang)
	@echo "=== launchd job ==="
	@launchctl list 2>/dev/null | grep dev.ailang.coordinator || echo "  (not loaded)"
	@echo ""
	@echo "=== ailang coordinator status ==="
	@ailang coordinator status 2>&1 | head -10
