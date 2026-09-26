package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// M-PKG-MULTI-NAMESPACE-AUTH — scoped registry keys.
//
// Every write request resolves to exactly one principal:
//
//   - the superuser key (REGISTRY_API_KEY env) → owner "superuser", scope "*"
//   - a scoped key whose SHA-256 is a document in the key store → that doc's
//     owner + scopes
//   - anything else → 403
//
// Scopes are path.Match globs over the full package name ("daneel/*",
// "sunholo/daneel_*"). Reads are not gated here at all — the bucket is public.

const (
	keyPrefix      = "ailr_"
	superuserOwner = "superuser"
)

// principal is the resolved identity behind an X-API-Key header.
type principal struct {
	ID     string // doc id (hex sha256); empty for superuser
	Owner  string
	Scopes []string
}

func (p principal) isSuperuser() bool { return p.Owner == superuserOwner }

// canWrite reports whether this principal may publish/unpublish pkgName.
func (p principal) canWrite(pkgName string) bool {
	for _, s := range p.Scopes {
		if s == "*" {
			return true
		}
		if ok, err := path.Match(s, pkgName); err == nil && ok {
			return true
		}
	}
	return false
}

// keyRecord is the stored shape of a scoped key. The plaintext key is never
// stored — only its SHA-256 (the document id).
type keyRecord struct {
	ID        string     `firestore:"-" json:"id"`
	Owner     string     `firestore:"owner" json:"owner"`
	Scopes    []string   `firestore:"scopes" json:"scopes"`
	Note      string     `firestore:"note,omitempty" json:"note,omitempty"`
	CreatedBy string     `firestore:"created_by" json:"created_by"`
	CreatedAt time.Time  `firestore:"created_at" json:"created_at"`
	RevokedAt *time.Time `firestore:"revoked_at" json:"revoked_at,omitempty"`
}

// keyStore is the persistence seam. Firestore in prod, memory in tests.
type keyStore interface {
	Get(ctx context.Context, id string) (*keyRecord, error) // errKeyNotFound if absent
	Put(ctx context.Context, id string, rec *keyRecord) error
	List(ctx context.Context) ([]*keyRecord, error)
	Revoke(ctx context.Context, id string, at time.Time) error // errKeyNotFound if absent
}

var errKeyNotFound = errors.New("key not found")

// authError carries the HTTP status the failed authorization should produce.
type authError struct {
	status int
	msg    string
}

func (e *authError) Error() string { return e.msg }

func requestAPIKey(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return k
	}
	return r.URL.Query().Get("api_key")
}

func keyID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// authenticate resolves the request's key to a principal. It does not check
// scopes — call principal.canWrite for that, or authorizeWrite for both.
func (v *validator) authenticate(ctx context.Context, r *http.Request) (principal, *authError) {
	provided := requestAPIKey(r)
	if provided == "" {
		return principal{}, &authError{http.StatusForbidden, "Invalid or missing API key"}
	}
	if v.apiKey != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(v.apiKey)) == 1 {
		return principal{Owner: superuserOwner, Scopes: []string{"*"}}, nil
	}
	if !strings.HasPrefix(provided, keyPrefix) {
		return principal{}, &authError{http.StatusForbidden, "Invalid or missing API key"}
	}
	if v.keys == nil {
		// No silent fallback: a scoped key on a server without a key store is a
		// deployment error, and the caller should be told that rather than
		// "invalid key".
		return principal{}, &authError{http.StatusForbidden,
			"scoped API keys are not configured on this server (FIRESTORE_DATABASE unset)"}
	}
	id := keyID(provided)
	rec, err := v.keys.Get(ctx, id)
	if errors.Is(err, errKeyNotFound) {
		return principal{}, &authError{http.StatusForbidden, "Invalid or missing API key"}
	}
	if err != nil {
		return principal{}, &authError{http.StatusInternalServerError, fmt.Sprintf("key lookup failed: %v", err)}
	}
	if rec.RevokedAt != nil {
		return principal{}, &authError{http.StatusForbidden, fmt.Sprintf("API key for %q was revoked", rec.Owner)}
	}
	return principal{ID: id, Owner: rec.Owner, Scopes: rec.Scopes}, nil
}

