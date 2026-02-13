package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
)

// ============================================================================
// Topology API Types
// ============================================================================

// TopologyAgent represents an agent in the topology graph
type TopologyAgent struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Status     string  `json:"status"` // idle, busy, blocked, error
	TrustScore int     `json:"trustScore"`
	TaskCount  int     `json:"taskCount"`
	Cost       float64 `json:"cost"`
}

// TopologyEdge represents a connection between agents
type TopologyEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	MessageCount int    `json:"messageCount"`
	LastActivity string `json:"lastActivity,omitempty"`
}

// TopologySink represents a terminal node (approval, main branch)
type TopologySink struct {
	ID           string `json:"id"`
	Label        string `json:"label,omitempty"`
	PendingCount int    `json:"pendingCount,omitempty"`
}

// TopologyResponse is the response for GET /api/controlplane/topology
type TopologyResponse struct {
	Agents []TopologyAgent `json:"agents"`
	Edges  []TopologyEdge  `json:"edges"`
	Sinks  []TopologySink  `json:"sinks"`
}

// ============================================================================
// Observed Topology Types
// ============================================================================

// ObservedTopologyNode represents a node in the observed topology
type ObservedTopologyNode struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	NodeType     string `json:"node_type"` // agent, source, sink
	MessagesSent int    `json:"messages_sent"`
	MessagesRecv int    `json:"messages_recv"`
	LastActivity string `json:"last_activity,omitempty"`
}

// ObservedTopologyEdge represents an edge derived from actual message flows
type ObservedTopologyEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	MessageCount int    `json:"message_count"`
	LastActivity string `json:"last_activity,omitempty"`
	Active       bool   `json:"active"`
}

// ObservedTopologyResponse is the response for GET /api/controlplane/topology/observed
type ObservedTopologyResponse struct {
	Nodes   []ObservedTopologyNode `json:"nodes"`
	Edges   []ObservedTopologyEdge `json:"edges"`
	IsEmpty bool                   `json:"is_empty"`
}

// GET /api/controlplane/topology - Get agent topology graph
func (s *Server) handleControlPlaneTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load agent configuration
	cfg, err := coordinator.LoadCoordinatorConfig()
	if err != nil {
		log.Printf("Failed to load coordinator config: %v", err)
		cfg = coordinator.DefaultCoordinatorConfig()
	}

	// Get coordinator store for queries
	coordStore := s.getCoordStoreForControlPlane()

	// Get running tasks to determine agent status
	runningAgents := make(map[string]bool)
	if coordStore != nil {
		ctx := context.Background()
		filter := &coordinator.TaskFilter{
			Status: []coordinator.TaskStatus{coordinator.TaskStatusRunning, coordinator.TaskStatusQueued},
		}
		tasks, err := coordStore.ListTasks(ctx, filter)
		if err == nil {
			for _, task := range tasks {
				if task.AgentID != "" {
					runningAgents[task.AgentID] = true
				}
			}
		}
	}

	// Get task stats per agent
	agentStats := make(map[string]struct {
		taskCount int
		cost      float64
	})
	if coordStore != nil {
		ctx := context.Background()
		tasks, err := coordStore.ListTasks(ctx, &coordinator.TaskFilter{})
		if err == nil {
			for _, task := range tasks {
				if task.AgentID != "" {
					stats := agentStats[task.AgentID]
					stats.taskCount++
					stats.cost += task.Cost
					agentStats[task.AgentID] = stats
				}
			}
		}
	}

	// Get pending approvals count
	pendingApprovals := 0
	if coordStore != nil {
		ctx := context.Background()
		stats, err := coordStore.GetTaskStats(ctx)
		if err == nil {
			pendingApprovals = stats.PendingApprovals
		}
	}

	// Build topology response
	var agents []TopologyAgent
	var edges []TopologyEdge
	edgeSet := make(map[string]bool) // Track unique edges

	for _, agentCfg := range cfg.Agents {
		// Determine status
		status := "idle"
		if runningAgents[agentCfg.ID] {
			status = "busy"
		}

		// Get stats for this agent
		stats := agentStats[agentCfg.ID]

		// Default trust score (placeholder until trust system is implemented)
		trustScore := 75

		agents = append(agents, TopologyAgent{
			ID:         agentCfg.ID,
			Label:      agentCfg.Label,
			Status:     status,
			TrustScore: trustScore,
			TaskCount:  stats.taskCount,
			Cost:       stats.cost,
		})

		// Build edges from trigger_on_complete
		for _, targetID := range agentCfg.TriggerOnComplete {
			edgeKey := agentCfg.ID + "->" + targetID
			if !edgeSet[edgeKey] {
				edges = append(edges, TopologyEdge{
					Source:       agentCfg.ID,
					Target:       targetID,
					MessageCount: 0, // TODO: Count handoff messages
				})
				edgeSet[edgeKey] = true
			}
		}
	}

	// Add source node (GitHub)
	hasIncomingEdge := make(map[string]bool)
	for _, edge := range edges {
		hasIncomingEdge[edge.Target] = true
	}

	for _, agent := range agents {
		if !hasIncomingEdge[agent.ID] {
			edges = append(edges, TopologyEdge{
				Source:       "github",
				Target:       agent.ID,
				MessageCount: 0,
			})
		}
	}

	// Add sink nodes
	sinks := []TopologySink{
		{
			ID:           "approval",
			Label:        "Approval Queue",
			PendingCount: pendingApprovals,
		},
		{
			ID:    "main",
			Label: "main branch",
		},
	}

	// Add edges to approval sink from agents with no outgoing edges
	hasOutgoingEdge := make(map[string]bool)
	for _, edge := range edges {
		hasOutgoingEdge[edge.Source] = true
	}

	for _, agent := range agents {
		if !hasOutgoingEdge[agent.ID] {
			edges = append(edges, TopologyEdge{
				Source:       agent.ID,
				Target:       "approval",
				MessageCount: 0,
			})
		}
	}

	// Add edge from approval to main
	edges = append(edges, TopologyEdge{
		Source:       "approval",
		Target:       "main",
		MessageCount: 0,
	})

	response := TopologyResponse{
		Agents: agents,
		Edges:  edges,
		Sinks:  sinks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode topology response: %v", err)
	}
}

