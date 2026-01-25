# M-AUTH: Firebase Authentication & Role-Based Authorization for Dashboard

**Status:** Implemented
**Target:** v0.7.0
**Priority:** P1 (High - blocks public deployment)
**Estimated:** 5 days
**Dependencies:** Global Collaboration Hub (v0.7.0), Google Cloud Run deployment

**Created:** January 22, 2026
**Last Modified:** January 23, 2026
**Implemented:** January 22, 2026

---

## Executive Summary

This design document specifies a comprehensive authentication and authorization system for the AILANG Collaboration Hub dashboard, enabling secure deployment to Google Cloud Run with multi-workspace access control and role-based permissions.

**Problem:** Currently, the dashboard has no authentication. When deployed to Cloud Run as a public service, any user can view all workspaces and approve/reject any task. This is a critical security gap for multi-tenant deployments.

**Solution:** Integrate Firebase Authentication for user verification and implement workspace-scoped access control with two role tiers:
- **Viewer**: Read-only access to tasks and events
- **Approver**: Full approval authority (merge/reject) + read access

**Key Features:**
1. Firebase Email/Password and OAuth (Google) authentication
2. Workspace-level access control (configured per user email)
3. Role-based permissions (Viewer vs Approver)
4. Automatic project scoping (public projects visible to all, private projects restricted)
5. Session management with JWT tokens
6. API-level authorization checks
7. Google Cloud Run compatible configuration

---

## Current State

**Dashboard Architecture:**
- Frontend: React + TypeScript (ui/)
- Backend: Go HTTP server (internal/server/)
- Database: SQLite (`~/.ailang/state/collaboration.db`)
- Current auth: None (anyone can access)
- Deployment: Cloud Run (public, no authentication)

**Collaboration Hub Features:**
- Real-time task execution monitoring
- Approval queue with git diff viewer
- Event streaming via WebSocket
- Multi-agent coordination
- Message/thread management

**Current Endpoints (Unauthenticated):**
- `GET /` - Dashboard UI
- `GET /api/tasks` - List all tasks
- `GET /api/tasks/:id` - Task details
- `POST /api/tasks/:id/approve` - Approve task
- `POST /api/tasks/:id/reject` - Reject task
- `WebSocket /ws` - Real-time events

---

## Problem Statement

### Security Gap
Current dashboard deployment is completely open. When deployed to Cloud Run:
- ❌ Any internet user can view all tasks, agent assignments, messages
- ❌ Any user can approve/reject any task in any workspace
- ❌ No audit trail of who made approvals
- ❌ Private projects are visible to unauthenticated users

### Multi-Tenant Requirements
AILANG may be deployed in multiple scenarios:
1. **Personal**: Single user, private workspace
2. **Team**: Multiple developers, shared AILang repo (ailang-core)
3. **Multiple Projects**: Public projects (everyone reads) + private projects (team only)
4. **External Integrations**: Contractors with read-only access

### Current Limitations
- No concept of "users" in system
- No workspace/project scoping
- All API endpoints return all data
- No permission enforcement
- Session management would be ad-hoc

---

## Goals

### Primary Goal
Enable secure, multi-tenant deployment of AILANG Collaboration Hub to Google Cloud Run with fine-grained access control based on GitHub workspace configuration.

### Success Metrics
- [ ] **Security**: Zero unauthenticated access to API endpoints
- [ ] **Usability**: Users can log in with Google account in <3 clicks
- [ ] **Access Control**: All endpoint tests verify role-based access
- [ ] **Audit**: All approvals logged with user identity
- [ ] **Performance**: Auth check + token validation <50ms per request
- [ ] **Integration**: Works seamlessly with existing coordinator approval workflow

### Non-Goals
- SSO via SAML or OIDC providers (v0.8.0+)
- Fine-grained resource-level permissions
- Permission delegation or delegation chains
- OAuth2 client credentials flow (service accounts)
- RBAC beyond Viewer/Approver

