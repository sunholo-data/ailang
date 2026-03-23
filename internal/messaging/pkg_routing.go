// Package messaging provides the collaboration hub's messaging infrastructure.
// This file implements package-scoped inbox addressing and routing for the
// package messaging graph (M-PKG-MSG).
package messaging

import (
	"fmt"
	"strings"
)

// InboxAddressType classifies inbox address prefixes.
type InboxAddressType string

const (
	InboxAddrPackage   InboxAddressType = "pkg"
	InboxAddrWorkspace InboxAddressType = "workspace"
	InboxAddrTeam      InboxAddressType = "team"
	InboxAddrPlain     InboxAddressType = "plain"
)

// InboxAddress is a parsed inbox address with type and name.
type InboxAddress struct {
	Type InboxAddressType
	Name string // The part after the prefix (e.g., "sunholo/auth" from "pkg:sunholo/auth")
	Raw  string // Original full address string
}

// ParseInboxAddress parses an inbox address string into its type and name.
// Supported formats:
//
//	"pkg:sunholo/auth"     → {Type: "pkg", Name: "sunholo/auth"}
//	"workspace:docparse"   → {Type: "workspace", Name: "docparse"}
//	"team:registry-admin"  → {Type: "team", Name: "registry-admin"}
//	"user"                 → {Type: "plain", Name: "user"}
func ParseInboxAddress(addr string) InboxAddress {
	for _, prefix := range []InboxAddressType{InboxAddrPackage, InboxAddrWorkspace, InboxAddrTeam} {
		pfx := string(prefix) + ":"
		if strings.HasPrefix(addr, pfx) {
			return InboxAddress{
				Type: prefix,
				Name: strings.TrimPrefix(addr, pfx),
				Raw:  addr,
			}
		}
	}
	return InboxAddress{
		Type: InboxAddrPlain,
		Name: addr,
		Raw:  addr,
	}
}

// FormatPackageInbox returns the canonical inbox address for a package.
func FormatPackageInbox(pkgName string) string {
	return "pkg:" + pkgName
}

// FormatWorkspaceInbox returns the canonical inbox address for a workspace.
func FormatWorkspaceInbox(workspaceName string) string {
	return "workspace:" + workspaceName
}

// FormatTeamInbox returns the canonical inbox address for a team.
func FormatTeamInbox(teamName string) string {
	return "team:" + teamName
}

// ListPackageInboxes returns distinct package-scoped inboxes from the store.
func (s *Store) ListPackageInboxes() ([]string, error) {
	return s.listInboxesByPrefix("pkg:%")
}

// ListWorkspaceInboxes returns distinct workspace-scoped inboxes from the store.
func (s *Store) ListWorkspaceInboxes() ([]string, error) {
	return s.listInboxesByPrefix("workspace:%")
}

func (s *Store) listInboxesByPrefix(prefix string) ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT to_inbox FROM inbox_messages WHERE to_inbox LIKE ? ORDER BY to_inbox`,
		prefix,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query inboxes: %w", err)
	}
	defer rows.Close()

	var inboxes []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err != nil {
			return nil, fmt.Errorf("failed to scan inbox: %w", err)
		}
		inboxes = append(inboxes, inbox)
	}
	return inboxes, rows.Err()
}

// CountPackageMessages returns the count of messages per status for a package inbox.
func (s *Store) CountPackageMessages(pkgName string) (map[string]int, error) {
	inbox := FormatPackageInbox(pkgName)
	rows, err := s.db.Query(
		`SELECT status, COUNT(*) FROM inbox_messages WHERE to_inbox = ? GROUP BY status`,
		inbox,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count package messages: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}
