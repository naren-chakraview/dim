#!/bin/bash
# delete-old-artifacts.sh

REPO="naren-chakraview/dim"
DAYS_OLD=4
CUTOFF_DATE=$(date -d "$DAYS_OLD days ago" -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Deleting artifacts older than $DAYS_OLD days ($CUTOFF_DATE)..."

/usr/bin/gh api repos/$REPO/actions/artifacts --paginate \
  --jq ".artifacts[] | select(.created_at < \"$CUTOFF_DATE\") | .id" \
  | while read ARTIFACT_ID; do
    echo "Deleting artifact $ARTIFACT_ID..."
    gh api repos/$REPO/actions/artifacts/$ARTIFACT_ID -X DELETE
  done

echo "Done!"
