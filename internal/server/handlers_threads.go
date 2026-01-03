package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// GET /api/threads - List all threads
// POST /api/threads - Create a new thread
func (s *Server) handleThreads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetThreads(w, r)
	case http.MethodPost:
		s.handleCreateThread(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGetThreads(w http.ResponseWriter, r *http.Request) {
	// Get filter params
	status := r.URL.Query().Get("status")
	workspace := r.URL.Query().Get("workspace")

	// Use filtered query if workspace is specified
	if workspace != "" {
		threads, err := s.store.GetThreadsFiltered(s.store.NewThreadFilter(status, workspace, 100))
		if err != nil {
			http.Error(w, "Failed to get threads", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(threads); err != nil {
			log.Printf("Failed to encode threads response: %v", err)
		}
		return
	}

	// Default: filter by status only
	threads, err := s.store.GetThreadsByStatus(status, 100)
	if err != nil {
		http.Error(w, "Failed to get threads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(threads); err != nil {
		log.Printf("Failed to encode threads response: %v", err)
	}
}

func (s *Server) handleCreateThread(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title         string `json:"title"`
		CreatedByType string `json:"created_by_type"`
		CreatedByID   string `json:"created_by_id"`
		TargetAgent   string `json:"target_agent"` // Which agent this conversation is with
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	// Default values
	if body.CreatedByType == "" {
		body.CreatedByType = "human"
	}
	if body.CreatedByID == "" {
		body.CreatedByID = "user"
	}

	thread, err := s.store.CreateThread(body.Title, body.CreatedByType, body.CreatedByID, body.TargetAgent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create thread: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(thread); err != nil {
		log.Printf("Failed to encode thread response: %v", err)
	}
}

// GET /api/threads/{id} - Get a specific thread
// PUT /api/threads/{id} - Update thread settings (workspace, title, etc.)
// DELETE /api/threads/{id} - Delete a thread and all its messages
func (s *Server) handleThread(w http.ResponseWriter, r *http.Request) {
	// Extract thread ID from path
	threadID := r.URL.Path[len("/api/threads/"):]
	if threadID == "" {
		http.Error(w, "Thread ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		thread, err := s.store.GetThread(threadID)
		if err != nil {
			http.Error(w, "Thread not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(thread); err != nil {
			log.Printf("Failed to encode thread response: %v", err)
		}

	case http.MethodPut:
		var body struct {
			Workspace string  `json:"workspace"`
			Title     *string `json:"title"` // Use pointer to distinguish between empty and not provided
		}

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Update thread title if provided
		if body.Title != nil && *body.Title != "" {
			if err := s.store.UpdateThreadTitle(threadID, *body.Title); err != nil {
				http.Error(w, fmt.Sprintf("Failed to update thread title: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// Update thread workspace if provided (can be empty to clear)
		if body.Workspace != "" || body.Title == nil {
			// Only update workspace if explicitly provided or if title wasn't the only update
			if body.Workspace != "" {
				if err := s.store.SetThreadWorkspace(threadID, body.Workspace); err != nil {
					http.Error(w, fmt.Sprintf("Failed to update thread: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}

		// Return updated thread
		thread, err := s.store.GetThread(threadID)
		if err != nil {
			http.Error(w, "Thread not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(thread); err != nil {
			log.Printf("Failed to encode thread response: %v", err)
		}

	case http.MethodDelete:
		// Delete the thread and all associated data
		if err := s.store.DeleteThread(threadID); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Thread not found", http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf("Failed to delete thread: %v", err), http.StatusInternalServerError)
			}
			return
		}

		log.Printf("Deleted thread %s", threadID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Thread %s deleted", threadID),
		}); err != nil {
			log.Printf("Failed to encode delete response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GET /api/workspaces - List all distinct workspaces
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaces, err := s.store.GetDistinctWorkspaces()
	if err != nil {
		http.Error(w, "Failed to get workspaces", http.StatusInternalServerError)
		return
	}

	// Sort for consistent ordering
	sort.Strings(workspaces)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(workspaces); err != nil {
		log.Printf("Failed to encode workspaces response: %v", err)
	}
}
