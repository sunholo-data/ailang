package apiserver

import (
	"strings"
	"testing"
)

// routeFixture builds a minimal RouteEntry for tests — only Method and Path
// are relevant to suggestRoutes.
func routeFixture(method, path string) RouteEntry {
	return RouteEntry{Method: method, Path: path}
}

// TestSuggestRoutes_ExactMissWithClose is the docparse scenario: a request
// to /api/v1/auth/device/token should suggest /api/v1/auth/device/poll.
func TestSuggestRoutes_ExactMissWithClose(t *testing.T) {
	routes := []RouteEntry{
		routeFixture("POST", "/api/v1/auth/device"),
		routeFixture("POST", "/api/v1/auth/device/poll"),
		routeFixture("POST", "/api/v1/auth/device/approve"),
		routeFixture("GET", "/api/v1/health"),
	}

	fix, available := suggestRoutes("POST", "/api/v1/auth/device/token", routes)

	if fix == "" {
		t.Fatal("expected suggested_fix to be populated, got empty")
	}
	if !strings.Contains(fix, "/api/v1/auth/device/poll") && !strings.Contains(fix, "/api/v1/auth/device/approve") {
		t.Errorf("suggested_fix should mention a close /device/* route, got: %q", fix)
	}
	if !strings.HasPrefix(fix, "Did you mean") {
		t.Errorf("suggested_fix should start with 'Did you mean', got: %q", fix)
	}

	// Available list should include the /api/v1/* routes but not unrelated ones.
	wantInAvailable := []string{
		"POST /api/v1/auth/device",
		"POST /api/v1/auth/device/poll",
		"POST /api/v1/auth/device/approve",
	}
	for _, w := range wantInAvailable {
		if !containsString(available, w) {
			t.Errorf("available_routes missing %q, got: %v", w, available)
		}
	}
}

// TestSuggestRoutes_NoCloseMatch verifies that a request with no close match
// gets empty suggested_fix but still gets a usable available_routes list.
func TestSuggestRoutes_NoCloseMatch(t *testing.T) {
	routes := []RouteEntry{
		routeFixture("GET", "/health"),
		routeFixture("GET", "/metrics"),
	}

	fix, available := suggestRoutes("POST", "/api/v1/auth/device/token", routes)

	if fix != "" {
		t.Errorf("expected empty suggested_fix for no close match, got: %q", fix)
	}
	// Falls back to returning all routes (since nothing shares 2 prefix segments).
	if len(available) == 0 {
		t.Error("expected available_routes to fall back to full list, got empty")
	}
}

// TestSuggestRoutes_WrongMethod verifies that when the method doesn't match
// any close route, we fall back to suggesting the any-method match.
func TestSuggestRoutes_WrongMethod(t *testing.T) {
	routes := []RouteEntry{
		routeFixture("GET", "/api/v1/users"),
		routeFixture("GET", "/api/v1/users/list"),
	}

	fix, _ := suggestRoutes("POST", "/api/v1/users", routes)

	if fix == "" {
		t.Fatal("expected a fallback any-method suggestion, got empty")
	}
	// Should suggest the GET route even though request was POST.
	if !strings.Contains(fix, "GET /api/v1/users") {
		t.Errorf("expected GET fallback suggestion, got: %q", fix)
	}
}

// TestSuggestRoutes_EmptyRegistry verifies safe behavior on a server with no
// registered routes at all.
func TestSuggestRoutes_EmptyRegistry(t *testing.T) {
	fix, available := suggestRoutes("POST", "/api/v1/foo", nil)

	if fix != "" {
		t.Errorf("expected empty suggested_fix for empty registry, got: %q", fix)
	}
	if len(available) != 0 {
		t.Errorf("expected empty available for empty registry, got: %v", available)
	}
}

// TestSuggestRoutes_SingleRoute verifies that even with one registered route,
// we suggest it if it overlaps the request path.
func TestSuggestRoutes_SingleRoute(t *testing.T) {
	routes := []RouteEntry{
		routeFixture("POST", "/api/v1/auth/device/poll"),
	}

	fix, available := suggestRoutes("POST", "/api/v1/auth/device/token", routes)

	if fix == "" {
		t.Fatal("expected suggestion with single overlapping route")
	}
	if !strings.Contains(fix, "/api/v1/auth/device/poll") {
		t.Errorf("expected suggestion for poll, got: %q", fix)
	}
	if len(available) != 1 {
		t.Errorf("expected 1 available route, got: %v", available)
	}
}

// TestSuggestRoutes_CapsAvailable verifies that a server with many routes
// under the same prefix returns at most maxAvailableRoutes entries.
func TestSuggestRoutes_CapsAvailable(t *testing.T) {
	var routes []RouteEntry
	for i := 0; i < 25; i++ {
		routes = append(routes, routeFixture("GET", "/api/v1/users/item-"+string(rune('a'+i))))
	}

	_, available := suggestRoutes("GET", "/api/v1/users/nonexistent", routes)

	if len(available) > maxAvailableRoutes {
		t.Errorf("available_routes not capped: got %d, max %d", len(available), maxAvailableRoutes)
	}
	if len(available) != maxAvailableRoutes {
		t.Errorf("expected exactly %d available routes, got %d", maxAvailableRoutes, len(available))
	}
}

// TestSuggestRoutes_MultipleClose verifies that multiple close matches are
// formatted as "Did you mean one of: ...".
func TestSuggestRoutes_MultipleClose(t *testing.T) {
	routes := []RouteEntry{
		routeFixture("POST", "/api/v1/auth/device/poll"),
		routeFixture("POST", "/api/v1/auth/device/approve"),
		routeFixture("POST", "/api/v1/auth/device/reject"),
	}

	fix, _ := suggestRoutes("POST", "/api/v1/auth/device/token", routes)

	if !strings.Contains(fix, "one of") {
		t.Errorf("expected 'one of' in multi-match suggestion, got: %q", fix)
	}
	// Should cap at maxSuggestedMatches — all 3 are close, all should appear.
	for _, want := range []string{"/poll", "/approve", "/reject"} {
		if !strings.Contains(fix, want) {
			t.Errorf("expected %s in suggestion, got: %q", want, fix)
		}
	}
}

// TestPathSegments verifies the helper handles edge cases.
func TestPathSegments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/api/v1/auth", []string{"api", "v1", "auth"}},
		{"api/v1/auth", []string{"api", "v1", "auth"}},
		{"/api/v1/auth/", []string{"api", "v1", "auth"}},
		{"/", nil},
		{"", nil},
		{"/single", []string{"single"}},
	}
	for _, tc := range cases {
		got := pathSegments(tc.in)
		if !equalStrings(got, tc.want) {
			t.Errorf("pathSegments(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestCommonPrefixLen verifies segment-level prefix matching.
func TestCommonPrefixLen(t *testing.T) {
	cases := []struct {
		a, b []string
		want int
	}{
		{[]string{"api", "v1", "auth"}, []string{"api", "v1", "auth", "poll"}, 3},
		{[]string{"api", "v1", "auth", "token"}, []string{"api", "v1", "auth", "poll"}, 3},
		{[]string{"api", "v1"}, []string{"api", "v2"}, 1},
		{nil, []string{"api"}, 0},
		{[]string{"foo"}, []string{"bar"}, 0},
	}
	for _, tc := range cases {
		got := commonPrefixLen(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("commonPrefixLen(%v, %v) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// --- helpers ---

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
