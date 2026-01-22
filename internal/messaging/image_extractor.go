// Package messaging provides image extraction from GitHub issue content.
package messaging

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// currentTime returns the current time. Can be overridden for testing.
var currentTime = time.Now

// imageMarkdownRegex matches markdown image syntax: ![alt](url)
var imageMarkdownRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// imageHTMLRegex matches HTML image syntax: <img ... src="url" ... />
// Captures: [0]=full match, [1]=pre-src attributes, [2]=url, [3]=post-src attributes
var imageHTMLRegex = regexp.MustCompile(`<img\s+([^>]*?)src="([^"]+)"([^>]*)/?>`)

// githubImageDomains are domains/paths that host GitHub images
var githubImageDomains = []string{
	"private-user-images.githubusercontent.com",
	"user-images.githubusercontent.com",
	"github.com/user-attachments/assets",
	"github.com",
	"raw.githubusercontent.com",
}

// maxImageSize is the maximum size of an image to download (10MB)
const maxImageSize = 10 * 1024 * 1024

// ExtractAndCacheImages finds markdown images in content, downloads them to local cache,
// and returns modified content with local file paths replacing URLs.
//
// Parameters:
//   - ctx: context for cancellation
//   - content: markdown content potentially containing images
//   - issueID: GitHub issue number (for organizing cache)
//   - repo: repository name in "owner/repo" or "owner-repo" format
//
// Returns:
//   - modified content with local paths
//   - list of cached image paths
//   - error if critical failure (individual image failures are logged but don't fail)
func ExtractAndCacheImages(ctx context.Context, content string, issueID int, repo string) (string, []string, error) {
	if content == "" {
		return content, nil, nil
	}

	// Find all markdown images: ![alt](url)
	mdMatches := imageMarkdownRegex.FindAllStringSubmatchIndex(content, -1)
	// Find all HTML images: <img src="url" />
	htmlMatches := imageHTMLRegex.FindAllStringSubmatchIndex(content, -1)

	if len(mdMatches) == 0 && len(htmlMatches) == 0 {
		return content, nil, nil
	}

	// Prepare cache directory
	cacheDir, err := getImageCacheDir(repo, issueID)
	if err != nil {
		return content, nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	var cachedPaths []string
	var replacements []struct {
		start, end int
		newText    string
	}

	// Process markdown images: ![alt](url)
	for _, match := range mdMatches {
		// match[0:1] = full match start/end
		// match[2:3] = alt text start/end
		// match[4:5] = URL start/end
		fullStart, fullEnd := match[0], match[1]
		altStart, altEnd := match[2], match[3]
		urlStart, urlEnd := match[4], match[5]

		altText := content[altStart:altEnd]
		imageURL := content[urlStart:urlEnd]

		// Check if it's a GitHub image URL
		if !isGitHubImageURL(imageURL) {
			continue
		}

		// Download and cache the image
		localPath, err := downloadAndCacheImage(ctx, imageURL, cacheDir)
		if err != nil {
			// Log warning but continue - don't fail for individual image issues
			fmt.Fprintf(os.Stderr, "Warning: failed to download image %s: %v\n", truncateURL(imageURL), err)
			continue
		}

		cachedPaths = append(cachedPaths, localPath)

		// Create replacement text with local path (keep markdown format)
		newMarkdown := fmt.Sprintf("![%s](%s)", altText, localPath)
		replacements = append(replacements, struct {
			start, end int
			newText    string
		}{fullStart, fullEnd, newMarkdown})
	}

	// Process HTML images: <img ... src="url" ... />
	for _, match := range htmlMatches {
		// match[0:1] = full match start/end
		// match[4:5] = URL start/end (group 2)
		fullStart, fullEnd := match[0], match[1]
		urlStart, urlEnd := match[4], match[5]

		imageURL := content[urlStart:urlEnd]

		// Check if it's a GitHub image URL
		if !isGitHubImageURL(imageURL) {
			continue
		}

		// Download and cache the image
		localPath, err := downloadAndCacheImage(ctx, imageURL, cacheDir)
		if err != nil {
			// Log warning but continue - don't fail for individual image issues
			fmt.Fprintf(os.Stderr, "Warning: failed to download image %s: %v\n", truncateURL(imageURL), err)
			continue
		}

		cachedPaths = append(cachedPaths, localPath)

		// Create replacement text with local path (convert to markdown for consistency)
		newMarkdown := fmt.Sprintf("![image](%s)", localPath)
		replacements = append(replacements, struct {
			start, end int
			newText    string
		}{fullStart, fullEnd, newMarkdown})
	}

	// Apply replacements in reverse order to preserve indices
	result := content
	for i := len(replacements) - 1; i >= 0; i-- {
		r := replacements[i]
		result = result[:r.start] + r.newText + result[r.end:]
	}

	return result, cachedPaths, nil
}

// getImageCacheDir returns the cache directory for a repo/issue, creating it if needed.
func getImageCacheDir(repo string, issueID int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Sanitize repo name for filesystem
	safeRepo := strings.ReplaceAll(repo, "/", "-")
	safeRepo = strings.ReplaceAll(safeRepo, "\\", "-")

	cacheDir := filepath.Join(homeDir, ".ailang", "cache", "images", safeRepo, fmt.Sprintf("issue-%d", issueID))

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}

// isGitHubImageURL checks if the URL is from a GitHub image domain.
func isGitHubImageURL(url string) bool {
	for _, domain := range githubImageDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

// downloadAndCacheImage downloads an image using gh api and saves it to the cache directory.
func downloadAndCacheImage(ctx context.Context, imageURL, cacheDir string) (string, error) {
	// Generate filename from URL hash
	hash := sha256.Sum256([]byte(imageURL))
	hashStr := fmt.Sprintf("%x", hash[:8]) // First 8 bytes = 16 hex chars

	// Extract extension from URL (before query params)
	ext := getImageExtension(imageURL)
	filename := hashStr + ext

	localPath := filepath.Join(cacheDir, filename)

	// Check if already cached
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Download using gh api (handles auth automatically)
	// For GitHub URLs, we use a direct download approach
	cmd := exec.CommandContext(ctx, "curl", "-sL", "-o", localPath, imageURL)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.Remove(localPath) // Clean up partial file
		return "", fmt.Errorf("download failed: %w (output: %s)", err, string(output))
	}

	// Verify file was created and check size
	info, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("file not created: %w", err)
	}
	if info.Size() == 0 {
		os.Remove(localPath)
		return "", fmt.Errorf("downloaded file is empty")
	}
	if info.Size() > maxImageSize {
		os.Remove(localPath)
		return "", fmt.Errorf("image too large (%d bytes, max %d)", info.Size(), maxImageSize)
	}

	return localPath, nil
}

// getImageExtension extracts the file extension from a URL.
func getImageExtension(url string) string {
	// Remove query params
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	// Get extension
	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".png" // Default to PNG for GitHub images
	}
	return ext
}

// truncateURL shortens a URL for logging.
func truncateURL(url string) string {
	if len(url) <= 80 {
		return url
	}
	return url[:40] + "..." + url[len(url)-37:]
}

// CleanupImageCache removes cached images for a list of paths.
func CleanupImageCache(imagePaths []string) error {
	for _, path := range imagePaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}

// CleanupOldImages removes cached images older than the specified duration.
// Returns the number of files removed.
func CleanupOldImages(olderThanDays int) (int, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}

	cacheDir := filepath.Join(homeDir, ".ailang", "cache", "images")

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return 0, nil // No cache directory
	}

	removed := 0
	cutoff := int64(olderThanDays * 24 * 60 * 60) // Convert days to seconds
	now := currentTime().Unix()

	err = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		age := now - info.ModTime().Unix()
		if age > cutoff {
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
		return nil
	})

	return removed, err
}
