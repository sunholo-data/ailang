package config

import "strconv"

// `ailang serve-api` (internal/apiserver).
const (
	EnvServeAPIAllowDrops = "AILANG_SERVE_API_ALLOW_DROPS"
	EnvRateLimitRPM       = "AILANG_RATELIMIT_RPM"
	EnvRateLimitBurst     = "AILANG_RATELIMIT_BURST"
)

var apiserverVars = []Var{
	{EnvServeAPIAllowDrops, "0", AreaAPIServer, "1 lets serve-api start with @route-bearing modules dropped for lying outside its base path (not for production)."},
	{EnvRateLimitRPM, "5", AreaAPIServer, "submit_feedback requests per minute per client."},
	{EnvRateLimitBurst, "3", AreaAPIServer, "submit_feedback burst allowance per client."},
}

// ServeAPIAllowDrops reports AILANG_SERVE_API_ALLOW_DROPS=1.
func ServeAPIAllowDrops() bool { return getOr(EnvServeAPIAllowDrops) == "1" }

// RateLimitRPM returns AILANG_RATELIMIT_RPM as an int, default 5; a
// malformed value keeps the default.
func RateLimitRPM() int { return intOr(EnvRateLimitRPM) }

// RateLimitBurst returns AILANG_RATELIMIT_BURST as an int, default 3.
func RateLimitBurst() int { return intOr(EnvRateLimitBurst) }

// intOr parses name as an int, falling back to the registered default when
// the variable is unset or does not parse.
func intOr(name string) int {
	if v := get(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	n, _ := strconv.Atoi(defaultOf(name))
	return n
}
