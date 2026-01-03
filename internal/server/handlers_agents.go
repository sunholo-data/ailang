package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
)

// GET /api/agents - List known agent IDs and running agents
// POST /api/agents - Spawn a new agent process
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetAgents(w, r)
	case http.MethodPost:
		s.handleSpawnAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetAgents(w http.ResponseWriter, r *http.Request) {
	// Get known agents from database
	knownAgents, err := s.store.GetKnownAgents()
	if err != nil {
		http.Error(w, "Failed to get agents", http.StatusInternalServerError)
		return
	}

	// Get running agents
	s.agentsMu.RLock()
	runningAgents := make([]*AgentProcess, 0, len(s.agents))
	for _, agent := range s.agents {
		runningAgents = append(runningAgents, agent)
	}
	s.agentsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"known":   knownAgents,
		"running": runningAgents,
	}); err != nil {
		log.Printf("Failed to encode agents response: %v", err)
	}
}

func (s *Server) handleSpawnAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InstanceID string `json:"instance_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	// Check if agent already running
	s.agentsMu.RLock()
	if _, exists := s.agents[body.InstanceID]; exists {
		s.agentsMu.RUnlock()
		http.Error(w, "Agent with this instance_id is already running", http.StatusConflict)
		return
	}
	s.agentsMu.RUnlock()

	// ailang-agent is deprecated - use coordinator instead
	// Return an error directing users to the new approach
	http.Error(w, "ailang-agent is deprecated. Use 'ailang coordinator start' instead. "+
		"See: https://ailang.sunholo.com/docs/guides/collaboration-hub", http.StatusGone)
}

// DELETE /api/agents/{id} - Stop a running agent
func (s *Server) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract instance ID from path
	instanceID := r.URL.Path[len("/api/agents/"):]
	if instanceID == "" {
		http.Error(w, "Instance ID required", http.StatusBadRequest)
		return
	}

	// First, try to find in tracked agents (UI-spawned)
	s.agentsMu.Lock()
	agent, exists := s.agents[instanceID]
	if exists {
		// Kill the tracked process
		if err := agent.cmd.Process.Signal(os.Interrupt); err != nil {
			// Try harder with SIGKILL
			_ = agent.cmd.Process.Kill()
		}
		delete(s.agents, instanceID)
		s.agentsMu.Unlock()
		log.Printf("Stopped tracked agent %s (PID %d)", instanceID, agent.PID)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Agent %s stopped", instanceID),
		})
		return
	}
	s.agentsMu.Unlock()

	// Not a tracked agent - try to extract PID from instance_id pattern
	// Patterns: agent_<pid>, eval_<pid>, run_<pid>, process_<pid>, external_<pid>
	var pid int
	for _, prefix := range []string{"agent_", "eval_", "run_", "process_", "external_"} {
		if len(instanceID) > len(prefix) && instanceID[:len(prefix)] == prefix {
			pidStr := instanceID[len(prefix):]
			// Handle patterns like eval_IMP001_abc12345 (take last numeric part)
			parts := splitLast(pidStr, "_")
			if p, err := strconv.Atoi(parts); err == nil {
				pid = p
				break
			}
		}
	}

	if pid == 0 {
		http.Error(w, "Agent not found or not running", http.StatusNotFound)
		return
	}

	// Kill the discovered process by PID
	proc, err := os.FindProcess(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Process %d not found: %v", pid, err), http.StatusNotFound)
		return
	}

	// Try SIGTERM first, then SIGKILL
	if err := proc.Signal(os.Interrupt); err != nil {
		if err := proc.Kill(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to kill process %d: %v", pid, err), http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Stopped discovered process %s (PID %d)", instanceID, pid)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Process %s (PID %d) stopped", instanceID, pid),
	})
}

// splitLast returns the last segment after underscore, or the whole string if no underscore
func splitLast(s, sep string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return s[i+1:]
		}
	}
	return s
}
