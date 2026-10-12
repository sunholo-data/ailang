package apiserver

import (
	"fmt"
	"net/http"
	"strings"
)

// StaticHeaderOptions controls headers on static files only.
type StaticHeaderOptions struct {
	Disabled bool
	Headers  []string
}

// ParseStaticHeaders validates operator overrides before the server starts.
func ParseStaticHeaders(path string, disabled bool, flags []string) (http.Header, error) {
	if path == "" && (disabled || len(flags) > 0) {
		return nil, fmt.Errorf("--static-header and --no-static-security-headers need --static")
	}
	h := make(http.Header)
	if !disabled {
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "frame-ancestors 'none'")
		h.Set("Referrer-Policy", "no-referrer")
	}
	for _, raw := range flags {
		name, value, ok := strings.Cut(raw, ":")
		name = strings.Trim(name, " ")
		// Trim spaces only: trimming controls would silently accept an unsafe value.
		value = strings.Trim(value, " ")
		if !ok || !isHTTPToken(name) || value == "" || !validResponseHeaderValue(value) {
			return nil, fmt.Errorf("--static-header %q: want 'Name: value' with a valid HTTP token name and nonempty value without control characters", raw)
		}
		h.Set(name, value)
	}
	return h, nil
}

func validResponseHeaderValue(value string) bool {
	for _, r := range value {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func staticHeadersHandler(headers http.Header, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for name, values := range headers {
			w.Header()[name] = append([]string(nil), values...)
		}
		next.ServeHTTP(w, r)
	})
}

// Config was checked by ValidateStaticHeaders before serving. Keeping the
// wrapper here leaves the large server routing module with a single hook.
func (s *Server) staticHandler() http.Handler {
	return staticCacheHandler(s.staticCache, http.FileServer(http.Dir(s.staticPath)))
}

// ServeMux can commit a clean-path redirect before invoking FileServer. Apply
// static headers around routing, but only for the selected static fallback.
func (s *Server) staticRoutingHandler(mux *http.ServeMux) http.Handler {
	if s.staticPath == "" {
		return mux
	}
	headers, err := ParseStaticHeaders(s.staticPath, s.staticHeaders.Disabled, s.staticHeaders.Headers)
	if err != nil {
		panic(err)
	} // public startup validates before building routes.
	withHeaders := staticHeadersHandler(headers, mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if pattern == "/" {
			withHeaders.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) ValidateStaticHeaders() error {
	_, err := ParseStaticHeaders(s.staticPath, s.staticHeaders.Disabled, s.staticHeaders.Headers)
	return err
}