// authorizeWrite authenticates and then checks the principal may write pkgName.
func (v *validator) authorizeWrite(ctx context.Context, r *http.Request, pkgName string) (principal, *authError) {
	p, aerr := v.authenticate(ctx, r)
	if aerr != nil {
		return principal{}, aerr
	}
	if !p.canWrite(pkgName) {
		return principal{}, &authError{http.StatusForbidden,
			fmt.Sprintf("API key for %q is not authorized to write %s (scopes: %s)",
				p.Owner, pkgName, strings.Join(p.Scopes, ", "))}
	}
	return p, nil
}

// requireSuperuser gates admin endpoints and /rebuild-index.
func (v *validator) requireSuperuser(ctx context.Context, r *http.Request) *authError {
	if v.apiKey == "" {
		return &authError{http.StatusForbidden, "this endpoint requires REGISTRY_API_KEY to be configured on the server"}
	}
	p, aerr := v.authenticate(ctx, r)
	if aerr != nil {
		return aerr
	}
	if !p.isSuperuser() {
		return &authError{http.StatusForbidden, fmt.Sprintf("API key for %q is not the superuser key", p.Owner)}
	}
	return nil
}

// newScopedKey generates a fresh key. Returns (plaintext, id).
func newScopedKey() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	key := keyPrefix + base64.RawURLEncoding.EncodeToString(raw)
	return key, keyID(key), nil
}

// validateScopes rejects patterns path.Match cannot parse and shapes that
// can never match a "vendor/name" package.
func validateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return errors.New("at least one scope is required")
	}
	for _, s := range scopes {
		if s == "*" {
			return errors.New(`scope "*" is reserved for the superuser key; use "vendor/*"`)
		}
		if _, err := path.Match(s, "vendor/name"); err != nil {
			return fmt.Errorf("scope %q: %v", s, err)
		}
		if !strings.Contains(s, "/") {
			return fmt.Errorf("scope %q must be of the form vendor/name or vendor/*", s)
		}
	}
	return nil
}

// ── Firestore store ─────────────────────────────────────────

type firestoreKeyStore struct {
	col *firestore.CollectionRef
}

func newFirestoreKeyStore(ctx context.Context, project, database, collection string) (*firestoreKeyStore, error) {
	client, err := firestore.NewClientWithDatabase(ctx, project, database)
	if err != nil {
		return nil, err
	}
	return &firestoreKeyStore{col: client.Collection(collection)}, nil
}

func (s *firestoreKeyStore) Get(ctx context.Context, id string) (*keyRecord, error) {
	snap, err := s.col.Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, errKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	var rec keyRecord
	if err := snap.DataTo(&rec); err != nil {
		return nil, err
	}
	rec.ID = id
	return &rec, nil
}

func (s *firestoreKeyStore) Put(ctx context.Context, id string, rec *keyRecord) error {
	_, err := s.col.Doc(id).Set(ctx, rec)
	return err
}

func (s *firestoreKeyStore) List(ctx context.Context) ([]*keyRecord, error) {
	var out []*keyRecord
	it := s.col.Documents(ctx)
	for {
		snap, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var rec keyRecord
		if err := snap.DataTo(&rec); err != nil {
			return nil, err
		}
		rec.ID = snap.Ref.ID
		out = append(out, &rec)
	}
	return out, nil
}

func (s *firestoreKeyStore) Revoke(ctx context.Context, id string, at time.Time) error {
	_, err := s.col.Doc(id).Update(ctx, []firestore.Update{{Path: "revoked_at", Value: at}})
	if status.Code(err) == codes.NotFound {
		return errKeyNotFound
	}
	return err
}
