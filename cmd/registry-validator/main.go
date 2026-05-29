// Registry Validator — AILANG package registry quality gate.
//
// Receives package tarballs via POST /publish, validates them
// (compile, effects, contracts), and uploads to GCS if valid.
//
// Environment:
//
//	REGISTRY_BUCKET  — GCS bucket name (required)
//	PORT             — HTTP port (default: 8080)
//	GOOGLE_CLOUD_PROJECT — GCP project (for GCS auth)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sunholo-data/ailang/internal/pkg"
)

const cacheTTL = 5 * time.Minute

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	bucket := os.Getenv("REGISTRY_BUCKET")
	if bucket == "" {
		log.Fatal("REGISTRY_BUCKET environment variable is required")
	}

	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
	}
	defer gcsClient.Close()

	bucketHandle := gcsClient.Bucket(bucket)
	v := &validator{
		bucket:     bucketHandle,
		bucketName: bucket,
		apiKey:     os.Getenv("REGISTRY_API_KEY"),
		cache:      newRegistryCache(bucketHandle, cacheTTL),
	}

	http.HandleFunc("/publish", v.handlePublish)
	http.HandleFunc("/unpublish", v.handleUnpublish)
	http.HandleFunc("/rebuild-index", v.handleRebuildIndex)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/version", handleVersion)

	// Read-only API for package explorer website
	http.HandleFunc("/api/packages", corsMiddleware(v.handleAPIPackages))
	http.HandleFunc("/api/packages/", corsMiddleware(v.handleAPIPackageDetail))
	http.HandleFunc("/api/stats", corsMiddleware(v.handleAPIStats))

	log.Printf("Registry validator listening on :%s (bucket: %s)", port, bucket)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type validator struct {
	bucket     *storage.BucketHandle
	bucketName string
	apiKey     string // if set, requires X-API-Key header on publish
	cache      *registryCache
}

// validatorBuildVersion is set at build time via -ldflags.
// Falls back to git describe at runtime if not set.
var validatorBuildVersion = "dev"

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	ailangVer := getAilangVersion()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"validator": validatorBuildVersion,
		"ailang":    ailangVer,
		"features":  "check-package,path-dep-rewrite,relative-imports,type-alias-propagation",
	})
}

