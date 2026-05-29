package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// handleRebuildIndex scans all metadata.json files in the bucket and
// rebuilds index.json from scratch. Requires API key auth.
func (v *validator) handleRebuildIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Same API key auth as publish
	if v.apiKey != "" {
		provided := r.Header.Get("X-API-Key")
		if provided == "" {
			provided = r.URL.Query().Get("api_key")
		}
		if provided != v.apiKey {
			jsonError(w, http.StatusForbidden, "Invalid or missing API key")
			return
		}
	}

	if v.bucket == nil {
		jsonError(w, http.StatusInternalServerError, "No GCS bucket configured")
		return
	}

	ctx := r.Context()
	log.Printf("Rebuilding index.json from metadata files...")

	// Scan all metadata.json files in packages/
	var index pkg.RegistryIndex
	index.Schema = "ailang.registry/v1"

	query := &storage.Query{Prefix: "packages/"}
	query.SetAttrSelection([]string{"Name"})

	it := v.bucket.Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err != nil {
			break // end of iteration
		}

		// Only process metadata.json files
		if !strings.HasSuffix(attrs.Name, "/metadata.json") {
			continue
		}

		// Read metadata
		reader, err := v.bucket.Object(attrs.Name).NewReader(ctx)
		if err != nil {
			log.Printf("  Skip %s: %v", attrs.Name, err)
			continue
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}

		var meta pkg.PackageMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			log.Printf("  Skip %s: bad JSON: %v", attrs.Name, err)
			continue
		}

		log.Printf("  Found %s@%s", meta.Name, meta.Version)

		// Find or create entry
		found := false
		for i := range index.Packages {
			if index.Packages[i].Name == meta.Name {
				// Add version if new
				if !containsString(index.Packages[i].Versions, meta.Version) {
					index.Packages[i].Versions = append(index.Packages[i].Versions, meta.Version)
				}
				// Update latest to highest version
				index.Packages[i].Latest = meta.Version
				index.Packages[i].ContractsVerified = meta.Validation.ContractsVerified
				found = true
				break
			}
		}

		if !found {
			index.Packages = append(index.Packages, pkg.IndexEntry{
				Name:              meta.Name,
				Latest:            meta.Version,
				Versions:          []string{meta.Version},
				AISummary:         meta.Manifest.AISummary,
				Effects:           meta.Manifest.EffectsMax,
				Stability:         meta.Manifest.Stability,
				Exports:           meta.Manifest.Exports,
				ContractsVerified: meta.Validation.ContractsVerified,
				HasAgentDoc:       meta.Manifest.HasAgentDoc,
				Repository:        meta.Manifest.Repository,
				Homepage:          meta.Manifest.Homepage,
				LicenseURL:        meta.Manifest.LicenseURL,
			})
		}
	}

	index.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	indexJSON, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to marshal index: %v", err)
		return
	}

	// Write index (unconditional — this is a full rebuild)
	writer := v.bucket.Object("index.json").NewWriter(ctx)
	writer.ContentType = "application/json"
	writer.CacheControl = "no-cache, no-store, must-revalidate"
	if _, err := io.Copy(writer, bytes.NewReader(indexJSON)); err != nil {
		writer.Close()
		jsonError(w, http.StatusInternalServerError, "Failed to write index: %v", err)
		return
	}
	if err := writer.Close(); err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to finalize index: %v", err)
		return
	}

	log.Printf("Rebuilt index.json with %d packages", len(index.Packages))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"packages": len(index.Packages),
	})
}