---

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React)                        │
│  - Login page (Firebase UI)                                 │
│  - Dashboard (task list, approvals, events)                 │
└────────────────────────────┬────────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │ Firebase Client │
                    │ SDK (web v9)    │
                    │ getIdToken()    │
                    └────────┬────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │  Authorization: Bearer <JWT token>      │
        │  X-Workspace: <workspace-id>            │
        │  X-Project: <project-id> (optional)     │
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │       Go HTTP Server (internal/server/) │
        │ ┌──────────────────────────────────────┤
        │ │ Middleware: AuthMiddleware            │
        │ │ - Verify Firebase JWT                 │
        │ │ - Extract user UID, email             │
        │ │ - Check workspace access              │
        │ │ - Check role permissions              │
        │ │ - Set ctx.User, ctx.Workspace         │
        │ └──────────────────────────────────────┤
        │ ┌──────────────────────────────────────┤
        │ │ Handlers: GET /api/tasks              │
        │ │ - Require authenticated user          │
        │ │ - Filter by workspace + role          │
        │ │ - Only Approver can see approval UI   │
        │ └──────────────────────────────────────┤
        └────────────────────┬────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │    Firestore (User Access Config)       │
        │ Collections:                             │
        │ - workspaces/{ws}/access_control        │
        │   Document: {email, role, created_by}   │
        │ - audit_log                             │
        │   Document: {user, action, workspace}   │
        └────────────────────────────────────────┘
                             │
        ┌────────────────────▼────────────────────┐
        │  SQLite (Existing Coordinator Data)     │
        │  - No auth changes to schema             │
        │  - Auth layer filters results           │
        └────────────────────────────────────────┘
```

### Key Components

#### 1. Firebase Configuration
- **Project**: Use existing Google Cloud project
- **Auth Methods**:
  - Email/Password (for development)
  - Google OAuth (for deployment)
- **Firestore**: Store workspace access control rules
- **Security Rules**: Restrict Firestore access to authenticated users

#### 2. Frontend Changes (ui/)
- **Login Page** (`src/pages/LoginPage.tsx`):
  - Firebase Authentication UI (FirebaseUI or custom form)
  - Social login (Google)
  - Email/password fallback
  - Remember me (via localStorage JWT)

- **Layout Component** (wrap all protected routes):
  - Check Firebase auth state on mount
  - Redirect unauthenticated users to login
  - Show user profile menu (email, role, logout)

- **Approval UI**: Show approval buttons only if user has `Approver` role

#### 3. Backend Changes (internal/server/)
- **Auth Middleware** (`handlers_auth.go`):
  ```go
  type User struct {
    UID      string // Firebase UID
    Email    string
    Role     Role   // Viewer or Approver
    Workspace string
    Project   string
  }

  type Role string
  const (
    RoleViewer   Role = "viewer"
    RoleApprover Role = "approver"
  )
  ```

- **JWT Verification**:
  - Verify Firebase ID token
  - Extract user UID and email
  - Cache verification results (5 min TTL)
  - Handle token refresh

- **Access Control**:
  - Fetch user role from Firestore
  - Check workspace membership
  - Enforce role-based endpoint access

#### 4. Firestore Schema
```firestore
workspaces/{workspace_id}/
  access_control/
    {document_id}:
      email: "user@example.com"
      role: "approver" | "viewer"
      created_by: "admin-uid"
      created_at: timestamp
      modified_at: timestamp

audit_log/{entry_id}:
  user_uid: "firebase-uid"
  user_email: "user@example.com"
  workspace: "workspace-id"
  action: "approve" | "reject" | "login"
  task_id: "task-123"
  details: {...}
  timestamp: timestamp
```

#### 5. Configuration
Store in `~/.ailang/config.yaml`:
```yaml
auth:
  provider: firebase          # Options: firebase, oauth2

firebase:
  project_id: "project-id"
  api_key: "web-api-key"
  auth_domain: "project.firebaseapp.com"

  # Cloud Run environment variable injection
  # Prefer: FIREBASE_PROJECT_ID environment variable

  # For local development
  emulator:
    enabled: false            # Set to true for local testing
    auth_host: "localhost:9099"
    firestore_host: "localhost:8080"

workspace_config:
  # Map workspace ID to Firestore collection path
  workspace_mappings:
    "default": "workspaces/ailang-core"
    "staging": "workspaces/ailang-staging"

  # Public workspaces (no auth required for read)
  public_workspaces: ["docs"]
