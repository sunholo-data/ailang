package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

// Admin API — superuser only. Mints, lists and revokes scoped keys.
//
//	POST   /admin/keys        {owner, scopes, note}  → {id, key, owner, scopes}
//	GET    /admin/keys                               → {keys: [...]}
//	DELETE /admin/keys/<id>                          → {id, revoked_at}
//
// The plaintext key appears in exactly one response: the POST that minted it.

type createKeyRequest struct {
	Owner  string   `json:"owner"`
	Scopes []string `json:"scopes"`
	Note   string   `json:"note,omitempty"`
}

func (v *validator) handleAdminKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if aerr := v.requireSuperuser(ctx, r); aerr != nil {
		jsonError(w, aerr.status, "%s", aerr.msg)
		return
	}
	if v.keys == nil {
		jsonError(w, http.StatusServiceUnavailable, "scoped API keys are not configured on this server (FIRESTORE_DATABASE unset)")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/admin/keys")
	id = strings.Trim(id, "/")

	switch {
	case r.Method == http.MethodPost && id == "":
		v.adminCreateKey(w, r)
	case r.Method == http.MethodGet && id == "":
		v.adminListKeys(w, r)
	case r.Method == http.MethodDelete && id != "":
		v.adminRevokeKey(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (v *validator) adminCreateKey(w http.ResponseWriter, r *http.Request) {
	var req createKeyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body: %v", err)
		return
	}
	req.Owner = strings.TrimSpace(req.Owner)
	if req.Owner == "" || req.Owner == superuserOwner {
		jsonError(w, http.StatusBadRequest, "owner is required and may not be %q", superuserOwner)
		return
	}
	if err := validateScopes(req.Scopes); err != nil {
		jsonError(w, http.StatusBadRequest, "%v", err)
		return
	}

	key, id, err := newScopedKey()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "key generation failed: %v", err)
		return
	}
	rec := &keyRecord{
		Owner:     req.Owner,
		Scopes:    req.Scopes,
		Note:      req.Note,
		CreatedBy: superuserOwner,
		CreatedAt: time.Now().UTC(),
	}
	if err := v.keys.Put(r.Context(), id, rec); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to store key: %v", err)
		return
	}
	log.Printf("Minted scoped key %s for %q (scopes: %s)", id[:12], req.Owner, strings.Join(req.Scopes, ", "))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     id,
		"key":    key,
		"owner":  rec.Owner,
		"scopes": rec.Scopes,
	})
}

func (v *validator) adminListKeys(w http.ResponseWriter, r *http.Request) {
	recs, err := v.keys.List(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list keys: %v", err)
		return
	}
	if recs == nil {
		recs = []*keyRecord{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"keys": recs})
}

func (v *validator) adminRevokeKey(w http.ResponseWriter, r *http.Request, id string) {
	at := time.Now().UTC()
	err := v.keys.Revoke(r.Context(), id, at)
	if errors.Is(err, errKeyNotFound) {
		jsonError(w, http.StatusNotFound, "key %s not found", id)
		return
	}
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to revoke key: %v", err)
		return
	}
	log.Printf("Revoked scoped key %s", id[:min(12, len(id))])
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "revoked_at": at.Format(time.RFC3339)})
}
