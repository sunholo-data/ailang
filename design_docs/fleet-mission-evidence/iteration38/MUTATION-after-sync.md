| arm | mutant | red? | restored green? |
|---|---|---|---|
| M1-default | `return ArtifactDir, nil } → return filepath.Join(os.TempDir(), ArtifactDir), nil }` | rc=1 | rc=0 |
| M1-constant | `const ArtifactDir = ".ailang/state/mission-quorum" → const ArtifactDir = ".ailang/state/quorum"` | rc=1 | rc=0 |
| M1-spaces | `return override, nil → return ArtifactDir, nil` | rc=1 | rc=0 |
| M1-explicit | `if dir != "" && dir != ArtifactDir { → if false {` | rc=1 | rc=0 |
| M1-invalid | `return "", err 		} 		return override → return ArtifactDir, nil 		} 		return override` | rc=1 | rc=0 |
| M1-nul | `if strings.ContainsRune(dir, '\x00') { → if false && strings.ContainsRune(dir, '\x00') {` | rc=1 | rc=0 |
| M1-destination | `return "", fmt.Errorf("create artifact dir: %w", err) → return "", nil` | rc=1 | rc=0 |
| M1-whitespace | `return get(EnvAgentPolicy) != "" → return strings.TrimSpace(get(EnvAgentPolicy)) != ""` | rc=1 | rc=0 |
| M1-host-absolute | `if config.AgentPolicyPresentRaw() { → if false {` | rc=1 | rc=0 |
| M1-host-invalid | `if config.AgentPolicyPresentRaw() { → if false {` | rc=1 | rc=0 |
| M1-diagnostic | `fmt.Fprintf(os.Stderr, "quorum artifact override suppressed: policy marker present; effective local directory: %s\n", effective) → _ = effective` | rc=1 | rc=0 |
| M1-deny-control | `if config.AgentPolicyPresentRaw() { → if false {` | rc=1 | rc=0 |
| M1-cwd | `return "", fmt.Errorf("resolve confined artifact dir: %w", err) → return ArtifactDir, nil` | rc=1 | rc=0 |
| M1-registry | `{EnvMissionQuorumArtifactDir, "", AreaMission, "Opt-in absolute quorum evidence directory; unset uses the current repo local directory. Raw nonempty AILANG_AGENT_POLICY suppresses default overrides for confinement; distinct explicit artifact paths retain precedence."}, → ` | rc=1 | rc=0 |
| M1-raw-getter | `return get(EnvMissionQuorumArtifactDir) → return strings.TrimSpace(get(EnvMissionQuorumArtifactDir))` | rc=1 | rc=0 |
| M1-collision | `os.O_CREATE\|os.O_EXCL\|os.O_WRONLY → os.O_CREATE\|os.O_TRUNC\|os.O_WRONLY` | rc=1 | rc=0 |
| M2-handoff-clean | `ExtraEnv: map[string]string{ → ExtraEnv: map[string]string{             config.EnvMissionQuorumArtifactDir: "",` | rc=1 | rc=0 |
| M2-handoff-inherited_marker | `ExtraEnv: map[string]string{ → ExtraEnv: map[string]string{             config.EnvMissionQuorumArtifactDir: "",` | rc=1 | rc=0 |
| M2-handoff-task_policy | `ExtraEnv: map[string]string{ → ExtraEnv: map[string]string{             config.EnvMissionQuorumArtifactDir: "",` | rc=1 | rc=0 |
| M2-cross_tree_durable (1) | `MC_QUORUM_DIRS=("$MISSION_QUORUM_ARTIFACT_DIR") → MC_QUORUM_DIRS=("$root/.ailang/state/mission-quorum")` | rc=1 | rc=0  |
| M2-repo_namespace_missions (1) | `repo=${MISSION_REPO:-} → repo=${MISSION_NAME:-}` | rc=1 | rc=0  |
| M2-repo_namespace_isolation (1) | `case "$repo" in → case "sunholo-data/ailang" in` | rc=1 | rc=0  |
| M2-invalid_repo_and_home (1) | `mc_quorum_repo_valid "$repo" \|\| → : \|\|` | rc=1 | rc=0  |
| M2-invalid_repo_and_home (1) | `mc_quorum_absolute "${HOME:-}" \|\| → : \|\|` | rc=1 | rc=0  |
| M2-invalid_and_explicit_override (1) | `mc_quorum_absolute "$MISSION_QUORUM_ARTIFACT_DIR" \|\| → : \|\|` | rc=1 | rc=0  |
| M2-gate2_extracted_active_matrix (1) | `if ! . "$helper"; then echo 'quorum ERROR: cannot source artifact helper' >&2; return 2; fi → . "$helper"` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-lookup_error_matrix (1) | `if [ "$bad" = 1 ]; then return 2; fi → if [ "$bad" = 1 ]; then return 1; fi` | rc=1 | rc=0  |
| M2-extraction_is_nonvacuous (1) | `# --- QUORUM READER START --- → ` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-driver_export_before_both_launches (1) | `export MISSION_QUORUM_ARTIFACT_DIR → :` | rc=1 | rc=0  |
| M2-codex_actual_forwarding (1) | `mc_codex_env_args 2>>"$LOG" → MC_CODEX_ENV_ARGS=()` | rc=1 | rc=0  |
| M2-contaminated_host_startup (1) | `if [ -n "${AILANG_AGENT_POLICY:-}" ]; then → if false; then` | rc=1 | rc=0  |
| M2-standalone_policy_reader (2) | `if [ -n "${AILANG_AGENT_POLICY:-}" ]; then → if false; then` | rc=1 | rc=0  |
| M2-bash32_and_suite_reach (1) | `@$(LAUNCHD_SUITE) tools/launchd/test_quorum_artifacts.sh → ` | rc=1 | rc=0  |
| M2-repo_namespace_missions (1) | `repo=${MISSION_REPO:-} → repo=${MISSION_REPO,,}` | rc=1 | rc=0  |
| M2-attended_no_exports (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → MC_QUORUM_DIRS+=("$HOME/.ailang/state/mission-quorum")` | rc=1 | rc=0  |
| M2-attended_absent_and_errors (1) | `helper="$root/tools/launchd/lib/quorum-artifacts.sh" → echo missing-driver-root >&2; return 2` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-cross_mode_fixed_order (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → :` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-cross_mode_fixed_order (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum"); if [ "${#MC_QUORUM_DIRS[@]}" = 2 ]; then MC_QUORUM_DIRS=("${MC_QUORUM_DIRS[1]}" "${MC_QUORUM_DIRS[0]}"); fi` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-cross_mode_errors_dominate (1) | `MC_QUORUM_MATCHES+=("$file") → MC_QUORUM_MATCHES+=("$file"); return 0` | rc=1 | rc=0  |
| M2-generated_docs_and_mirror (1) | `{EnvMissionQuorumArtifactDir, "", AreaMission, "Opt-in absolute quorum evidence directory; unset uses the current repo local directory. Raw nonempty AILANG_AGENT_POLICY suppresses default overrides for confinement; distinct explicit artifact paths retain precedence."}, → ` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-generated_docs_and_mirror (1) | `# --- QUORUM READER START --- → # --- MUTANT READER START ---` | rc=1 | rc=1 UNINFORMATIVE UNDER SANDBOX: protected mirror could not sync |
| M2-fixture_home_and_observations (1) | `MISSION_QUORUM_ARTIFACT_DIR="$HOME/.ailang/state/mission-quorum" → MISSION_QUORUM_ARTIFACT_DIR="/nonexistent/quorum-mutation-home"` | rc=1 | rc=0  |
| M2-cross_tree_durable (1) | `MC_QUORUM_DIRS=("$MISSION_QUORUM_ARTIFACT_DIR") → MC_QUORUM_DIRS=("$root/.ailang/state/mission-quorum")` | rc=1 | rc=0  |
| M2-repo_namespace_missions (1) | `repo=${MISSION_REPO:-} → repo=${MISSION_NAME:-}` | rc=1 | rc=0  |
| M2-repo_namespace_isolation (1) | `case "$repo" in → case "sunholo-data/ailang" in` | rc=1 | rc=0  |
| M2-invalid_repo_and_home (1) | `mc_quorum_repo_valid "$repo" || → : ||` | rc=1 | rc=0  |
| M2-invalid_repo_and_home (1) | `mc_quorum_absolute "${HOME:-}" || → : ||` | rc=1 | rc=0  |
| M2-invalid_and_explicit_override (1) | `mc_quorum_absolute "$MISSION_QUORUM_ARTIFACT_DIR" || → : ||` | rc=1 | rc=0  |
| M2-gate2_extracted_active_matrix (1) | `if ! . "$helper"; then echo 'quorum ERROR: cannot source artifact helper' >&2; return 2; fi → . "$helper"` | rc=1 | rc=0  |
| M2-lookup_error_matrix (1) | `if [ "$bad" = 1 ]; then return 2; fi → if [ "$bad" = 1 ]; then return 1; fi` | rc=1 | rc=0  |
| M2-extraction_is_nonvacuous (1) | `# --- QUORUM READER START --- → ` | rc=1 | rc=0  |
| M2-driver_export_before_both_launches (1) | `export MISSION_QUORUM_ARTIFACT_DIR → :` | rc=1 | rc=0  |
| M2-codex_actual_forwarding (1) | `mc_codex_env_args 2>>"$LOG" → MC_CODEX_ENV_ARGS=()` | rc=1 | rc=0  |
| M2-contaminated_host_startup (1) | `if [ -n "${AILANG_AGENT_POLICY:-}" ]; then → if false; then` | rc=1 | rc=0  |
| M2-standalone_policy_reader (2) | `if [ -n "${AILANG_AGENT_POLICY:-}" ]; then → if false; then` | rc=1 | rc=0  |
| M2-bash32_and_suite_reach (1) | `@$(LAUNCHD_SUITE) tools/launchd/test_quorum_artifacts.sh → ` | rc=1 | rc=0  |
| M2-repo_namespace_missions (1) | `repo=${MISSION_REPO:-} → repo=${MISSION_REPO,,}` | rc=1 | rc=0  |
| M2-attended_no_exports (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → MC_QUORUM_DIRS+=("$HOME/.ailang/state/mission-quorum")` | rc=1 | rc=0  |
| M2-attended_absent_and_errors (1) | `helper="$root/tools/launchd/lib/quorum-artifacts.sh" → echo missing-driver-root >&2; return 2` | rc=1 | rc=0  |
| M2-cross_mode_fixed_order (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → :` | rc=1 | rc=0  |
| M2-cross_mode_fixed_order (1) | `MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum") → MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum"); if [ "${#MC_QUORUM_DIRS[@]}" = 2 ]; then MC_QUORUM_DIRS=("${MC_QUORUM_DIRS[1]}" "${MC_QUORUM_DIRS[0]}"); fi` | rc=1 | rc=0  |
| M2-cross_mode_errors_dominate (1) | `MC_QUORUM_MATCHES+=("$file") → MC_QUORUM_MATCHES+=("$file"); return 0` | rc=1 | rc=0  |
| M2-generated_docs_and_mirror (1) | `{EnvMissionQuorumArtifactDir, "", AreaMission, "Opt-in absolute quorum evidence directory; unset uses the current repo local directory. Raw nonempty AILANG_AGENT_POLICY suppresses default overrides for confinement; distinct explicit artifact paths retain precedence."}, → ` | rc=1 | rc=0  |
| M2-generated_docs_and_mirror (1) | `# --- QUORUM READER START --- → # --- MUTANT READER START ---` | rc=1 | rc=0  |
| M2-fixture_home_and_observations (1) | `MISSION_QUORUM_ARTIFACT_DIR="$HOME/.ailang/state/mission-quorum" → MISSION_QUORUM_ARTIFACT_DIR="/nonexistent/quorum-mutation-home"` | rc=1 | rc=0  |