```

---

## Implementation Plan

### Phase 1: Firebase Setup & Backend Auth (2 days)

**Deliverables:**
- Firebase project configured
- Backend JWT verification working
- Firestore access control schema
- API authentication middleware

**Tasks:**

1. **Firebase Configuration** (~2 hours)
   - [ ] Create Firebase project (or use existing)
   - [ ] Enable Authentication (Email/Password + Google)
   - [ ] Enable Firestore database
   - [ ] Create web app (get API keys)
   - [ ] Generate service account key for backend
   - [ ] Document Firebase setup in DEPLOYMENT.md

2. **Backend Auth Middleware** (~4 hours)
   - [ ] Create `internal/server/auth/auth.go`
     - Firebase JWT verification
     - Token caching with expiration
     - Error handling
   - [ ] Create `internal/server/auth/firestore.go`
     - Firestore client initialization
     - User role lookup
     - Access control validation
   - [ ] Create context types:
     ```go
     type contextKey string
     const UserContextKey contextKey = "user"
     ```
   - [ ] Add error types:
     ```go
     var (
       ErrUnauthenticated = errors.New("authentication required")
       ErrUnauthorized    = errors.New("insufficient permissions")
       ErrInvalidToken    = errors.New("invalid Firebase token")
     )
     ```

3. **Authentication Middleware** (~3 hours)
   - [ ] Create `internal/server/handlers_auth.go`
     - Middleware: `AuthMiddleware(next http.Handler) http.Handler`
     - Extract Bearer token from Authorization header
     - Verify token with Firebase
     - Load user role from Firestore
     - Set user context
     - Return 401 if invalid
   - [ ] Add middleware to router
   - [ ] Skip auth for public endpoints (`/health`, static assets)

4. **Firestore Schema & Access Control** (~3 hours)
   - [ ] Firestore security rules (TypeScript rules format):
     ```typescript
     rules_version = '2';
     service cloud.firestore {
       match /databases/{database}/documents {
         // Only authenticated users can read
         match /workspaces/{workspace}/access_control/{document} {
           allow read: if request.auth != null;
           allow write: if request.auth.token.admin == true;
         }
         match /audit_log/{entry} {
           allow read: if request.auth != null;
           allow write: if false; // Backend only
         }
       }
     }
     ```
   - [ ] Create `internal/server/auth/access_control.go`
     - `func GetUserRole(ctx, uid, workspace) (Role, error)`
     - `func HasApprovalPermission(ctx, user, workspace) (bool, error)`
     - Cache results with TTL

5. **Configuration Loading** (~2 hours)
   - [ ] Update `internal/server/config.go` with auth settings
   - [ ] Load Firebase credentials from environment
   - [ ] Support local Firebase emulator for dev

**Tests:**
- `internal/server/auth/auth_test.go` - JWT verification, caching
- `internal/server/auth/firestore_test.go` - Role lookup, access control
- `internal/server/handlers_auth_test.go` - Middleware behavior

### Phase 2: Frontend Authentication (2 days)

**Deliverables:**
- Login page with Firebase UI
- Protected routes
- Token management
- User profile display

**Tasks:**

1. **Firebase Client Setup** (~1 hour)
   - [ ] Install `firebase` and `react-firebase-hooks`
   - [ ] Create `src/config/firebase.ts`
     - Initialize Firebase app
     - Export auth and firestore instances
   - [ ] Create `.env.local` template with Firebase config

2. **Login Page** (~3 hours)
   - [ ] Create `src/pages/LoginPage.tsx`
     - Email/password form
     - Google OAuth button
     - Error messages
     - Loading state
     - Redirect to dashboard on success
   - [ ] Create `src/components/FirebaseAuthUI.tsx`
     - Reusable auth UI component
     - Sign up / sign in toggle

3. **Protected Routes** (~2 hours)
   - [ ] Create `src/components/ProtectedRoute.tsx`
     - Check Firebase auth state
     - Redirect to login if not authenticated
     - Loading spinner while checking
   - [ ] Wrap Dashboard routes with ProtectedRoute
   - [ ] Handle token refresh

4. **Token Management** (~2 hours)
   - [ ] Create `src/utils/auth.ts`
     - `getIdToken()` - Get current user's ID token
     - `onAuthStateChanged()` - Listen for auth changes
     - `logout()` - Sign out user
     - Store token in sessionStorage (not localStorage for security)
   - [ ] Create auth context:
     ```typescript
     interface AuthContextType {
       user: User | null;
       loading: boolean;
       idToken: string | null;
     }
     export const AuthContext = React.createContext<AuthContextType>(...)
     ```

5. **API Client Updates** (`src/utils/api.ts`)** (~2 hours)
   - [ ] Update fetch helper to include Authorization header
     ```typescript
     async function apiCall(endpoint: string, options?: RequestInit) {
       const idToken = await getIdToken();
       const headers = {
         ...options?.headers,
         'Authorization': `Bearer ${idToken}`,
       };
       return fetch(endpoint, { ...options, headers });
     }
     ```
   - [ ] Handle 401 responses (token expired, redirect to login)

6. **User Profile Component** (~1 hour)
   - [ ] Create `src/components/UserProfile.tsx`
     - Display user email
     - Show role (Viewer / Approver)
     - Logout button
     - Place in header/navigation

**Tests:**
- `src/pages/LoginPage.test.tsx` - Form submission, OAuth flow
- `src/components/ProtectedRoute.test.tsx` - Auth state, redirection
- `src/utils/auth.test.ts` - Token management, refresh

### Phase 3: Endpoint Authorization (1.5 days)

**Deliverables:**
- All endpoints require authentication
- Role-based access control enforced
- Audit logging of all approvals

**Tasks:**

1. **Endpoint Authorization** (~3 hours)
   - [ ] Update `handlers_tasks.go`
     - Add `@authorize approver` comments
     - GET /api/tasks - Viewer (filter by workspace)
     - POST /api/tasks/:id/approve - Approver only
     - POST /api/tasks/:id/reject - Approver only
   - [ ] Update `handlers_events.go`
     - WebSocket /ws - Viewer (filter events by workspace)
   - [ ] Create helper functions:
     ```go
     func RequireAuth(next http.Handler) http.Handler
     func RequireApprover(next http.Handler) http.Handler
     func GetUserFromContext(r *http.Request) (*User, error)
     ```

2. **Workspace Filtering** (~2 hours)
   - [ ] Update query builders to filter by user's workspace
   - [ ] Example: `SELECT * FROM tasks WHERE workspace = ?`
     - Add workspace context to all queries
   - [ ] WebSocket event filtering by workspace

3. **Audit Logging** (~2 hours)
   - [ ] Create `internal/server/audit.go`
     - `LogApproval(ctx, user, taskID, action)`
     - Write to Firestore audit_log collection
     - Include timestamp, user email, workspace, action
   - [ ] Call audit logging in approval handlers
   - [ ] Make audit logs queryable from UI (future)

4. **Error Handling** (~1 hour)
   - [ ] Create standard error responses:
     ```go
     {
       "error": "unauthorized",
       "message": "You don't have permission to approve tasks",
       "status": 403
     }
     ```
   - [ ] Update error handlers to return proper HTTP status codes

**Tests:**
- `internal/server/handlers_tasks_test.go` - Endpoint authorization
- `internal/server/audit_test.go` - Audit logging
- Integration tests with different role combinations

### Phase 4: Deployment & Documentation (1 day)

**Deliverables:**
- Cloud Run deployment configuration
- Deployment guide
- Configuration examples

**Tasks:**

1. **Cloud Run Configuration** (~2 hours)
   - [ ] Create `Dockerfile` (if not exists)
   - [ ] Create `cloudbuild.yaml` for CI/CD
   - [ ] Document environment variables:
     ```bash
     FIREBASE_PROJECT_ID=...
     FIREBASE_API_KEY=...
     FIREBASE_AUTH_DOMAIN=...
     GOOGLE_APPLICATION_CREDENTIALS=/var/run/secrets/cloud.google.com/...
     ```
   - [ ] Create Cloud Run service with secrets injection

2. **Documentation** (~2 hours)
   - [ ] Create `docs/DEPLOYMENT_AUTH.md`
     - Firebase setup instructions
     - Environment variables
     - Cloud Run deployment steps
     - Firestore security rules
   - [ ] Create `docs/WORKSPACE_ACCESS_CONTROL.md`
     - How to add users to workspace
     - Role descriptions
     - Examples
   - [ ] Update README.md with auth information

3. **Configuration Examples** (~1 hour)
   - [ ] Create `config.example.yaml` with full auth config
   - [ ] Create `.env.example` for frontend
   - [ ] Document local development setup

---

## Configuration & Deployment

### Local Development Setup

**Firebase Emulator Suite (optional but recommended):**
```bash
# Install Firebase CLI
npm install -g firebase-tools

