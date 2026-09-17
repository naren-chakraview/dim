# Deployment: Git-Sync Mechanism

This directory contains the deployment configuration for the git-sync delivery mechanism, which pulls route changes from the main branch to running `dimd` instances.

## How It Works

1. `git-sync.sh` runs as a sidecar process (or cron job) on the host running `dimd`.
2. Every N seconds (default 30), it polls the origin for new commits on the configured branch.
3. If new commits are found, it performs a fast-forward pull.
4. **Current limitation**: `dimd` currently requires a manual restart to pick up route changes. Hot-reload support (§14.1 of the design) is implemented in `dimctl run` but not yet in long-running `dimd` instances. This is on the Phase 4 Tier 2 roadmap (T1.5).

## Setup Options

### Option A: Sidecar Process (Systemd)

Run git-sync as a systemd service alongside `dimd`:

```bash
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
```

Then enable and start:
```bash
sudo systemctl enable dim-git-sync
sudo systemctl start dim-git-sync
```

### Option B: Kubernetes DaemonSet

Use the provided Kubernetes manifest (TODO: create this as part of M4.1.2 if K8s is a deployment target).

### Option C: Cron Job

Run git-sync on a fixed schedule:
```bash
# Every 5 minutes
*/5 * * * * /opt/dim/deploy/git-sync.sh /opt/dim-routes master 5
```

## Monitoring

Monitor git-sync's health:
```bash
# View logs
journalctl -u dim-git-sync -f

# Check last pull status
git -C /opt/dim-routes log --oneline -1
```

## Rollback

Since routes are pulled from git, rollback is a manual `git revert` followed by a re-push to main. After git-sync pulls the reverted config, restart `dimd` instances to pick up the change. (Once hot-reload is implemented, this will happen automatically.)

## Restrictions

- Only fast-forward pulls are performed; local changes to route files are never merged.
- If a route file is malformed and breaks validation, `dimd` will reject it via its schema validation and routing on the new generation will fail; the old generation continues draining. Manual intervention (a corrective commit + push) is required.
