# M-AUTH Sprint Plan: Firebase Authentication & Dashboard Authorization

**Sprint ID:** M-AUTH
**Design Doc:** [m-auth-dashboard-firebase.md](m-auth-dashboard-firebase.md)
**Target Version:** v0.7.0
**Estimated Duration:** 5 days (38 hours)
**Total LOC Estimate:** 2,100 lines
**Risk Level:** Medium
**Status:** Ready for Execution

---

## Executive Summary

This sprint implements comprehensive authentication and authorization for the AILANG Collaboration Hub dashboard using Firebase and Firestore. The implementation enables secure multi-tenant deployment to Google Cloud Run with role-based access control (Viewer and Approver roles).

**Key Deliverables:**
1. Firebase JWT verification middleware for backend
2. Firestore access control configuration system
3. React login page with Firebase authentication
4. Protected routes and token management
5. Role-based endpoint authorization
6. Audit logging for all approvals
7. Complete deployment documentation

---

## Milestones

### M1: Firebase Setup & Backend Auth (2 days, 850 LOC)

**Acceptance Criteria:**
- ✅ Firebase project configured with Auth + Firestore
- ✅ Backend JWT verification working with caching
- ✅ Firestore role lookup functional
- ✅ Auth middleware integrated into router
- ✅ All auth tests passing
- ✅ Configuration loading from environment

**Deliverables:**
- `internal/server/auth/auth.go` - JWT verification (~200 LOC)
- `internal/server/auth/firestore.go` - Access control (~150 LOC)
- `internal/server/handlers_auth.go` - Auth middleware (~300 LOC)
- `internal/server/audit.go` - Audit logging (~150 LOC)
- Test files (~250 LOC)
- Updated `go.mod` with Firebase dependencies

**Key Tasks:**
1. Add Firebase Admin SDK to `go.mod`
2. Create `internal/server/auth/auth.go` with:
   - JWT token verification
   - Token caching with 5-minute TTL
   - Error handling for invalid/expired tokens
3. Create `internal/server/auth/firestore.go` with:
   - Firestore client initialization
   - User role lookup from Firestore
   - Workspace membership validation
4. Create `internal/server/handlers_auth.go` with:
   - `AuthMiddleware` that extracts Bearer token
   - Verifies token with Firebase
   - Loads user role from Firestore
   - Returns 401 for invalid tokens
5. Create `internal/server/audit.go` with:
   - Audit log entry recording to Firestore
6. Update `internal/server/server.go` to register auth middleware
7. Update `internal/server/config.go` to load Firebase credentials
8. Write comprehensive tests

**Estimated Hours:** 16 hours
**Risk:** Low-Medium (Firebase SDK reliability)
**Passes:** null

---

### M2: Frontend Authentication (2 days, 650 LOC)

**Acceptance Criteria:**
- ✅ Firebase client SDK initialized
- ✅ Login page with email/password and Google OAuth
- ✅ Protected routes redirect to login
- ✅ Token stored securely in sessionStorage
- ✅ API client includes Authorization header
- ✅ User profile component shows email and role
- ✅ All frontend tests passing

**Deliverables:**
- `src/pages/LoginPage.tsx` (~300 LOC)
- `src/components/FirebaseAuthUI.tsx` (~200 LOC)
- `src/components/ProtectedRoute.tsx` (~100 LOC)
- `src/components/UserProfile.tsx` (~100 LOC)
- `src/config/firebase.ts` (~50 LOC)
- `src/utils/auth.ts` (~150 LOC)
- `src/context/AuthContext.tsx` (~100 LOC)
- Test files (~400 LOC)
- `.env.example` template

**Key Tasks:**
1. Create `src/config/firebase.ts` to initialize Firebase client
2. Create `src/pages/LoginPage.tsx` with:
   - Email/password sign-up and sign-in forms
   - Google OAuth button
   - Error message display
   - Loading state
   - Redirect to dashboard on success
3. Create `src/context/AuthContext.tsx` with:
   - User state management
   - Token storage/retrieval
   - Login/logout functions
4. Create `src/utils/auth.ts` with:
   - `getIdToken()` helper
   - `onAuthStateChanged()` listener
   - Token refresh logic
5. Create `src/components/ProtectedRoute.tsx` with:
   - Auth state checking
   - Redirect to login if not authenticated
   - Loading spinner
6. Update `src/utils/api.ts` to include Authorization header
7. Create `src/components/UserProfile.tsx` for header
8. Update `src/App.tsx` to wrap with AuthProvider
9. Update `package.json` with Firebase dependencies
10. Write comprehensive tests