# Initialize Firebase emulator
firebase init emulator
firebase emulators:start

# Configure in ~/.ailang/config.yaml
auth:
  firebase:
    emulator:
      enabled: true
      auth_host: localhost:9099
      firestore_host: localhost:8080
```

**Environment Variables:**
```bash
# Backend
export FIREBASE_PROJECT_ID=ailang-dev
export FIREBASE_API_KEY=AIzaSyD...
export FIREBASE_AUTH_DOMAIN=ailang-dev.firebaseapp.com
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/serviceAccountKey.json

# Frontend (.env.local)
VITE_FIREBASE_API_KEY=AIzaSyD...
VITE_FIREBASE_PROJECT_ID=ailang-dev
VITE_FIREBASE_AUTH_DOMAIN=ailang-dev.firebaseapp.com
```

### Google Cloud Run Deployment

**Prerequisites:**
- Google Cloud project
- Cloud Build enabled
- Service account with Firestore permissions

**Deployment Steps:**
```bash
# 1. Build Docker image
gcloud builds submit --tag gcr.io/PROJECT_ID/ailang-dashboard

# 2. Deploy to Cloud Run
gcloud run deploy ailang-dashboard \
  --image gcr.io/PROJECT_ID/ailang-dashboard \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars FIREBASE_PROJECT_ID=PROJECT_ID,... \
  --service-account=ailang-dashboard@PROJECT_ID.iam.gserviceaccount.com