func (v *validator) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Step 0: API key authentication (if configured)
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

	// Step 1: Read tarball from multipart form
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		jsonError(w, http.StatusBadRequest, "Failed to parse multipart form: %v", err)
		return
	}

	file, _, err := r.FormFile("package")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Missing 'package' field in form data")
		return
	}
	defer file.Close()

	tarballData, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Failed to read tarball: %v", err)
		return
	}

	// Step 2: Extract to temp dir
	tempDir, err := os.MkdirTemp("", "ailang-validate-*")
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to create temp dir: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)

	if err := pkg.ExtractTarball(tarballData, tempDir); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid tarball: %v", err)
		return
	}

	// Step 3: Parse ailang.toml
	manifest, err := pkg.LoadManifest(tempDir)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid manifest: %v", err)
		return
	}

	name := manifest.Package.Name
	version := manifest.Package.Version
	log.Printf("Validating %s@%s", name, version)

	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		jsonError(w, http.StatusBadRequest, "Invalid package name: %s (must be vendor/name)", name)
		return
	}

	// Step 4: Check immutability — reject if version already exists
	ctx := r.Context()
	metaPath := fmt.Sprintf("packages/%s/%s/%s/metadata.json", parts[0], parts[1], version)
	if v.bucket != nil {
		_, err = v.bucket.Object(metaPath).Attrs(ctx)
		if err == nil {
			jsonError(w, http.StatusConflict, "Version %s@%s already published (immutable)", name, version)
			return
		}
	}

	// Step 5: Namespace auth — deferred (accept all publishers for now)

	// Step 5.5: Provider-safe tool name validation (M-EXT-AUTHOR-DX M3,
	// v0.20.1). Reject publish if any advertised tool name contains
	// characters that Anthropic Bedrock + Vertex AI reject at the
	// tools[].custom.name validator (only [A-Za-z0-9_] survives). The
	// `--allow-dotted-tool-names` flag on the publisher side sends the
	// X-Allow-Dotted-Tool-Names header to downgrade to a warning during
	// the v0.20.x migration window.
	if r.Header.Get("X-Allow-Dotted-Tool-Names") != "true" {
		if names, badName, reason := validateToolNames(tempDir); badName != "" {
			jsonError(w, http.StatusBadRequest,
				"tool name %q is not provider-safe: %s\n"+
					"Anthropic Bedrock and Vertex AI reject names with characters outside [A-Za-z0-9_].\n"+
					"Suggestion: rename to %q (underscores) or %q (PascalCase).\n"+
					"Override (migration grace, v0.20.x only): publish with --allow-dotted-tool-names.\n"+
					"All advertised tool names scanned: %s",
				badName, reason, suggestSafeName(badName, "_"), suggestSafeName(badName, ""), strings.Join(names, ", "))
			return
		}
	}

	// Step 6: Compile check
	compileOk, compileErr := runAilangCheck(tempDir)

	// Step 7: Effect ceiling check (done as part of ailang check)

	// Step 8: Contract verification (best-effort)
	contractsVerified, contractsTotal, contractsSkipped := runAilangVerify(tempDir)

	if !compileOk {
		jsonError(w, http.StatusBadRequest, "Compilation failed:\n%s", compileErr)
		return
	}

	// Step 9: Compute hashes
	contentHash, _ := pkg.ContentHash(tempDir)
	interfaceHash := pkg.InterfaceHash(manifest)
	tarballHash := pkg.TarballHash(tarballData)

	// Step 10: Generate metadata.json
	ailangVersion := getAilangVersion()
	hasAgentDoc := fileExists(filepath.Join(tempDir, "AGENT.md"))

	meta := pkg.PackageMetadata{
		Schema:      "ailang.package-metadata/v1",
		Name:        name,
		Version:     version,
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
		PublishedBy: r.Header.Get("X-Publisher-Identity"),
		ContentHash: contentHash,
		InterfHash:  interfaceHash,
		TarballHash: tarballHash,
		TarballSize: int64(len(tarballData)),
		Validation: pkg.ValidationResult{
			Compiles:          compileOk,
			EffectsValid:      compileOk, // effect ceiling checked during compile
			ContractsVerified: contractsVerified,
			ContractsTotal:    contractsTotal,
			ContractsSkipped:  contractsSkipped,
			AILANGVersion:     ailangVersion,
		},
		Manifest: pkg.MetadataManifest{
			Edition:     manifest.Package.Edition,
			EffectsMax:  manifest.Effects.Max,
			Exports:     manifest.Exports.Modules,
			Stability:   manifest.Stability.Level,
			AISummary:   getMetaString(manifest.Metadata, "ai_summary"),
			HasAgentDoc: hasAgentDoc,
			Repository:  getMetaString(manifest.Metadata, "repository"),
			Homepage:    getMetaString(manifest.Metadata, "homepage"),
			LicenseURL:  getMetaString(manifest.Metadata, "license_url"),
		},
	}

	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Failed to marshal metadata: %v", err)
		return
	}

	// Step 11: Upload to GCS (skip if no bucket configured — dry-run/test mode)
	if v.bucket != nil {
		tarballGCSPath := fmt.Sprintf("packages/%s/%s/%s/package.tar.gz", parts[0], parts[1], version)

		if err := v.uploadToGCS(ctx, tarballGCSPath, tarballData, "application/gzip"); err != nil {
			jsonError(w, http.StatusInternalServerError, "Failed to upload tarball: %v", err)
			return
		}

		if err := v.uploadToGCS(ctx, metaPath, metaJSON, "application/json"); err != nil {
			jsonError(w, http.StatusInternalServerError, "Failed to upload metadata: %v", err)
			return
		}

		// M-PKG-AUTONOMOUS-UPDATES: Upload AGENT.md as a separate first-class artifact.
		// Makes it directly queryable without downloading the tarball.
		agentMDPath := filepath.Join(tempDir, "AGENT.md")
		if hasAgentDoc {
			agentMDData, err := os.ReadFile(agentMDPath)
			if err == nil {
				gcsAgentPath := fmt.Sprintf("packages/%s/%s/%s/AGENT.md", parts[0], parts[1], version)
				if err := v.uploadToGCS(ctx, gcsAgentPath, agentMDData, "text/markdown"); err != nil {
					log.Printf("Warning: failed to upload AGENT.md: %v", err)
				} else {
					log.Printf("Uploaded AGENT.md as first-class artifact")
				}
			}
		}

		// Update index.json
		if err := v.updateIndex(ctx, manifest, &meta); err != nil {
			log.Printf("WARNING: Failed to update index.json: %v (package uploaded but index stale)", err)
		}
	} else {
		log.Printf("No GCS bucket configured — validation-only mode (no upload)")
	}

	log.Printf("Published %s@%s (%d bytes, %d/%d contracts verified)", name, version, len(tarballData), contractsVerified, contractsTotal)

	// Invalidate API cache so fresh data is served immediately
	if v.cache != nil {
		v.cache.Invalidate()
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metaJSON)
}

// uploadToGCS writes data to a GCS object.
func (v *validator) uploadToGCS(ctx context.Context, path string, data []byte, contentType string) error {
	obj := v.bucket.Object(path)
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType

	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		writer.Close()
		return fmt.Errorf("write to %s: %w", path, err)
	}
	return writer.Close()
}