**Estimated Hours:** 14 hours
**Risk:** Low (React + Firebase libraries mature)
**Passes:** null

---

### M3: Endpoint Authorization (1.5 days, 350 LOC)

**Acceptance Criteria:**
- ✅ All API endpoints require authentication (401 for missing token)
- ✅ GET /api/tasks filters by workspace
- ✅ Approval endpoints require Approver role (403 for Viewer)
- ✅ WebSocket events filtered by workspace
- ✅ Audit logs created for all approvals
- ✅ Proper error responses with status codes
- ✅ All authorization tests passing

**Deliverables:**
- Updated `internal/server/handlers_tasks.go`
- Updated `internal/server/handlers_events.go`
- Updated `internal/server/handlers_coordinator.go`
- Authorization helper functions
- Test files (~200 LOC)

**Key Tasks:**
1. Create authorization helper functions in `handlers_auth.go`:
   - `RequireAuth(next http.Handler) http.Handler`
   - `RequireApprover(next http.Handler) http.Handler`
   - `GetUserFromContext(r *http.Request) (*User, error)`
2. Update `handlers_tasks.go`:
   - Add `RequireAuth` to all endpoints
   - Filter GET /api/tasks by user's workspace
   - Add `RequireApprover` to approval endpoints
   - Call audit logging on approval
3. Update `handlers_events.go`:
   - Filter WebSocket events by user's workspace
4. Update error handling:
   - Return 401 for unauthenticated requests
   - Return 403 for insufficient permissions
   - Include descriptive error messages
5. Write integration tests for all scenarios:
   - Unauthenticated access
   - Viewer trying to approve
   - Approver approving
   - Workspace boundary violations
6. Verify no performance regressions (auth <50ms per request)

**Estimated Hours:** 10 hours
**Risk:** Medium (complex authorization logic)
**Passes:** null

---

### M4: Documentation & Deployment (0.5 days, 250 LOC)

**Acceptance Criteria:**
- ✅ Firebase setup guide complete
- ✅ Deployment instructions for Cloud Run
- ✅ Workspace access control guide written
- ✅ Configuration examples provided
- ✅ Environment variables documented
- ✅ Firestore security rules documented

**Deliverables:**
- `docs/DEPLOYMENT_AUTH.md` (~400 LOC)
- `docs/WORKSPACE_ACCESS_CONTROL.md` (~300 LOC)
- `firestore.rules` (~100 LOC)
- `.env.example` template
- `config.example.yaml` with auth section
- Updated `README.md` and `DEPLOYMENT.md`

**Key Tasks:**
1. Create `docs/DEPLOYMENT_AUTH.md` with:
   - Firebase project setup steps
   - Service account key generation
   - Local Firebase Emulator setup
   - Cloud Run deployment instructions
   - Environment variables reference
   - Troubleshooting guide
2. Create `docs/WORKSPACE_ACCESS_CONTROL.md` with:
   - How to add users to workspace
   - Role descriptions (Viewer vs Approver)
   - Firestore access_control schema
   - Examples with actual commands
3. Create `firestore.rules` with:
   - Security rules for authenticated access only
   - Workspace-scoped access control
   - Audit log restrictions
4. Create configuration templates:
   - `.env.example` for frontend
   - `config.example.yaml` for backend
5. Update `README.md` to note authentication requirement
6. Update `DEPLOYMENT.md` with auth section

**Estimated Hours:** 4 hours
**Risk:** Low (documentation)
**Passes:** null

---

## Implementation Schedule

| Milestone | Duration | Key Dates | Completion Criteria |
|-----------|----------|-----------|-------------------|
| **M1: Firebase Setup & Backend Auth** | 2 days | Day 1-2 | Tests pass, auth middleware working |
| **M2: Frontend Authentication** | 2 days | Day 2-4 | Login page functional, protected routes work |
| **M3: Endpoint Authorization** | 1.5 days | Day 4-5 | All endpoints auth-protected, audit logging |
| **M4: Documentation & Deployment** | 0.5 days | Day 5 | Docs complete, deployment guide ready |
| **Final Testing & Fixes** | 0.5 days | Day 5-6 | All tests passing, no regressions |

---

## Testing Strategy

### Unit Tests
- JWT verification with valid/invalid tokens
- Token caching and TTL expiration
- Firestore role lookup
- Authorization helper functions
- Audit log creation

### Integration Tests
- Complete login flow (UI + backend)
- Protected route access
- API endpoint authorization
- Workspace boundary enforcement
- Approver workflow

