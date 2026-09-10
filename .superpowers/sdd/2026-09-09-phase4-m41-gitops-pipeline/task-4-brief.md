# Task 4: Set up git-sync delivery mechanism

## Files to Create

- Create: `deploy/git-sync.sh`
- Create: `deploy/README.md`
- Create: `docs/self-service/GIT_SYNC_SETUP.md`

## Interfaces

- Consumes: Running `dimd` instance with hot-reload support (existing from Phase 3, §14.1)
- Produces: Mechanism for pulling route changes from git to running instances, with setup documentation

## Steps to Execute

### Step 1: Create deploy/git-sync.sh

Create this file with the following content:

```bash
#!/bin/bash
# Simple git-sync sidecar loop: pull route changes from main branch
# every N seconds and let dimd's hot-reload handle the update.

set -euo pipefail

REPO_PATH="${1:-.}"
BRANCH="${2:-master}"
POLL_INTERVAL="${3:-30}"  # seconds

# Validate inputs
if [ ! -d "$REPO_PATH/.git" ]; then
  echo "Error: $REPO_PATH is not a git repository"
  exit 1
fi

echo "Starting git-sync: polling $REPO_PATH/$BRANCH every ${POLL_INTERVAL}s"
echo "Press Ctrl+C to stop."

while true; do
  cd "$REPO_PATH"
  
  # Fetch updates from origin without fast-forwarding local changes
  git fetch origin "$BRANCH" || {
    echo "Warning: git fetch failed; retrying in ${POLL_INTERVAL}s"
    sleep "$POLL_INTERVAL"
    continue
  }
  
  # Check if we're behind origin
  LOCAL_COMMIT=$(git rev-parse HEAD)
  REMOTE_COMMIT=$(git rev-parse origin/"$BRANCH")
  
  if [ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]; then
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] New commits detected; pulling..."
    
    # Pull changes (routes only, via sparse-checkout if desired, or full pull)
    if git pull origin "$BRANCH" --ff-only 2>&1; then
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Pull successful. Routes in $REPO_PATH/domains/ are now live (via hot-reload)."
    else
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Warning: git pull failed (non-fast-forward or conflict); review manually."
    fi
  else
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] Already up to date."
  fi
  
  sleep "$POLL_INTERVAL"
done
```

Make it executable: `chmod +x deploy/git-sync.sh`

### Step 2: Write deploy/README.md

Create this file with the following content:

```markdown
# Deployment: Git-Sync Mechanism

This directory contains the deployment configuration for the git-sync delivery mechanism, which pulls route changes from the main branch to running `dimd` instances.

## How It Works

1. `git-sync.sh` runs as a sidecar process (or cron job) on the host running `dimd`.
2. Every N seconds (default 30), it polls the origin for new commits on the configured branch.
3. If new commits are found, it performs a fast-forward pull.
4. `dimd`'s existing hot-reload mechanism (§14.1 of the design) detects file changes in `domains/` and reloads routes without a restart.

## Setup Options

### Option A: Sidecar Process (Systemd)

Run git-sync as a systemd service alongside `dimd`:

\`\`\`bash
# Create /etc/systemd/system/dim-git-sync.service
[Unit]
Description=DIM Git-Sync Route Delivery
After=network.target
Wants=dimd.service

[Service]
Type=simple
User=dim
WorkingDirectory=/opt/dim-routes
ExecStart=/opt/dim/deploy/git-sync.sh /opt/dim-routes master 30
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
\`\`\`

Then enable and start:
\`\`\`bash
sudo systemctl enable dim-git-sync
sudo systemctl start dim-git-sync
\`\`\`

### Option B: Kubernetes DaemonSet

Use the provided Kubernetes manifest (TODO: create this as part of M4.1.2 if K8s is a deployment target).

### Option C: Cron Job

Run git-sync on a fixed schedule:
\`\`\`bash
# Every 5 minutes
*/5 * * * * /opt/dim/deploy/git-sync.sh /opt/dim-routes master 5
\`\`\`

## Monitoring

Monitor git-sync's health:
\`\`\`bash
# View logs
journalctl -u dim-git-sync -f

# Check last pull status
git -C /opt/dim-routes log --oneline -1
\`\`\`

## Rollback

Since routes are pulled from git, rollback is a manual \`git revert\` followed by a re-push to main. \`dimd\`'s hot-reload will pick up the reverted config within the poll interval.

## Restrictions

- Only fast-forward pulls are performed; local changes to route files are never merged.
- If a route file is malformed and breaks validation, \`dimd\` will reject it via its schema validation and routing on the new generation will fail; the old generation continues draining. Manual intervention (a corrective commit + push) is required.
\`\`\`

### Step 3: Write docs/self-service/GIT_SYNC_SETUP.md

Create this file with the following content:

\`\`\`markdown
# Setting Up Git-Sync for Your Deployment

This guide walks through setting up the git-sync delivery mechanism for your \`dimd\` instances.

## Prerequisites

- \`dimd\` running on a host or cluster
- Routes checked into git main branch under \`domains/\`
- SSH access to the git repository (or HTTPS with credentials)

## Quick Start: Systemd Service

1. **Copy the git-sync script** to your host:
   \`\`\`bash
   scp deploy/git-sync.sh your-host:/opt/dim/git-sync.sh
   ssh your-host chmod +x /opt/dim/git-sync.sh
   \`\`\`

2. **Create the systemd service file** \`/etc/systemd/system/dim-git-sync.service\`:
   \`\`\`ini
   [Unit]
   Description=DIM Git-Sync Route Delivery
   After=network.target
   Wants=dimd.service

   [Service]
   Type=simple
   User=dim
   WorkingDirectory=/opt/dim-routes
   ExecStart=/opt/dim/git-sync.sh /opt/dim-routes master 30
   Restart=always
   RestartSec=10
   StandardOutput=journal
   StandardError=journal

   [Install]
   WantedBy=multi-user.target
   \`\`\`

3. **Enable and start the service**:
   \`\`\`bash
   sudo systemctl enable dim-git-sync
   sudo systemctl start dim-git-sync
   \`\`\`

4. **Verify it's running**:
   \`\`\`bash
   sudo systemctl status dim-git-sync
   journalctl -u dim-git-sync -f  # tail logs
   \`\`\`

## Verification: Post a Route Change

1. In your local repo, create a test route and push to main:
   \`\`\`bash
   git checkout -b test-route
   # Create a simple route in domains/test/test-route.yaml
   git push -u origin test-route
   # Open a PR, get approval, merge to main
   \`\`\`

2. On the \`dimd\` host, check git-sync logs:
   \`\`\`bash
   journalctl -u dim-git-sync -f
   # You should see "[...] New commits detected; pulling..."
   # and "[...] Pull successful. Routes in /opt/dim-routes/domains/ are now live..."
   \`\`\`

3. Query the engine to verify the new route is live:
   \`\`\`bash
   curl -s http://localhost:9090/health/routes | jq '.routes | keys'
   # Should list your new route name
   \`\`\`

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| "git fetch failed; retrying" | Network/SSH issue | Check SSH keys and git config; verify origin is reachable from host |
| "git pull failed (non-fast-forward)" | Force-push or conflicting local changes | Manually review; if local changes, stash or reset per your policy |
| Routes not updating after 5 minutes | dimd not watching \`domains/\` for changes | Check dimd logs; ensure hot-reload is enabled in dimd config |
| Old route still serving traffic | Generation draining is in progress (normal) | Wait up to 2 minutes for stragglers; expected behavior per design §14.1 |
\`\`\`

All files written. Now verify structure, make git-sync.sh executable, and commit.
