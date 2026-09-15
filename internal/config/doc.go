// Package config is the one place AILANG resolves its cloud identity — which
// GCP project and region a process acts on — and the one place a deprecated
// production default is allowed to live while it is being retired.
//
// # Cloud project
//
// CloudProject resolves, in order, and stops at the first non-empty answer:
//
//  1. AILANG_CLOUD_PROJECT
//  2. GOOGLE_CLOUD_PROJECT (what Cloud Run, GKE and App Engine set)
//  3. the pubsub.project_id key of ~/.ailang/config.yaml (or the file named by
//     AILANG_CONFIG). Only that one key is read here; the full config loader
//     is a later Phase 2.1 step and does not belong in a leaf.
//  4. the GCE metadata server, with a short timeout, unless AILANG_NO_METADATA
//     is set. The outcome is cached for the life of the process, so a laptop
//     pays the timeout once, not per call.
//
// When nothing resolves it returns ErrNoCloudProject. There is no default:
// acting on a guessed project looks exactly like success, and the gcloud CLI's
// current project is deliberately not consulted because it points at whatever
// the operator last worked on. GCP_PROJECT, an older alias some sites read,
// is no longer honoured.
//
// # Region
//
// Region resolves AILANG_CLOUD_REGION, then GOOGLE_CLOUD_REGION, then falls
// through to DeprecatedDefault with the value the fleet was hard-coded to.
//
// # Deprecated defaults (ruling D3, 2026-09-15)
//
// Production defaults such as "ailang-multivac" and "europe-west1" become a
// deprecation warning now and a hard error in v1.0.0. DeprecatedDefault is the
// only sanctioned way to keep using one: it returns the value and writes ONE
// warning per process per variable to stderr, naming what to set. With
// AILANG_STRICT_CONFIG=1 it refuses instead and returns an error wrapping
// ErrDeprecatedDefault — the v1.0.0 behaviour, testable today.
//
// # The rule
//
// This package is the ONLY place that reads AILANG_CLOUD_PROJECT,
// GOOGLE_CLOUD_PROJECT, AILANG_CLOUD_REGION, GOOGLE_CLOUD_REGION,
// AILANG_STRICT_CONFIG, AILANG_NO_METADATA and AILANG_CONFIG. Every other
// package calls in. A forbidigo rule will enforce that in Phase 2.14; until
// then tools/simplicity_metrics.sh counts the leaks as getenv_outside_config.
//
// It is a leaf — standard library plus gopkg.in/yaml.v3 — so the store
// implementations, the AI providers and cmd/ailang can all import it.
// leaf_test.go enforces that.
package config