// updateIndex reads index.json, merges the new package entry, and writes back.
// Uses GCS generation preconditions for optimistic locking — retries on conflict.
func (v *validator) updateIndex(ctx context.Context, manifest *pkg.PackageManifest, meta *pkg.PackageMetadata) error {
	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Back off before retry
			time.Sleep(time.Duration(attempt*200) * time.Millisecond)
			log.Printf("Retrying index.json update (attempt %d/%d)", attempt+1, maxRetries)
		}

		err := v.tryUpdateIndex(ctx, manifest, meta)
		if err == nil {
			return nil
		}

		// If the error is a precondition failure (generation mismatch), retry
		if strings.Contains(err.Error(), "conditionNotMet") || strings.Contains(err.Error(), "generation") {
			log.Printf("index.json generation conflict, retrying: %v", err)
			continue
		}

		// Non-retryable error
		return err
	}
	return fmt.Errorf("failed to update index.json after %d retries (generation conflicts)", maxRetries)
}

func (v *validator) tryUpdateIndex(ctx context.Context, manifest *pkg.PackageManifest, meta *pkg.PackageMetadata) error {
	obj := v.bucket.Object("index.json")

	// Read current index with its generation number
	var index pkg.RegistryIndex
	var generation int64

	attrs, err := obj.Attrs(ctx)
	if err == nil {
		generation = attrs.Generation

		reader, err := obj.NewReader(ctx)
		if err == nil {
			data, _ := io.ReadAll(reader)
			reader.Close()
			json.Unmarshal(data, &index)
		}
	} else {
		// First publish — create new index
		index = pkg.RegistryIndex{Schema: "ailang.registry/v1"}
		generation = 0
	}

	// Extract dependency names from manifest (M-PKG-AUTONOMOUS-UPDATES)
	var depNames []string
	for depName := range manifest.Dependencies {
		depNames = append(depNames, depName)
	}

	// Find or create entry for this package
	found := false
	for i := range index.Packages {
		if index.Packages[i].Name == manifest.Package.Name {
			index.Packages[i].Latest = manifest.Package.Version
			if !containsString(index.Packages[i].Versions, manifest.Package.Version) {
				index.Packages[i].Versions = append(index.Packages[i].Versions, manifest.Package.Version)
			}
			index.Packages[i].ContractsVerified = meta.Validation.ContractsVerified
			index.Packages[i].Dependencies = depNames
			index.Packages[i].LastUpdated = meta.PublishedAt
			index.Packages[i].UpdatedBy = meta.PublishedBy
			index.Packages[i].Repository = meta.Manifest.Repository
			index.Packages[i].Homepage = meta.Manifest.Homepage
			index.Packages[i].LicenseURL = meta.Manifest.LicenseURL
			// Update metadata fields that may change between versions
			index.Packages[i].AISummary = getMetaString(manifest.Metadata, "ai_summary")
			index.Packages[i].Tags = getMetaStringSlice(manifest.Metadata, "tags")
			index.Packages[i].Effects = manifest.Effects.Max
			index.Packages[i].Stability = manifest.Stability.Level
			index.Packages[i].Exports = manifest.Exports.Modules
			index.Packages[i].HasAgentDoc = meta.Manifest.HasAgentDoc
			found = true
			break
		}
	}

	if !found {
		tags := getMetaStringSlice(manifest.Metadata, "tags")
		index.Packages = append(index.Packages, pkg.IndexEntry{
			Name:              manifest.Package.Name,
			Latest:            manifest.Package.Version,
			Versions:          []string{manifest.Package.Version},
			AISummary:         getMetaString(manifest.Metadata, "ai_summary"),
			Tags:              tags,
			Effects:           manifest.Effects.Max,
			Stability:         manifest.Stability.Level,
			Exports:           manifest.Exports.Modules,
			ContractsVerified: meta.Validation.ContractsVerified,
			HasAgentDoc:       meta.Manifest.HasAgentDoc,
			Dependencies:      depNames,
			LastUpdated:       meta.PublishedAt,
			UpdatedBy:         meta.PublishedBy,
			Repository:        meta.Manifest.Repository,
			Homepage:          meta.Manifest.Homepage,
			LicenseURL:        meta.Manifest.LicenseURL,
		})
	}

	index.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	indexJSON, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	// Write with generation precondition — fails if someone else updated since we read
	var writer *storage.Writer
	if generation > 0 {
		writer = obj.If(storage.Conditions{GenerationMatch: generation}).NewWriter(ctx)
	} else {
		writer = obj.If(storage.Conditions{DoesNotExist: true}).NewWriter(ctx)
	}
	writer.ContentType = "application/json"
	writer.CacheControl = "no-cache, no-store, must-revalidate" // prevent stale reads

	if _, err := io.Copy(writer, bytes.NewReader(indexJSON)); err != nil {
		writer.Close()
		return fmt.Errorf("write index.json: %w", err)
	}
	return writer.Close()
}
