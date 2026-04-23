// In-flight package build tracking (M-PKG-INFLIGHT).
//
// Mirrors each publish attempt to a Firestore `package_builds` collection and
// emits PackageEvents on the events topic so the laptop daemon / dashboard can
// observe work that is in progress — not just what has already landed in the
// registry index.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

// packageBuildsCollection is the Firestore collection name for build tracking.
const packageBuildsCollection = "package_builds"

// buildTracker records package build progression in Firestore and publishes
// PackageEvents to Pub/Sub. Both backends are optional: if neither is
// configured the tracker is a no-op, so the validator still works in local /
// dry-run / test modes.
type buildTracker struct {
	fs        *firestore.Client // nil if Firestore is not configured
	publisher *pubsub.Publisher // nil if Pub/Sub is not configured
	projectID string
}

// newBuildTracker wires up Firestore and Pub/Sub from the ambient environment.
// Firestore uses GOOGLE_CLOUD_PROJECT (Cloud Run injects this); Pub/Sub uses
// AILANG_CLOUD_PROJECT if set, otherwise GOOGLE_CLOUD_PROJECT. Either backend
// failing to initialize is logged but non-fatal — the validator must continue
// to serve publish requests even if visibility is degraded.
func newBuildTracker(ctx context.Context) *buildTracker {
	t := &buildTracker{}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Printf("build-tracker: GOOGLE_CLOUD_PROJECT not set — package_builds disabled")
		return t
	}
	t.projectID = projectID

	fsClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("build-tracker: Firestore init failed (%v) — package_builds docs disabled", err)
	} else {
		t.fs = fsClient
	}

	pubsubProject := os.Getenv("AILANG_CLOUD_PROJECT")
	if pubsubProject == "" {
		pubsubProject = projectID
	}
	prefix := os.Getenv("AILANG_PUBSUB_PREFIX")
	psClient, err := pubsub.NewClient(ctx, pubsubProject, prefix)
	if err != nil {
		log.Printf("build-tracker: Pub/Sub init failed (%v) — package events disabled", err)
	} else {
		t.publisher = pubsub.NewPublisher(psClient)
	}

	return t
}

// Close releases Firestore and Pub/Sub resources.
func (t *buildTracker) Close() {
	if t == nil {
		return
	}
	if t.publisher != nil {
		t.publisher.Stop()
	}
	if t.fs != nil {
		_ = t.fs.Close()
	}
}

// buildInfo identifies the build being tracked. It is built once at publish
// time and passed through the state transitions.
type buildInfo struct {
	BuildID   string
	TaskID    string
	AgentID   string
	Vendor    string
	Name      string
	Version   string
	Workspace string
}

// makeBuildID returns the {task_id}-{vendor}-{name} identifier used as the
// Firestore doc ID. When task_id is missing (e.g. a human-driven publish) we
// fall back to a timestamp so each attempt still gets a unique doc.
func makeBuildID(taskID, vendor, name string) string {
	tid := taskID
	if tid == "" {
		tid = fmt.Sprintf("manual-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s-%s", tid, vendor, name)
}

// Start writes the initial `validating` document and fires the matching
// PackageEvent. Best-effort: errors are logged, never propagated.
func (t *buildTracker) Start(ctx context.Context, info buildInfo, artifactPath string) {
	if t == nil {
		return
	}
	now := time.Now().UTC()

	if t.fs != nil {
		doc := map[string]interface{}{
			"build_id":      info.BuildID,
			"task_id":       info.TaskID,
			"agent_id":      info.AgentID,
			"vendor":        info.Vendor,
			"name":          info.Name,
			"version":       info.Version,
			"status":        pubsub.PackageStatusValidating,
			"started_at":    now,
			"completed_at":  nil,
			"error":         "",
			"artifact_path": artifactPath,
			"registry_url":  "",
			"workspace":     info.Workspace,
		}
		if _, err := t.fs.Collection(packageBuildsCollection).Doc(info.BuildID).Set(ctx, doc); err != nil {
			log.Printf("build-tracker: Firestore Start(%s) failed: %v", info.BuildID, err)
		}
	}

	t.fireEvent(ctx, info, pubsub.PackageStatusValidating, "", "")
}

// Succeed records a successful publish and fires a `published` event.
func (t *buildTracker) Succeed(ctx context.Context, info buildInfo, registryURL string) {
	if t == nil {
		return
	}
	if t.fs != nil {
		updates := []firestore.Update{
			{Path: "status", Value: pubsub.PackageStatusPublished},
			{Path: "completed_at", Value: time.Now().UTC()},
			{Path: "registry_url", Value: registryURL},
		}
		if _, err := t.fs.Collection(packageBuildsCollection).Doc(info.BuildID).Update(ctx, updates); err != nil {
			log.Printf("build-tracker: Firestore Succeed(%s) failed: %v", info.BuildID, err)
		}
	}
	t.fireEvent(ctx, info, pubsub.PackageStatusPublished, registryURL, "")
}

// Fail records a failed publish and fires a `failed` event. Safe to call even
// if Start was never reached — it will still emit the event so downstream
// consumers see the failure.
func (t *buildTracker) Fail(ctx context.Context, info buildInfo, errMsg string) {
	if t == nil {
		return
	}
	trimmed := strings.TrimSpace(errMsg)
	if t.fs != nil {
		// Use Set with MergeAll so callers that failed before Start still get a
		// record (e.g. malformed manifest before the Firestore row existed).
		doc := map[string]interface{}{
			"build_id":     info.BuildID,
			"task_id":      info.TaskID,
			"agent_id":     info.AgentID,
			"vendor":       info.Vendor,
			"name":         info.Name,
			"version":      info.Version,
			"status":       pubsub.PackageStatusFailed,
			"completed_at": time.Now().UTC(),
			"error":        trimmed,
			"workspace":    info.Workspace,
		}
		if _, err := t.fs.Collection(packageBuildsCollection).Doc(info.BuildID).Set(ctx, doc, firestore.MergeAll); err != nil {
			log.Printf("build-tracker: Firestore Fail(%s) failed: %v", info.BuildID, err)
		}
	}
	t.fireEvent(ctx, info, pubsub.PackageStatusFailed, "", trimmed)
}

func (t *buildTracker) fireEvent(ctx context.Context, info buildInfo, status, registryURL, errMsg string) {
	if t == nil || t.publisher == nil {
		return
	}
	evt := pubsub.PackageEvent{
		BuildID:     info.BuildID,
		TaskID:      info.TaskID,
		AgentID:     info.AgentID,
		Vendor:      info.Vendor,
		Name:        info.Name,
		Version:     info.Version,
		Status:      status,
		RegistryURL: registryURL,
		Error:       errMsg,
	}
	if err := t.publisher.PublishPackageEvent(ctx, evt, info.Workspace); err != nil {
		log.Printf("build-tracker: publish package event (%s status=%s) failed: %v", info.BuildID, status, err)
	}
}