// Helper to get the full coordinator store interface
func (s *Server) getCoordStoreForControlPlane() coordinator.Store {
	return s.coordStoreRaw
}

// GET /api/controlplane/topology/observed - Get topology derived from actual message flows
// This returns a data-driven graph based on from_agent -> to_inbox message relationships
func (s *Server) handleControlPlaneTopologyObserved(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := ObservedTopologyResponse{
		Nodes: []ObservedTopologyNode{},
		Edges: []ObservedTopologyEdge{},
	}

	// Get message flow edges from the messaging store
	edges, err := s.store.GetMessageFlowEdges()
	if err != nil {
		log.Printf("Failed to get message flow edges: %v", err)
	}

	// Get active agents from the messaging store
	agents, err := s.store.GetActiveAgents()
	if err != nil {
		log.Printf("Failed to get active agents: %v", err)
	}

	// Build nodes from active agents
	nodeMap := make(map[string]bool)
	for _, agent := range agents {
		nodeMap[agent.ID] = true
		response.Nodes = append(response.Nodes, ObservedTopologyNode{
			ID:           agent.ID,
			Label:        formatAgentLabel(agent.ID),
			NodeType:     "agent",
			MessagesSent: agent.MessagesSent,
			MessagesRecv: agent.MessagesRecv,
			LastActivity: agent.LastActivity,
		})
	}

	// Build edges and ensure all nodes exist
	for _, edge := range edges {
		// Add source node if not already present
		if !nodeMap[edge.FromAgent] {
			nodeMap[edge.FromAgent] = true
			response.Nodes = append(response.Nodes, ObservedTopologyNode{
				ID:       edge.FromAgent,
				Label:    formatAgentLabel(edge.FromAgent),
				NodeType: "agent",
			})
		}

		// Add target node if not already present
		if !nodeMap[edge.ToInbox] {
			nodeMap[edge.ToInbox] = true
			response.Nodes = append(response.Nodes, ObservedTopologyNode{
				ID:       edge.ToInbox,
				Label:    formatAgentLabel(edge.ToInbox),
				NodeType: "agent",
			})
		}

		// Determine if edge is active (activity in last 5 minutes)
		active := false
		if edge.LastActivity != "" {
			if t, err := time.Parse(time.RFC3339, edge.LastActivity); err == nil {
				active = time.Since(t) < 5*time.Minute
			}
		}

		response.Edges = append(response.Edges, ObservedTopologyEdge{
			Source:       edge.FromAgent,
			Target:       edge.ToInbox,
			MessageCount: edge.MessageCount,
			LastActivity: edge.LastActivity,
			Active:       active,
		})
	}

	// Detect node types based on edge topology
	hasIncoming := make(map[string]bool)
	hasOutgoing := make(map[string]bool)
	for _, edge := range response.Edges {
		hasIncoming[edge.Target] = true
		hasOutgoing[edge.Source] = true
	}

	// Update node types: sources have no incoming, sinks have no outgoing
	for i := range response.Nodes {
		nodeID := response.Nodes[i].ID
		if !hasIncoming[nodeID] && hasOutgoing[nodeID] {
			response.Nodes[i].NodeType = "source"
		} else if hasIncoming[nodeID] && !hasOutgoing[nodeID] {
			response.Nodes[i].NodeType = "sink"
		}
	}

	response.IsEmpty = len(response.Nodes) == 0

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode observed topology response: %v", err)
	}
}

// formatAgentLabel converts an agent ID to a human-readable label
func formatAgentLabel(agentID string) string {
	// Handle special cases
	switch agentID {
	case "github":
		return "GitHub Issues"
	case "approval":
		return "Approval Queue"
	case "main":
		return "Main Branch"
	case "user":
		return "User"
	}

	// Convert kebab-case to Title Case
	parts := strings.Split(agentID, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}
