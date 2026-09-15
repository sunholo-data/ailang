package observatory

import (
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/httpjson"
	"net/http"
)

// ===== Workspace Handlers =====

func (a *API) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := a.backend.ListWorkspaces(r.Context())
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpjson.Write(w, http.StatusOK, workspaces)
}

func (a *API) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var ws Workspace
	if err := json.NewDecoder(r.Body).Decode(&ws); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.backend.CreateWorkspace(r.Context(), &ws); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpjson.Write(w, http.StatusCreated, ws)
}

func (a *API) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ws, err := a.backend.GetWorkspace(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			httpjson.Error(w, http.StatusNotFound, "workspace not found")
			return
		}
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpjson.Write(w, http.StatusOK, ws)
}

func (a *API) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var ws Workspace
	if err := json.NewDecoder(r.Body).Decode(&ws); err != nil {
		httpjson.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	ws.ID = id
	if err := a.backend.UpdateWorkspace(r.Context(), &ws); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpjson.Write(w, http.StatusOK, ws)
}

func (a *API) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteWorkspace(r.Context(), id); err != nil {
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleGetWorkspaceStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stats, err := a.backend.GetWorkspaceStats(r.Context(), id)
	if err != nil {
		httpjson.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpjson.Write(w, http.StatusOK, stats)
}
