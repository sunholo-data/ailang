package apiserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// --static-cache (M-SERVEAPI-OPERATOR-SURFACE D6): an opt-in Cache-Control
// for the --static file server. Without the flag no header is set.

// staticCacheMaxAge is one year, the conventional ceiling for immutable assets.
const staticCacheMaxAge = 31536000

// ParseStaticCache turns a --static-cache value into its Cache-Control header:
//
//	immutable -> "public, max-age=31536000, immutable"
//	N         -> "public, max-age=N"   (1 <= N <= 31536000 seconds)
//
// "" returns "" (no header). Anything else is an error naming both forms.
func ParseStaticCache(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	if strings.EqualFold(v, "immutable") {
		return fmt.Sprintf("public, max-age=%d, immutable", staticCacheMaxAge), nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > staticCacheMaxAge {
		return "", fmt.Errorf("--static-cache %q: want 'immutable' or a max-age in seconds (1-%d)", v, staticCacheMaxAge)
	}
	return fmt.Sprintf("public, max-age=%d", n), nil
}

// staticCacheHandler sets Cache-Control on responses that are safe to cache:
// 2xx and 304. A 404 or 416 must never be cached, least of all as immutable.
func staticCacheHandler(cacheControl string, next http.Handler) http.Handler {
	if cacheControl == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&cacheWriter{ResponseWriter: w, value: cacheControl}, r)
	})
}

type cacheWriter struct {
	http.ResponseWriter
	value       string
	wroteHeader bool
}

func (c *cacheWriter) WriteHeader(status int) {
	if !c.wroteHeader {
		c.wroteHeader = true
		if (status >= 200 && status < 300) || status == http.StatusNotModified {
			c.Header().Set("Cache-Control", c.value)
		}
	}
	c.ResponseWriter.WriteHeader(status)
}

// No Write override: http.FileServer calls WriteHeader explicitly for file
// content (200/206) and 304, which is where the header belongs. A body
// written without WriteHeader (a directory listing) stays uncached, which is
// right for a listing that changes when files are added.

// Unwrap lets http.ResponseController reach the underlying writer.
func (c *cacheWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }
