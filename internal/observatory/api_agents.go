package observatory

import (
	"encoding/json"
	"net/http"
)

// ===== Agent Assignment Handlers =====

func (a *API) handleListAgentAssignments(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("taskId")
	agents, err := a.backend.ListAgentAssignments(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agents)
}

func (a *API) handleCreateAgentAssignment(w http.ResponseWriter, r *http.Request) {
	var agent AgentAssignment
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateAgentAssignment(r.Context(), &agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (a *API) handleGetAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := a.backend.GetAgentAssignment(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "agent assignment not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) handleUpdateAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var agent AgentAssignment
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	agent.ID = id
	if err := a.backend.UpdateAgentAssignment(r.Context(), &agent); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) handleDeleteAgentAssignment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteAgentAssignment(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetAgentStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, err := a.backend.GetAgentStats(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