# 3. Grant Cloud Run Invoker role to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member=serviceAccount:ailang-dashboard@PROJECT_ID.iam.gserviceaccount.com \
  --role=roles/run.invoker
```

**Firestore Database Creation:**
```bash
# Enable Firestore
gcloud firestore databases create --location=us-central1

# Deploy security rules
firebase deploy --only firestore:rules
```

### Workspace Access Control Configuration

**Adding Users to Workspace:**
```bash
# Via CLI (future command)
ailang auth add-user user@example.com --workspace ailang-core --role approver

# Manual Firestore entry
# Collection: workspaces/ailang-core/access_control
# Document: user@example.com
{
  "email": "user@example.com",
  "role": "approver",
  "created_by": "admin@example.com",
  "created_at": timestamp,
}
```

**Workspace Configuration:**
```yaml
# ~/.ailang/config.yaml
workspace_config:
  workspace_mappings:
    "ailang-core": "workspaces/ailang-core"
    "ailang-examples": "workspaces/ailang-examples"

  # Public workspaces (visible to all authenticated users)
  public_workspaces: []
```

---

## Examples

### Example 1: User Attempting to Access Task Without Permission

**Request:**
```bash
curl -H "Authorization: Bearer <valid-token>" \
     -H "X-Workspace: other-workspace" \
     https://dashboard.example.com/api/tasks
```

**User:** john@example.com (has access to `ailang-core` workspace only)
**Request Workspace:** `other-workspace`

**Response:**
```json
{
  "error": "unauthorized",
  "message": "You don't have access to workspace 'other-workspace'",
  "status": 403
}
```

**Audit Log:**
```
{
  "timestamp": "2026-01-22T10:30:00Z",
  "user_uid": "firebase-uid-123",
  "user_email": "john@example.com",
  "action": "access_denied",
  "reason": "workspace_not_found",
  "workspace": "other-workspace",
  "endpoint": "GET /api/tasks"
}
```

### Example 2: Approver Approving a Task

**Request:**
```bash
curl -X POST \
     -H "Authorization: Bearer <valid-token>" \
     -H "Content-Type: application/json" \
     -d '{"feedback": "Looks good"}' \
     https://dashboard.example.com/api/tasks/task-123/approve
