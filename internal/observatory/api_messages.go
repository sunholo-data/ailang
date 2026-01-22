package observatory

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ===== Message Handlers =====

func (a *API) handleListMessages(w http.ResponseWriter, r *http.Request) {
	opts := MessageListOptions{
		Inbox:     r.URL.Query().Get("inbox"),
		Status:    MessageStatus(r.URL.Query().Get("status")),
		FromAgent: r.URL.Query().Get("from"),
		TaskID:    r.URL.Query().Get("task_id"),
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			opts.Limit = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			opts.Offset = o
		}
	}

	messages, err := a.backend.ListMessages(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (a *API) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if err := a.backend.CreateMessage(r.Context(), &message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, message)
}

func (a *API) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	message, err := a.backend.GetMessage(r.Context(), id)
	if err != nil {
		if isNotFoundError(err) {
			writeError(w, http.StatusNotFound, "message not found: "+id)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func (a *API) handleUpdateMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var message Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	message.ID = id
	if err := a.backend.UpdateMessage(r.Context(), &message); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, message)
}

func (a *API) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.DeleteMessage(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkMessageRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.MarkMessageRead(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleMarkMessageArchived(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.backend.MarkMessageArchived(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
