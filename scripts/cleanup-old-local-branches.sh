#!/usr/bin/env bash
set -euo pipefail

# cleanup-old-local-branches.sh
# Removes local git branches when:
#   1) The branch does NOT exist on the remote (no upstream or remote ref missing)
#   2) The branch's latest commit is older than 2 weeks
# Supports --dry flag to preview actions without deleting.
#
# Usage:
#   scripts/cleanup-old-local-branches.sh [--dry] [--remote origin]
#
# Examples:
#   scripts/cleanup-old-local-branches.sh --dry
#   scripts/cleanup-old-local-branches.sh --remote upstream
#
# Notes:
# - Assumes the script is run inside a git repository.
# - Skips the current branch to avoid accidental deletion.
# - Only affects local branches; does NOT delete remote branches.

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
      sed -n '2,40p' "$0"
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

# Build a set of remote branch names for quick membership checks
# Using ls-remote to avoid fetching objects
REMOTE_BRANCHES=$(git ls-remote --heads "$REMOTE" | awk '{print $2}' | sed 's#refs/heads/##')

exists_on_remote_branch() {
  local branch="$1"
  printf '%s\n' "$REMOTE_BRANCHES" | grep -Fxq "$branch"
}

branch_last_commit_ts() {
  local branch="$1"
  # Get commit timestamp via rev-list
  local commit
  if ! commit=$(git rev-list -n 1 "$branch" 2>/dev/null); then
    return 1
  fi
  git show -s --format=%ct "$commit"
}

current_branch=$(git rev-parse --abbrev-ref HEAD)

# Collect candidates
mapfile -t local_branches < <(git for-each-ref --format='%(refname:short)' refs/heads/)

# Initialize deletion list (compat with set -u)
to_delete=()

for br in "${local_branches[@]}"; do
  [[ -z "$br" ]] && continue
  # Skip current branch
  if [[ "$br" == "$current_branch" ]]; then
    continue
  fi

  # Determine if branch exists on remote
  if exists_on_remote_branch "$br"; then
    # If the branch exists remotely, skip (requirement: must NOT exist on remote)
    continue
  fi

  # Get last commit timestamp
  ts=$(branch_last_commit_ts "$br" || echo "")
  if [[ -z "$ts" ]]; then
    # Unresolvable; be conservative and skip
    continue
  fi

  age=$((NOW_TS - ts))
  if (( age >= TWO_WEEKS )); then
    to_delete+=("$br")
  fi
done

if (( ${#to_delete[@]} == 0 )); then
  echo "No local branches eligible for deletion (not on remote and older than 2 weeks)."
  exit 0
fi

if $DRY_RUN; then
  echo "[DRY RUN] The following local branches would be deleted:" 
  for b in "${to_delete[@]}"; do
    echo "  $b"
  done
  exit 0
fi

# Delete branches safely; -d refuses to delete unmerged branches
for b in "${to_delete[@]}"; do
  echo "Deleting local branch: $b"
  # Prefer -d (safe) but if it fails because it's unmerged yet old, use -D to force as per cleanup intent.
  if ! git branch -d "$b"; then
    git branch -D "$b" || echo "Warning: failed to delete branch $b" >&2
  fi
done

echo "Done. Deleted ${#to_delete[@]} branch(es)."