```

**User:** alice@example.com (Approver role in `ailang-core`)
**Task:** task-123 (in `ailang-core` workspace)

**Response:**
```json
{
  "success": true,
  "message": "Task approved",
  "task_id": "task-123",
  "approved_by": "alice@example.com"
}
```

**Audit Log:**
```
{
  "timestamp": "2026-01-22T10:35:00Z",
  "user_uid": "firebase-uid-456",
  "user_email": "alice@example.com",
  "workspace": "ailang-core",
  "action": "approve",
  "task_id": "task-123",
  "details": {
    "feedback": "Looks good"
  }
}
```

### Example 3: Viewer Cannot Approve

**Request:**
```bash
curl -X POST \
     -H "Authorization: Bearer <valid-token>" \
     -H "Content-Type: application/json" \
     https://dashboard.example.com/api/tasks/task-456/approve
```

**User:** bob@example.com (Viewer role in `ailang-core`)

**Response:**
```json
{
  "error": "forbidden",
  "message": "Your role (viewer) doesn't have permission to approve tasks. Required: approver",
  "status": 403
}
```

**Audit Log:**
```
{
  "timestamp": "2026-01-22T10:40:00Z",
  "user_uid": "firebase-uid-789",
  "user_email": "bob@example.com",
  "workspace": "ailang-core",
  "action": "access_denied",
  "reason": "insufficient_role",
  "required_role": "approver",
  "actual_role": "viewer"
}
```

---

## Files to Create/Modify

### Backend Files

**New Files:**
- `internal/server/auth/auth.go` (~200 LOC) - Firebase JWT verification
- `internal/server/auth/firestore.go` (~150 LOC) - Firestore access control
- `internal/server/auth/auth_test.go` (~250 LOC) - Auth tests
- `internal/server/auth/firestore_test.go` (~200 LOC) - Firestore tests
- `internal/server/handlers_auth.go` (~300 LOC) - Auth middleware
- `internal/server/audit.go` (~150 LOC) - Audit logging
- `internal/server/handlers_auth_test.go` (~300 LOC) - Handler tests

**Modified Files:**
- `internal/server/config.go` - Add auth configuration
- `internal/server/handlers_tasks.go` - Add authorization checks
- `internal/server/handlers_events.go` - Filter events by workspace
- `internal/server/server.go` - Register auth middleware
- `Dockerfile` - Add Firebase client library
- `go.mod` - Add Firebase Admin SDK

### Frontend Files

**New Files:**
- `src/pages/LoginPage.tsx` (~300 LOC) - Login UI
- `src/components/FirebaseAuthUI.tsx` (~200 LOC) - Firebase UI component
- `src/components/ProtectedRoute.tsx` (~100 LOC) - Route protection
- `src/components/UserProfile.tsx` (~100 LOC) - User menu
- `src/config/firebase.ts` (~50 LOC) - Firebase initialization
- `src/utils/auth.ts` (~150 LOC) - Auth helpers
- `src/context/AuthContext.tsx` (~100 LOC) - Auth context provider
- `src/pages/LoginPage.test.tsx` (~250 LOC) - Login tests
- `src/components/ProtectedRoute.test.tsx` (~150 LOC) - Route tests
- `src/utils/auth.test.ts` (~200 LOC) - Auth tests
- `.env.example` - Environment variables template

**Modified Files:**
- `src/App.tsx` - Add AuthProvider wrapper
- `src/utils/api.ts` - Add Authorization header
- `src/pages/Dashboard.tsx` - Add auth status display
- `package.json` - Add Firebase dependencies
- `vite.config.ts` - Add environment variable support

### Configuration Files

**New Files:**
- `docs/DEPLOYMENT_AUTH.md` - Authentication setup guide
- `docs/WORKSPACE_ACCESS_CONTROL.md` - Access control guide
- `firestore.rules` - Firestore security rules
- `.env.example` - Frontend environment template
- `config.example.yaml` - Backend configuration example

**Modified Files:**
- `DEPLOYMENT.md` - Add auth section
- `README.md` - Note that auth is now required
- `.github/workflows/deploy.yml` - Add Firebase config

---

## Testing Strategy

### Unit Tests

**Auth Tests:**
- Valid token verification
- Invalid/expired token handling
- Token caching and TTL
- Firestore role lookup
- Role-based access control

**Handler Tests:**
- Unauthenticated requests → 401
- Authenticated but unauthorized → 403
- Approver requests → 200
- Viewer requests to approval endpoints → 403

### Integration Tests

**End-to-End Flows:**
1. User logs in → Frontend stores token → Can access dashboard
2. Approver approves task → Audit log created
3. Viewer tries to approve → 403 error
4. Multi-workspace user → Only sees own workspace tasks

### Security Tests

- [ ] CSRF protection (Django-style token in forms)
- [ ] JWT verification with Firebase public keys
- [ ] Token expiration handling
- [ ] Workspace boundary enforcement
- [ ] Audit log completeness

### Performance Tests

- [ ] JWT verification <10ms (with caching <1ms)
- [ ] Firestore role lookup <50ms (with caching <5ms)
- [ ] Middleware adds <50ms total latency

---

## Success Criteria

- [ ] **Authentication Required**: 401 error for all unauthenticated API requests
- [ ] **Role-Based Access**: All endpoints enforce role permissions
- [ ] **Audit Trail**: All approvals logged with user identity
- [ ] **Multi-Workspace**: Users can only access configured workspaces
- [ ] **Performance**: Auth checks add <50ms latency
- [ ] **UX**: Users can log in with Google in <3 clicks
- [ ] **Documentation**: Deployment guide covers Firebase setup
- [ ] **Tests Passing**: All auth tests pass, no regressions

---

## Related Documents

- [Global Collaboration Hub Design](global-collaboration-hub.md) - Dashboard architecture
- [M-Coordinator Always-On Daemon](m-coordinator-always-on-daemon.md) - Task approval workflow
- [M-WORKSPACE-ACCESS: Multi-Workspace Access Control](../v0_7_1/m-workspace-access-control.md) - Workspace-level permissions with Firestore schema details
- [DEPLOYMENT.md](../../DEPLOYMENT.md) - Deployment procedures

---

## Risks & Mitigations

### Risk: Firebase Outage
**Impact:** Users cannot log in or perform actions
**Mitigation:**
- Graceful degradation: Show offline message
- Cache auth decisions for 5 minutes
- Document manual admin procedure (v0.8.0)

### Risk: Token Leakage
**Impact:** Unauthorized access to user account
**Mitigation:**
- Store tokens in sessionStorage only (not localStorage)
- Use HTTPS only (enforced by Cloud Run)
- Implement token rotation
- Monitor audit logs for suspicious activity

### Risk: Over-Permissioned Service Account
**Impact:** Compromised service account has excessive access
**Mitigation:**
- Create dedicated service account for dashboard
- Grant only Firestore read/write permissions
- Use least privilege principle
- Rotate credentials monthly

### Risk: Workspace Access Misconfiguration
**Impact:** Users access wrong workspace data
**Mitigation:**
- Validate workspace membership on every request
- Extensive integration tests
- Audit logs show all access attempts
- Dashboard shows current workspace clearly

---

## Future Enhancements (v0.8.0+)

- **SAML/OIDC Integration**: Enterprise SSO support
- **Fine-Grained Permissions**: Resource-level access control
- **Team Management UI**: Add/remove users without CLI
- **Permission Delegation**: Approvers delegate authority
- **MFA/2FA**: Two-factor authentication
- **API Keys**: Service account tokens for CI/CD
- **Audit Dashboard**: Query and visualize access logs

---

## Notes

- This design assumes existing Firebase/GCP project setup; v0.7.0 focuses on integration, not infrastructure
- Firestore chosen over Cloud SQL for simplicity and scalability
- Role model kept simple (Viewer/Approver) to unblock deployment; fine-grained permissions can be added later
- Local Firebase Emulator recommended for development but optional
