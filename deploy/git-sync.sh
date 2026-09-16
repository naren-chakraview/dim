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
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Pull successful. Routes in $REPO_PATH/domains/ are updated."

      # TODO: Send SIGHUP to dimd process to trigger hot-reload of route configurations.
      # This requires dimd to support hot-reload signal handling (currently only in dimctl run).
      # Once dimd hot-reload is implemented, uncomment:
      #   pkill -SIGHUP -f "dimd" || true
      # For now, dimd instances must be restarted manually to pick up route changes.
    else
      echo "[$(date +'%Y-%m-%d %H:%M:%S')] Warning: git pull failed (non-fast-forward or conflict); review manually."
    fi
  else
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] Already up to date."
  fi

  sleep "$POLL_INTERVAL"
done
