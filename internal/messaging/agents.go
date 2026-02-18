package messaging

import "time"

// RegisterAgent registers or updates an agent in the collaboration hub.
// If the agent already exists, updates its status and last_active_at timestamp.
func (s *Store) RegisterAgent(agentID, label, status string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO agents (id, label, status, created_at, updated_at, last_active_at, config_json)
		VALUES (?, ?, ?, ?, ?, ?, '{}')
		ON CONFLICT(id) DO UPDATE SET status=?, label=?, updated_at=?, last_active_at=?
	`, agentID, label, status, now, now, now, status, label, now, now)
	return err
}

// UpdateAgentStatus updates the status and last_active_at for an agent.
func (s *Store) UpdateAgentStatus(agentID, status string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		UPDATE agents SET status=?, updated_at=? WHERE id=?
	`, status, now, agentID)
	return err
}

// RecordAgentInstance creates an instance history entry for an agent run.
func (s *Store) RecordAgentInstance(agentID, instanceID string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO instance_history (id, agent_id, instance_id, started_at)
		VALUES (?, ?, ?, ?)
	`, instanceID, agentID, instanceID, now)
	return err
}
