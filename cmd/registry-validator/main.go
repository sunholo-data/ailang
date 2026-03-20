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
	}

	http.HandleFunc("/publish", v.handlePublish)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Registry validator listening on :%s (bucket: %s)", port, bucket)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type validator struct {
	bucket     *storage.BucketHandle
	bucketName string
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
func (v *validator) updateIndex(ctx context.Context, manifest *pkg.PackageManifest, meta *pkg.PackageMetadata) error {
	// Read current index
	var index pkg.RegistryIndex
	reader, err := v.bucket.Object("index.json").NewReader(ctx)
	if err == nil {
		defer reader.Close()
		data, _ := io.ReadAll(reader)
		json.Unmarshal(data, &index)
	} else {
		// First publish — create new index
		index = pkg.RegistryIndex{Schema: "ailang.registry/v1"}
	}

	// Find or create entry for this package
	found := false
	for i := range index.Packages {
		if index.Packages[i].Name == manifest.Package.Name {
			// Update existing entry
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

	return v.uploadToGCS(ctx, "index.json", indexJSON, "application/json")
}

// runAilangCheck runs ailang check on the package directory.
func runAilangCheck(dir string) (bool, string) {
	// Find .ail files to check
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

func jsonError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
