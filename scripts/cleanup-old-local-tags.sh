#!/usr/bin/env bash
set -euo pipefail

# cleanup-old-local-tags.sh
# Removes local git tags when:
#   1) The tag does NOT exist on the remote
#   2) The tag's referenced commit is older than 2 weeks
# Supports --dry flag to preview actions without deleting.
#
# Usage:
#   scripts/cleanup-old-local-tags.sh [--dry] [--remote origin]
#
# Examples:
#   scripts/cleanup-old-local-tags.sh --dry
#   scripts/cleanup-old-local-tags.sh --remote upstream
#
# Notes:
# - Assumes the script is run inside a git repository.
# - Checks tags that are reachable locally via `git tag`.

DRY_RUN=false
REMOTE="origin"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry)
      DRY_RUN=true
      shift
      ;;
    --remote)
      if [[ $# -lt 2 ]]; then
        echo "Error: --remote requires a value" >&2
        exit 1
      fi
      REMOTE="$2"
      shift 2
      ;;
    -h|--help)
      sed -n '2,30p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

# Verify we're inside a git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
  echo "Error: This script must be run inside a git repository" >&2
  exit 1
fi

# Verify remote exists
if ! git remote get-url "$REMOTE" > /dev/null 2>&1; then
  echo "Error: Remote '$REMOTE' does not exist" >&2
  exit 1
fi

# Two weeks threshold (in seconds)
TWO_WEEKS=$((14 * 24 * 60 * 60))
NOW_TS=$(date +%s)

# Fetch remote tags list once for efficiency
# Note: Using ls-remote avoids fetching objects; it only lists refs.
REMOTE_TAGS=$(git ls-remote --tags "$REMOTE" | awk '{print $2}' | sed 's#refs/tags/##')

# Convert to newline-separated list and prepare for exact matching
# We'll use grep -Fx to check membership.

# Function: check if tag exists on remote
exists_on_remote() {
  local tag="$1"
  # shellcheck disable=SC2154
  printf '%s\n' "$REMOTE_TAGS" | grep -Fxq "$tag"
}

# Function: get tag commit timestamp
# Works for lightweight tags and annotated tags.
get_tag_commit_ts() {
  local tag="$1"
  local commit
  if ! commit=$(git rev-list -n 1 "$tag" 2>/dev/null); then
    return 1
  fi
  git show -s --format=%ct "$commit"
}

# Collect candidates
to_delete=()

while IFS= read -r tag; do
  [[ -z "$tag" ]] && continue

  # Skip if tag exists on remote
  if exists_on_remote "$tag"; then
    continue
  fi

  # Get commit timestamp and compare age
  ts=$(get_tag_commit_ts "$tag" || echo "")
  if [[ -z "$ts" ]]; then
    # If we cannot resolve the timestamp, be conservative and skip
    continue
  fi

  age=$((NOW_TS - ts))
  if (( age >= TWO_WEEKS )); then
    to_delete+=("$tag")
  fi

done < <(git tag)

# Report and act
if (( ${#to_delete[@]} == 0 )); then
  echo "No local tags eligible for deletion (not on remote and older than 2 weeks)."
  exit 0
fi

if $DRY_RUN; then
  echo "[DRY RUN] The following local tags would be deleted:" 
  for t in "${to_delete[@]}"; do
    echo "  $t"
  done
  exit 0
fi

# Delete tags
for t in "${to_delete[@]}"; do
  echo "Deleting local tag: $t"
  git tag -d "$t" || {
    echo "Warning: failed to delete tag $t" >&2
  }
  # Optional: do NOT push delete to remote, by design we only clean local
  # If you need to delete remote tags in future, use: git push "$REMOTE" :refs/tags/$t
done

echo "Done. Deleted ${#to_delete[@]} tag(s)."