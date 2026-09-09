# Setting Up Git-Sync for Your Deployment

This guide walks through setting up the git-sync delivery mechanism for your `dimd` instances.

## Prerequisites

- `dimd` running on a host or cluster
- Routes checked into git main branch under `domains/`
- SSH access to the git repository (or HTTPS with credentials)

## Quick Start: Systemd Service

1. **Copy the git-sync script** to your host:
   ```bash
   scp deploy/git-sync.sh your-host:/opt/dim/git-sync.sh
   ssh your-host chmod +x /opt/dim/git-sync.sh
   ```

2. **Create the systemd service file** `/etc/systemd/system/dim-git-sync.service`:
   ```ini
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
   ```

3. **Enable and start the service**:
   ```bash
   sudo systemctl enable dim-git-sync
   sudo systemctl start dim-git-sync
   ```

4. **Verify it's running**:
   ```bash
   sudo systemctl status dim-git-sync
   journalctl -u dim-git-sync -f  # tail logs
   ```

## Verification: Post a Route Change

1. In your local repo, create a test route and push to main:
   ```bash
   git checkout -b test-route
   # Create a simple route in domains/test/test-route.yaml
   git push -u origin test-route
   # Open a PR, get approval, merge to main
   ```

2. On the `dimd` host, check git-sync logs:
   ```bash
   journalctl -u dim-git-sync -f
   # You should see "[...] New commits detected; pulling..."
   # and "[...] Pull successful. Routes in /opt/dim-routes/domains/ are now live..."
   ```

3. Query the engine to verify the new route is live:
   ```bash
   curl -s http://localhost:9090/health/routes | jq '.routes | keys'
   # Should list your new route name
   ```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| "git fetch failed; retrying" | Network/SSH issue | Check SSH keys and git config; verify origin is reachable from host |
| "git pull failed (non-fast-forward)" | Force-push or conflicting local changes | Manually review; if local changes, stash or reset per your policy |
| Routes not updating after 5 minutes | dimd not watching `domains/` for changes | Check dimd logs; ensure hot-reload is enabled in dimd config |
| Old route still serving traffic | Generation draining is in progress (normal) | Wait up to 2 minutes for stragglers; expected behavior per design §14.1 |
