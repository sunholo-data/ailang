# Workspace Access Control

Workspaces provide multi-tenant isolation for the AILANG Dashboard. Users only see data from workspaces they have access to.

## Quick Start

```bash
# Create a workspace
ailang workspaces add --id sunholo-data/ailang --name "AILANG Project" --public

# Grant user access
ailang workspaces grant --id sunholo-data/ailang --email dev@example.com --role Approver

# List accessible workspaces
ailang workspaces list --email m@sunholo.com
```

## Concepts

### Workspace ID

Workspaces use GitHub repository format as their ID:

```
owner/repo
```

Examples:
- `sunholo-data/ailang`
- `MarkEdmondson1234/TwilightGame`

### Roles

| Role | Description |
|------|-------------|
| `Approver` | Can approve/reject coordinator tasks, full access |
| `Viewer` | Read-only access to workspace data |

### Public vs Private

- **Public workspaces**: Visible to all users (authenticated or not)
- **Private workspaces**: Only visible to users with explicit access grants

## CLI Commands

### List Workspaces

```bash
# List all public workspaces
ailang workspaces list

# List workspaces accessible to a specific user
ailang workspaces list --email user@example.com

# Use a specific Firebase project
ailang workspaces list --project my-firebase-project
```

### Create Workspace

```bash
# Create a private workspace
ailang workspaces add --id sunholo-data/ailang --name "AILANG Project"

# Create a public workspace
ailang workspaces add --id sunholo-data/ailang --name "AILANG Project" --public
```

### Show Workspace Details

```bash
ailang workspaces show --id sunholo-data/ailang
```

Output:
```
Workspace: sunholo-data/ailang
Name:      AILANG Project
Public:    true
GitHub:    sunholo-data/ailang
Created:   2026-01-23T10:00:00Z
```

### Grant Access

```bash
# Grant Approver access
ailang workspaces grant --id sunholo-data/ailang --email dev@example.com --role Approver

# Grant Viewer access
ailang workspaces grant --id sunholo-data/ailang --email viewer@example.com --role Viewer
```

### Revoke Access

```bash
ailang workspaces revoke --id sunholo-data/ailang --email dev@example.com
```

### Toggle Public Visibility

```bash
# Make workspace public
ailang workspaces set-public --id sunholo-data/ailang --public

# Make workspace private
ailang workspaces set-public --id sunholo-data/ailang --private
```

## Configuration

### Path Pattern Mapping

Map local file paths to workspace IDs using patterns in `~/.ailang/config.yaml`:

```yaml
workspaces:
  default_workspace: "public"
  mappings:
    - pattern: "**/dev/sunholo/ailang"
      workspace: "sunholo-data/ailang"
    - pattern: "**/dev/TwilightGame"
      workspace: "MarkEdmondson1234/TwilightGame"
```

Pattern syntax:
- `*` - Matches exactly one path component
- `**` - Matches zero or more path components

### Firebase Project

Set the Firebase project via:

1. CLI flag: `--project my-project`
2. Environment variable: `AILANG_FIREBASE_PROJECT=my-project`
3. Config file: `~/.ailang/config.yaml`

```yaml
firebase:
  project_id: my-firebase-project
```

## Firestore Schema

### Workspaces Collection

```
workspaces/{workspace_id}
├── id: string
├── name: string
├── github_repo: string
├── is_public: boolean
├── created_at: timestamp
└── created_by: string
```

### Access Control Subcollection

```
workspace_access/{workspace_id}/users/{email}
├── email: string
├── workspace_id: string
├── role: string ("Viewer" | "Approver")
├── granted_at: timestamp
└── granted_by: string
```

## API Endpoints

### List Accessible Workspaces

```
GET /api/workspaces
```

Returns workspaces the authenticated user can access:

```json
[
  {
    "id": "sunholo-data/ailang",
    "name": "AILANG Project",
    "role": "Approver",
    "is_public": true
  }
]
```

### Workspace Filtering

All dashboard endpoints support workspace filtering via query parameter:

```
GET /api/threads?workspace=sunholo-data/ailang
GET /api/tasks?workspace=sunholo-data/ailang
```

## Dashboard Integration

The Collaboration Hub dashboard automatically filters data by workspace access:

1. **Unauthenticated users**: See only public workspace data
2. **Authenticated users**: See public + granted workspace data
3. **Workspace selector**: Shows role badges (Approver/Viewer)

## Security Model

Defense in depth:

1. **Frontend**: Workspace selector only shows accessible workspaces
2. **API Middleware**: Validates workspace access on every request
3. **Query Filtering**: Database queries scoped to accessible workspaces
4. **Safe Defaults**: Unknown workspaces default to "public" (if exists)

## Troubleshooting

### "Firebase project not configured"

Set the project via environment variable or config:

```bash
export AILANG_FIREBASE_PROJECT=my-project
```

### User Can't See Workspace

1. Check if workspace exists: `ailang workspaces show --id workspace-id`
2. Check user access: `ailang workspaces list --email user@example.com`
3. Grant access: `ailang workspaces grant --id workspace-id --email user@example.com --role Viewer`

### Workspace Not Appearing in Dashboard

1. Ensure workspace is public OR user has explicit access
2. Check browser console for API errors
3. Verify Firebase authentication is working