// handleUnpublish removes a specific package version from the registry.
// Deletes the GCS objects and rebuilds the index. Requires API key auth.
func (v *validator) handleUnpublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// API key authentication (always required for unpublish)
	if v.apiKey == "" {
		jsonError(w, http.StatusForbidden, "Unpublish requires API key configuration on the server")
		return
	}
	provided := r.Header.Get("X-API-Key")
	if provided == "" {
		provided = r.URL.Query().Get("api_key")
	}
	if provided != v.apiKey {
		jsonError(w, http.StatusForbidden, "Invalid or missing API key")
		return
	}

	name := r.URL.Query().Get("name")
	version := r.URL.Query().Get("version")
	if name == "" || version == "" {
		jsonError(w, http.StatusBadRequest, "Both 'name' and 'version' query parameters are required")
		return
	}

	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		jsonError(w, http.StatusBadRequest, "Invalid package name: %s (must be vendor/name)", name)
		return
	}

	ctx := r.Context()
	vendor, pkgName := parts[0], parts[1]
	prefix := fmt.Sprintf("packages/%s/%s/%s/", vendor, pkgName, version)

	// Verify the version exists by checking for metadata.json
	metaObj := v.bucket.Object(prefix + "metadata.json")
	if _, err := metaObj.Attrs(ctx); err != nil {
		jsonError(w, http.StatusNotFound, "Package %s@%s not found in registry", name, version)
		return
	}

	log.Printf("Unpublishing %s@%s — deleting objects with prefix %s", name, version, prefix)

	// Delete all objects under this version prefix
	query := &storage.Query{Prefix: prefix}
	query.SetAttrSelection([]string{"Name"})
	it := v.bucket.Objects(ctx, query)
	var deleted int
	for {
		attrs, err := it.Next()
		if err != nil {
			break
		}
		if err := v.bucket.Object(attrs.Name).Delete(ctx); err != nil {
			log.Printf("  Failed to delete %s: %v", attrs.Name, err)
		} else {
			log.Printf("  Deleted %s", attrs.Name)
			deleted++
		}
	}

	log.Printf("Deleted %d objects for %s@%s", deleted, name, version)

	// Rebuild the index by removing this version
	remainingVersions, err := v.removeVersionFromIndex(ctx, name, version)
	if err != nil {
		log.Printf("Warning: failed to update index after unpublish: %v", err)
		// Still return success — the objects are deleted, index can be rebuilt manually
	}

	// Invalidate cache
	v.cache.Invalidate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":            fmt.Sprintf("Removed %s@%s (%d objects deleted)", name, version, deleted),
		"remaining_versions": remainingVersions,
	})
}

// removeVersionFromIndex updates index.json to remove a specific version.
// If no versions remain, removes the package entry entirely.
// Returns the remaining versions list.
func (v *validator) removeVersionFromIndex(ctx context.Context, name, version string) ([]string, error) {
	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*200) * time.Millisecond)
			log.Printf("Retrying index.json update for unpublish (attempt %d/%d)", attempt+1, maxRetries)
		}

		remaining, err := v.tryRemoveVersionFromIndex(ctx, name, version)
		if err == nil {
			return remaining, nil
		}

		if strings.Contains(err.Error(), "conditionNotMet") || strings.Contains(err.Error(), "generation") {
			log.Printf("index.json generation conflict, retrying: %v", err)
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("failed to update index.json after %d retries", maxRetries)
}

func (v *validator) tryRemoveVersionFromIndex(ctx context.Context, name, version string) ([]string, error) {
	obj := v.bucket.Object("index.json")

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json attrs: %w", err)
	}
	generation := attrs.Generation

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json: %w", err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json body: %w", err)
	}

	var index pkg.RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse index.json: %w", err)
	}

	var remaining []string
	found := false
	for i := range index.Packages {
		if index.Packages[i].Name != name {
			continue
		}
		found = true

		// Remove the version from the versions list
		var newVersions []string
		for _, v := range index.Packages[i].Versions {
			if v != version {
				newVersions = append(newVersions, v)
			}
		}

		if len(newVersions) == 0 {
			// No versions left — remove the package entry entirely
			index.Packages = append(index.Packages[:i], index.Packages[i+1:]...)
			log.Printf("Removed package %s entirely from index (no versions remain)", name)
		} else {
			index.Packages[i].Versions = newVersions
			// Update Latest to the last remaining version
			index.Packages[i].Latest = newVersions[len(newVersions)-1]
			index.Packages[i].LastUpdated = time.Now().UTC().Format(time.RFC3339)
			remaining = newVersions
			log.Printf("Updated index for %s: removed %s, latest now %s", name, version, index.Packages[i].Latest)
		}
		break
	}

	if !found {
		return nil, nil // Package not in index — nothing to do
	}

	index.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	indexJSON, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}

	writer := obj.If(storage.Conditions{GenerationMatch: generation}).NewWriter(ctx)
	writer.ContentType = "application/json"
	writer.CacheControl = "no-cache, no-store, must-revalidate"
	if _, err := io.Copy(writer, bytes.NewReader(indexJSON)); err != nil {
		writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return remaining, nil
}
