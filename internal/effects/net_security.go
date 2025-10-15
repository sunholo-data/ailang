package effects

import (
	"fmt"
	"net"
)

// validateIP checks if an IP address is allowed for network requests
//
// Security policy (Phase 2 PM - FULL):
//   - Localhost blocked by default (127.x.x.x, ::1) unless ctx.Net.AllowLocalhost
//   - Private IPs always blocked (10.x, 192.168.x, 172.16-31.x)
//   - Link-local blocked (169.254.x.x, fe80::/10)
//   - Multicast blocked
//
// Parameters:
//   - ip: The IP address to validate
//   - ctx: Effect context (for AllowLocalhost flag)
//
// Returns:
//   - nil if IP is allowed
//   - Error with E_NET_IP_BLOCKED if IP is blocked
func validateIP(ip net.IP, ctx *EffContext) error {
	// Localhost check (127.x.x.x, ::1)
	if ip.IsLoopback() {
		if !ctx.Net.AllowLocalhost {
			return fmt.Errorf("E_NET_IP_BLOCKED: localhost IP blocked: %s (use --net-allow-localhost to enable)", ip)
		}
		return nil // Localhost allowed by flag
	}

	// Private IP check (ALWAYS BLOCKED, no override)
	if ip.IsPrivate() {
		return fmt.Errorf("E_NET_IP_BLOCKED: private IP blocked: %s (no override available)", ip)
	}

	// Link-local IPv4: 169.254.x.x
	// Link-local IPv6: fe80::/10
	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("E_NET_IP_BLOCKED: link-local IP blocked: %s", ip)
	}

	// Unspecified (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("E_NET_IP_BLOCKED: unspecified IP blocked: %s", ip)
	}

	// Multicast (224.x.x.x, ff00::/8)
	if ip.IsMulticast() {
		return fmt.Errorf("E_NET_IP_BLOCKED: multicast IP blocked: %s", ip)
	}

	return nil // IP is safe
}

// resolveAndValidateIP resolves a hostname to IP(s) and validates them
//
// This function prevents DNS rebinding attacks by:
//  1. Resolving hostname to IP addresses
//  2. Validating all resolved IPs against security policy
//  3. Returning the first valid IP for dialing
//
// Phase 2 PM: Full DNS rebinding prevention with forced IP dialing
//
// Parameters:
//   - hostname: The hostname to resolve and validate
//   - ctx: Effect context (for AllowLocalhost flag)
//
// Returns:
//   - First valid IP address
//   - Error if DNS fails or all IPs are blocked
func resolveAndValidateIP(hostname string, ctx *EffContext) (string, error) {
	// Special case: raw IP address (skip DNS)
	if ip := net.ParseIP(hostname); ip != nil {
		if err := validateIP(ip, ctx); err != nil {
			return "", err
		}
		return hostname, nil
	}

	// Resolve hostname to IPs
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", fmt.Errorf("E_NET_DNS_FAILED: %w", err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("E_NET_DNS_FAILED: no IPs found for %s", hostname)
	}

	// Validate all resolved IPs (fail if ANY are blocked)
	for _, ip := range ips {
		if err := validateIP(ip, ctx); err != nil {
			return "", fmt.Errorf("E_NET_DNS_REBINDING: %s resolves to blocked IP: %w", hostname, err)
		}
	}

	// Return first valid IP for dialer
	return ips[0].String(), nil
}
