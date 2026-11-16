package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "healthy",
		"version":     s.version,
		"connections": s.wsServer.GetConnectionCount(),
		"timestamp":   time.Now().Unix(),
	}); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// handleVersion returns the AILANG version
// GET /api/version
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"version": s.version,
	}); err != nil {
		log.Printf("Failed to encode version response: %v", err)
	}
}
