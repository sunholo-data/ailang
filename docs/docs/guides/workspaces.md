---
reviewBy: 2026-11-09
---

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

A dashboard bound to anything but loopback (Cloud Run, `--bind 0.0.0.0`) answers no `/api/` read
without its `COORDINATOR_API_KEY` as a bearer token or a Firebase
Viewer/Approver session, so "public" here means visible to every *authenticated* caller. A
loopback-bound local dashboard is exempt.

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

## Dashboard

The dashboard is a headless API (no UI since November 2026), and it no longer serves
`/api/workspaces` or workspace-filtered thread/task lists — those were read only by the removed
UI. Workspace grants are managed with the `ailang workspaces` CLI above; the dashboard's read gate
accepts a Firebase Viewer/Approver session as one of its credentials.

## Security Model

Defense in depth:

1. **API gate**: every dashboard `/api/` read needs a credential on a non-loopback bind
2. **Roles**: a Firebase session must hold Viewer or Approver
3. **Safe Defaults**: Unknown workspaces default to "public" (if exists)

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
