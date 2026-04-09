package apiserver

import (
	"fmt"
	"sort"
	"strings"
)

// maxAvailableRoutes caps the number of routes reported in an error's
// available_routes list. Prevents the response from ballooning when a server
// has hundreds of registered routes.
const maxAvailableRoutes = 10

// maxSuggestedMatches caps the number of "did you mean" suggestions. Keeps
// the suggested_fix string readable.
const maxSuggestedMatches = 3

// scoredRoute is a route paired with its common-prefix overlap score against
// the failed request path. Used internally by suggestRoutes for ranking.
type scoredRoute struct {
	entry   RouteEntry
	overlap int
}

// suggestRoutes analyzes a failed (method, path) request against the set of
// registered routes and returns:
//
//   - suggestedFix: a "Did you mean ...?" hint, or empty string if no route
//     is close enough to suggest.
//   - available:    a bounded list of routes that share a path prefix with
//     the request (for the error's available_routes field).
//
// The algorithm works on /-separated path segments:
//
//  1. A route is a "close match" if it shares at least half the request
//     path's segments as a leading prefix AND its method matches the request.
//     If no same-method matches exist, the method filter is relaxed.
//
//  2. The available list contains all routes sharing the first 2 segments
//     of the request path (e.g. a request to /api/v1/auth/foo pulls in all
//     routes under /api/v1/...). If fewer than 2 segments match, all routes
//     are returned. The list is capped at maxAvailableRoutes.
//
// suggestedFix is only populated when at least one close match exists.
// Results are sorted deterministically so responses are stable across calls.
func suggestRoutes(method, path string, routes []RouteEntry) (suggestedFix string, available []string) {
	if len(routes) == 0 {
		return "", nil
	}

	reqSegs := pathSegments(path)
	reqSegCount := len(reqSegs)

	// Minimum number of leading segments a route must share with the
	// request path to qualify as "close". At least 1, or half the request
	// path length, whichever is larger.
	minClosePrefix := reqSegCount / 2
	if minClosePrefix < 1 {
		minClosePrefix = 1
	}

	var sameMethodClose []scoredRoute
	var anyMethodClose []scoredRoute
	var prefix2 []RouteEntry

	for _, r := range routes {
		if r.Path == "" {
			continue
		}
		routeSegs := pathSegments(r.Path)
		overlap := commonPrefixLen(reqSegs, routeSegs)

		// Group 1: "close matches" for suggested_fix.
		if overlap >= minClosePrefix {
			if r.Method == method {
				sameMethodClose = append(sameMethodClose, scoredRoute{r, overlap})
			} else {
				anyMethodClose = append(anyMethodClose, scoredRoute{r, overlap})
			}
		}

		// Group 2: "available routes" — share first 2 segments with the
		// request path (or all routes if request is shorter than 2 segs).
		sharesTwo := overlap >= 2 || reqSegCount < 2
		if sharesTwo {
			prefix2 = append(prefix2, r)
		}
	}

	// Pick the close-match set: prefer same-method, fall back to any-method.
	closeMatches := sameMethodClose
	if len(closeMatches) == 0 {
		closeMatches = anyMethodClose
	}

	// Sort close matches: highest overlap first, then by path for stability.
	sort.SliceStable(closeMatches, func(i, j int) bool {
		if closeMatches[i].overlap != closeMatches[j].overlap {
			return closeMatches[i].overlap > closeMatches[j].overlap
		}
		return closeMatches[i].entry.Path < closeMatches[j].entry.Path
	})

	// Build suggested_fix from the top N close matches.
	if len(closeMatches) > 0 {
		top := closeMatches
		if len(top) > maxSuggestedMatches {
			top = top[:maxSuggestedMatches]
		}
		suggestedFix = formatSuggestion(top)
	}

	// If no prefix-2 matches, fall back to showing up to N routes from the
	// full set so the agent still gets a usable list.
	if len(prefix2) == 0 {
		prefix2 = routes
	}

	// Sort available list deterministically by "METHOD /path".
	sort.SliceStable(prefix2, func(i, j int) bool {
		a := prefix2[i].Method + " " + prefix2[i].Path
		b := prefix2[j].Method + " " + prefix2[j].Path
		return a < b
	})

	// Cap and format.
	if len(prefix2) > maxAvailableRoutes {
		prefix2 = prefix2[:maxAvailableRoutes]
	}
	available = make([]string, 0, len(prefix2))
	for _, r := range prefix2 {
		if r.Path == "" {
			continue
		}
		available = append(available, formatRoute(r.Method, r.Path))
	}

	return suggestedFix, available
}

// pathSegments splits a URL path into non-empty segments. Leading/trailing
// slashes are ignored, so "/api/v1/auth/" and "api/v1/auth" both yield
// ["api", "v1", "auth"].
func pathSegments(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// commonPrefixLen returns the number of leading segments that are equal
// between a and b.
func commonPrefixLen(a, b []string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// formatRoute renders a route as "METHOD /path" for display in errors.
func formatRoute(method, path string) string {
	return fmt.Sprintf("%s %s", method, path)
}

// formatSuggestion builds the "Did you mean ..." string from a sorted list
// of close matches (highest overlap first).
func formatSuggestion(matches []scoredRoute) string {
	if len(matches) == 1 {
		return fmt.Sprintf("Did you mean %s?", formatRoute(matches[0].entry.Method, matches[0].entry.Path))
	}
	parts := make([]string, len(matches))
	for i, m := range matches {
		parts[i] = formatRoute(m.entry.Method, m.entry.Path)
	}
	return fmt.Sprintf("Did you mean one of: %s?", strings.Join(parts, ", "))
}
