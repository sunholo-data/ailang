package policy

import (
	"fmt"
	"net"

	"github.com/sunholo-data/ailang/internal/effects"
)

// checkNetAllowEntry validates one net_allow entry (#1558). Every mode
// refuses a malformed entry by name — at runtime a malformed entry would
// silently admit nothing. Restricted mode additionally admits loopback only
// as a port-qualified LITERAL (127.0.0.1:PORT, [::1]:PORT), which opens that
// one port and nothing else, and refuses private, link-local (the metadata
// server included), unspecified and multicast literals, which restricted mode
// never connects to.
func checkNetAllowEntry(entry, mode string) error {
	host, port, err := effects.ParseNetAllowEntry(entry)
	if err != nil {
		return fmt.Errorf("net_allow entry %q is malformed: %v (want host, host:PORT or [IPv6]:PORT)", entry, err)
	}
	if mode != ModeRestricted {
		return nil
	}
	if effects.IsLoopbackEntryHost(host) {
		if net.ParseIP(host) == nil {
			return fmt.Errorf("net_allow entry %q names loopback by name — restricted mode admits loopback only as a port-qualified literal (127.0.0.1:PORT or [::1]:PORT), which opens that one port", entry)
		}
		if port == "" {
			return fmt.Errorf("net_allow entry %q would open every loopback port — restricted mode admits loopback only port-qualified (127.0.0.1:PORT or [::1]:PORT), which opens that one port; set security_mode = %q to grant all of loopback", entry, ModeTrustedHost)
		}
		return nil
	}
	if ip := net.ParseIP(host); ip != nil {
		var kind string
		switch {
		case ip.IsPrivate():
			kind = "a private"
		case ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast():
			kind = "a link-local"
		case ip.IsUnspecified():
			kind = "the unspecified"
		case ip.IsMulticast():
			kind = "a multicast"
		}
		if kind != "" {
			return fmt.Errorf("net_allow entry %q names %s address, which restricted mode never connects to (no override)", entry, kind)
		}
	}
	return nil
}