### Security Tests
- Unauthenticated access → 401
- Invalid tokens → 401
- Expired tokens → 401
- Viewer accessing approval → 403
- User accessing wrong workspace → 403

### Performance Tests
- JWT verification with caching <1ms
- Firestore lookup with caching <5ms
- Total auth middleware <50ms per request

---

## Dependencies & Prerequisites

### External Services
- Firebase project created and configured
- Google Cloud project with Firestore enabled
- Service account with Firestore read/write permissions

### Code Dependencies
- `github.com/firebase/firebase-admin-go/v4`
- `firebase` NPM package (frontend)
- `react-firebase-hooks` NPM package

### Existing Code
- Current `internal/server/` structure
- Current React app in `ui/`
- Database schema (no changes needed)

---

## Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Firebase outage | Users can't log in | Cache auth decisions, graceful degradation |
| Token leakage | Security breach | Use sessionStorage only, HTTPS enforced |
| Over-permissioned service account | Compromise exposure | Least privilege, dedicated service account |
| Workspace misconfiguration | Data access violation | Validation on every request, audit logs |

---

## Success Metrics

Upon completion, verify:
- ✅ All tests passing (`make test`)
- ✅ No linting errors (`make lint`)
- ✅ 401 responses for unauthenticated requests
- ✅ 403 responses for insufficient permissions
- ✅ Audit logs recording all approvals
- ✅ Users can log in with Google in <3 clicks
- ✅ Performance: auth adds <50ms per request
- ✅ Documentation complete and clear

---

## Files to Create/Modify

### Backend (Go)
**New:**
- `internal/server/auth/auth.go`
- `internal/server/auth/firestore.go`
- `internal/server/auth/auth_test.go`
- `internal/server/auth/firestore_test.go`
- `internal/server/handlers_auth.go`
- `internal/server/audit.go`
- `internal/server/handlers_auth_test.go`

**Modified:**
- `internal/server/config.go`
- `internal/server/handlers_tasks.go`
- `internal/server/handlers_events.go`
- `internal/server/server.go`
- `go.mod`
- `Dockerfile`

### Frontend (React/TypeScript)
**New:**
- `src/pages/LoginPage.tsx`
- `src/components/FirebaseAuthUI.tsx`
- `src/components/ProtectedRoute.tsx`
- `src/components/UserProfile.tsx`
- `src/config/firebase.ts`
- `src/utils/auth.ts`
- `src/context/AuthContext.tsx`
- Tests for above components

**Modified:**
- `src/App.tsx`
- `src/utils/api.ts`
- `package.json`
- `vite.config.ts`

### Configuration & Documentation
**New:**
- `docs/DEPLOYMENT_AUTH.md`
- `docs/WORKSPACE_ACCESS_CONTROL.md`
- `firestore.rules`
- `.env.example`
- `config.example.yaml`

**Modified:**
- `README.md`
- `DEPLOYMENT.md`

---

## Git Commit Strategy

Commits should be organized by milestone:

1. **M1 commits:**
   - "Add Firebase Admin SDK dependencies"
   - "Implement JWT verification and token caching"
   - "Add Firestore access control lookup"
   - "Add auth middleware and configuration"
   - "Add comprehensive auth tests"

2. **M2 commits:**
   - "Add Firebase client configuration"
   - "Implement login page and Firebase UI"
   - "Add auth context and protected routes"
   - "Add user profile component"
   - "Add frontend auth tests"

3. **M3 commits:**
   - "Add authorization helper functions"
   - "Protect API endpoints with role checks"
   - "Add audit logging for approvals"
   - "Add authorization integration tests"

4. **M4 commits:**
   - "Add authentication deployment documentation"
   - "Add workspace access control guide"
   - "Add configuration examples and templates"

---

## Related Documentation

- **Design Doc:** [m-auth-dashboard-firebase.md](m-auth-dashboard-firebase.md)
- **Collaboration Hub:** [global-collaboration-hub.md](global-collaboration-hub.md)
- **Coordinator:** [m-coordinator-always-on-daemon.md](m-coordinator-always-on-daemon.md)

---

## Notes

- This sprint is **critical path** for v0.7.0 release
- Firebase is a cloud service; have Internet connectivity for development
- The Firestore Emulator is recommended for local development
- All auth decisions should be logged for security audit trails
- Token caching is important for performance but must respect TTLs
- Performance targets: auth adds <50ms latency per request
- Security is paramount: no tokens in localStorage, HTTPS only in production
