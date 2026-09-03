package messaging

// ===== Observed Topology Queries =====

// MessageFlowEdge represents an edge between agents based on actual message handoffs
type MessageFlowEdge struct {
	FromAgent    string `json:"from_agent"`
	ToInbox      string `json:"to_inbox"`
	MessageCount int    `json:"message_count"`
	LastActivity string `json:"last_activity,omitempty"`
}

// ActiveAgent represents an agent that has sent or received messages
type ActiveAgent struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	MessagesSent int    `json:"messages_sent"`
	MessagesRecv int    `json:"messages_recv"`
	LastActivity string `json:"last_activity"`
}

// GetMessageFlowEdges returns edges derived from actual from_agent → to_inbox message flows
func (s *Store) GetMessageFlowEdges() ([]MessageFlowEdge, error) {
	rows, err := s.db.Query(`
		SELECT
			from_agent,
			to_inbox,
			COUNT(*) as message_count,
			MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE from_agent IS NOT NULL AND from_agent != ''
		  AND to_inbox IS NOT NULL AND to_inbox != ''
		GROUP BY from_agent, to_inbox
		ORDER BY message_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []MessageFlowEdge
	for rows.Next() {
		var edge MessageFlowEdge
		if err := rows.Scan(&edge.FromAgent, &edge.ToInbox, &edge.MessageCount, &edge.LastActivity); err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

// GetActiveAgents returns agents that have sent or received messages
func (s *Store) GetActiveAgents() ([]ActiveAgent, error) {
	// Get agents who have sent messages
	sentQuery := `
		SELECT from_agent as id, COUNT(*) as count, MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE from_agent IS NOT NULL AND from_agent != ''
		GROUP BY from_agent
	`

	// Get agents who have received messages (using to_inbox as agent id)
	recvQuery := `
		SELECT to_inbox as id, COUNT(*) as count, MAX(created_at) as last_activity
		FROM inbox_messages
		WHERE to_inbox IS NOT NULL AND to_inbox != ''
		GROUP BY to_inbox
	`

	// Combine both using a union and aggregate
	combinedQuery := `
		SELECT
			id,
			id as label,
			COALESCE(SUM(sent_count), 0) as messages_sent,
			COALESCE(SUM(recv_count), 0) as messages_recv,
			MAX(last_activity) as last_activity
		FROM (
			SELECT id, count as sent_count, 0 as recv_count, last_activity FROM (` + sentQuery + `)
			UNION ALL
			SELECT id, 0 as sent_count, count as recv_count, last_activity FROM (` + recvQuery + `)
		)
		GROUP BY id
		ORDER BY messages_sent + messages_recv DESC
	`

	rows, err := s.db.Query(combinedQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []ActiveAgent
	for rows.Next() {
		var agent ActiveAgent
		if err := rows.Scan(&agent.ID, &agent.Label, &agent.MessagesSent, &agent.MessagesRecv, &agent.LastActivity); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}
