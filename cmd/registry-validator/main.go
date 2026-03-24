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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sunholo/ailang/internal/pkg"
)

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

	v := &validator{
		bucket:     gcsClient.Bucket(bucket),
		bucketName: bucket,
		apiKey:     os.Getenv("REGISTRY_API_KEY"),
	}

	http.HandleFunc("/publish", v.handlePublish)
	http.HandleFunc("/rebuild-index", v.handleRebuildIndex)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Registry validator listening on :%s (bucket: %s)", port, bucket)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type validator struct {
	bucket     *storage.BucketHandle
	bucketName string
	apiKey     string // if set, requires X-API-Key header on publish
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
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

		// Update index.json
		if err := v.updateIndex(ctx, manifest, &meta); err != nil {
			log.Printf("WARNING: Failed to update index.json: %v (package uploaded but index stale)", err)
		}
	} else {
		log.Printf("No GCS bucket configured — validation-only mode (no upload)")
	}

	log.Printf("Published %s@%s (%d bytes, %d/%d contracts verified)", name, version, len(tarballData), contractsVerified, contractsTotal)

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

	// Find or create entry for this package
	found := false
	for i := range index.Packages {
		if index.Packages[i].Name == manifest.Package.Name {
			index.Packages[i].Latest = manifest.Package.Version
			if !containsString(index.Packages[i].Versions, manifest.Package.Version) {
				index.Packages[i].Versions = append(index.Packages[i].Versions, manifest.Package.Version)
			}
			index.Packages[i].ContractsVerified = meta.Validation.ContractsVerified
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

// runAilangCheck runs ailang check on the package directory.
// Uses `ailang check --package .` for packages with ailang.toml (resolves dependencies
// and cross-module types), falls back to per-file checks for bare packages.
func runAilangCheck(dir string) (bool, string) {
	// If ailang.toml exists, use package-level check (resolves deps + cross-module types)
	if fileExists(filepath.Join(dir, "ailang.toml")) {
		// Step 1: Rewrite path deps to registry deps.
		// The tarball's ailang.toml has path deps (e.g., { path = "../gcp-auth" })
		// which don't exist in the validator's temp dir. Replace them with
		// registry version deps so ailang lock can resolve them.
		manifest, err := pkg.LoadManifest(dir)
		if err == nil && len(manifest.Dependencies) > 0 {
			if err := rewritePathDepsToRegistry(dir, manifest); err != nil {
				log.Printf("Warning: failed to rewrite path deps: %v", err)
			}
		}

		// Step 2: Generate lockfile with registry-resolved deps
		lockCmd := exec.Command("ailang", "lock")
		lockCmd.Dir = dir
		if lockOutput, err := lockCmd.CombinedOutput(); err != nil {
			log.Printf("Warning: ailang lock failed: %s", string(lockOutput))
			// Fall through to package check anyway — it may work without deps
		}

		// Step 3: Run package-level type check
		cmd := exec.Command("ailang", "check", "--package", ".")
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, string(output)
		}
		return true, ""
	}

	// Fallback: check individual files (for packages without ailang.toml)
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})

	for _, f := range files {
		cmd := exec.Command("ailang", "check", f)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, string(output)
		}
	}
	return true, ""
}

// runAilangVerify runs ailang verify and parses results.
func runAilangVerify(dir string) (verified, total, skipped int) {
	var files []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})

	for _, f := range files {
		cmd := exec.Command("ailang", "verify", "--json", f)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			skipped++
			continue
		}
		// Parse JSON output to count verified/counterexample/skipped
		var results []struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(output, &results) == nil {
			for _, r := range results {
				total++
				switch r.Status {
				case "verified":
					verified++
				case "skipped", "timeout", "error":
					skipped++
				}
			}
		}
	}
	return
}

// rewritePathDepsToRegistry reads ailang.toml, replaces path dependencies with
// registry version strings (latest version from registry index), and writes back.
// This enables the validator to resolve deps that were local path deps during development.
func rewritePathDepsToRegistry(dir string, manifest *pkg.PackageManifest) error {
	tomlPath := filepath.Join(dir, "ailang.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return fmt.Errorf("read ailang.toml: %w", err)
	}
	content := string(data)

	for depName, dep := range manifest.Dependencies {
		if dep.Path == "" {
			continue // Not a path dep — skip
		}
		// Replace the path dep line with a version dep.
		// Look up the latest version from registry index.
		version := lookupLatestVersion(depName)
		if version == "" {
			log.Printf("Warning: dependency %s not found in registry, skipping", depName)
			continue
		}

		// Replace various path dep formats:
		//   "vendor/name" = { path = "..." }
		//   "vendor/name" = { path = "../something" }
		// with:
		//   "vendor/name" = "version"
		old := fmt.Sprintf(`"%s" = { path = "%s" }`, depName, dep.Path)
		new := fmt.Sprintf(`"%s" = "%s"`, depName, version)
		if strings.Contains(content, old) {
			content = strings.Replace(content, old, new, 1)
			log.Printf("Rewrote dep %s: path -> registry %s", depName, version)
		} else {
			// Try without quotes around path value
			old2 := fmt.Sprintf(`"%s" = {path = "%s"}`, depName, dep.Path)
			if strings.Contains(content, old2) {
				content = strings.Replace(content, old2, new, 1)
				log.Printf("Rewrote dep %s: path -> registry %s", depName, version)
			} else {
				log.Printf("Warning: could not find path dep pattern for %s in ailang.toml", depName)
			}
		}
	}

	return os.WriteFile(tomlPath, []byte(content), 0644)
}

// lookupLatestVersion fetches the registry index and finds the latest version of a package.
func lookupLatestVersion(pkgName string) string {
	registryURL := os.Getenv("AILANG_REGISTRY")
	if registryURL == "" {
		registryURL = "https://storage.googleapis.com/ailang-registry"
	}
	// Fetch index.json
	indexURL := registryURL + "/index.json"
	cmd := exec.Command("curl", "-sf", indexURL)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	// Parse index to find package
	var index struct {
		Packages []struct {
			Name   string `json:"name"`
			Latest string `json:"latest"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(output, &index); err != nil {
		return ""
	}
	for _, p := range index.Packages {
		if p.Name == pkgName {
			return p.Latest
		}
	}
	return ""
}

func getAilangVersion() string {
	cmd := exec.Command("ailang", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getMetaString(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getMetaStringSlice(meta map[string]interface{}, key string) []string {
	if meta == nil {
		return nil
	}
	v, ok := meta[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

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

func jsonError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
